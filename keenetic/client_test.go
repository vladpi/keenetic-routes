package keenetic

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/vladpi/keenetic-routes/routes"
)

func strPtr(v string) *Stringish {
	s := Stringish(v)
	return &s
}

func TestClientGetRoutesAuthFlow(t *testing.T) {
	login := "user"
	password := "pass"
	realm := "realm"
	challenge := "challenge"

	var authChecked bool
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth":
			if r.Method == http.MethodGet {
				w.Header().Set("X-NDM-Realm", realm)
				w.Header().Set("X-NDM-Challenge", challenge)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			if r.Method == http.MethodPost {
				var body map[string]string
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				md5Sum := md5.Sum([]byte(login + ":" + realm + ":" + password))
				md5Hex := hex.EncodeToString(md5Sum[:])
				shaSum := sha256.Sum256([]byte(challenge + md5Hex))
				shaHex := hex.EncodeToString(shaSum[:])
				if body["login"] != login || body["password"] != shaHex {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				mu.Lock()
				authChecked = true
				mu.Unlock()
				w.WriteHeader(http.StatusOK)
				return
			}
			w.WriteHeader(http.StatusMethodNotAllowed)
		case "/rci/ip/route":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]Route{
				{Host: strPtr("8.8.8.8"), Comment: strPtr("test")},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, err := NewClientWithHTTPClient(server.URL, login, password, &http.Client{})
	if err != nil {
		t.Fatalf("NewClientWithHTTPClient: %v", err)
	}
	routes, err := client.GetDomainRoutes()
	if err != nil {
		t.Fatalf("GetDomainRoutes: %v", err)
	}
	if len(routes) != 1 || routes[0].Host != "8.8.8.8" {
		t.Fatalf("unexpected routes: %+v", routes)
	}
	mu.Lock()
	checked := authChecked
	mu.Unlock()
	if !checked {
		t.Fatalf("auth POST did not validate credentials")
	}
}

func TestClientAddRoutesBatching(t *testing.T) {
	var mu sync.Mutex
	var payloadLens []int
	var payloadErr error

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth":
			w.WriteHeader(http.StatusOK)
		case "/rci/", "/rci":
			if r.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			var payload []map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				mu.Lock()
				payloadErr = err
				mu.Unlock()
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			mu.Lock()
			payloadLens = append(payloadLens, len(payload))
			if len(payload) > 0 {
				last := payload[len(payload)-1]
				sys, ok := last["system"].(map[string]any)
				if !ok {
					payloadErr = fmt.Errorf("missing system payload")
				} else if cfg, ok := sys["configuration"].(map[string]any); !ok || cfg["save"] != true {
					payloadErr = fmt.Errorf("missing save config payload")
				}
			}
			mu.Unlock()
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, err := NewClientWithHTTPClient(server.URL, "user", "pass", &http.Client{})
	if err != nil {
		t.Fatalf("NewClientWithHTTPClient: %v", err)
	}
	entries := make([]routes.Route, routeBatchSize+5)
	for i := range entries {
		entries[i] = routes.Route{
			Host:    fmt.Sprintf("10.0.0.%d", i+1),
			Gateway: "10.0.0.1",
		}
	}
	if err := client.AddRoutes(entries); err != nil {
		t.Fatalf("AddRoutes: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if payloadErr != nil {
		t.Fatalf("payload error: %v", payloadErr)
	}
	if len(payloadLens) != 2 {
		t.Fatalf("expected 2 batches, got %d", len(payloadLens))
	}
	if payloadLens[0] != routeBatchSize+1 {
		t.Fatalf("first batch size: got %d, want %d", payloadLens[0], routeBatchSize+1)
	}
	if payloadLens[1] != 6 {
		t.Fatalf("second batch size: got %d, want %d", payloadLens[1], 6)
	}
}

func TestClientDeleteAllRoutesEmpty(t *testing.T) {
	var mu sync.Mutex
	var deletePayloadLen int
	var payloadErr error

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth":
			w.WriteHeader(http.StatusOK)
		case "/rci/ip/route":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]Route{})
		case "/rci/", "/rci":
			var payload []map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				mu.Lock()
				payloadErr = err
				mu.Unlock()
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			mu.Lock()
			deletePayloadLen = len(payload)
			if len(payload) != 1 {
				payloadErr = fmt.Errorf("expected single save payload, got %d", len(payload))
			} else if sys, ok := payload[0]["system"].(map[string]any); !ok {
				payloadErr = fmt.Errorf("missing system payload")
			} else if cfg, ok := sys["configuration"].(map[string]any); !ok || cfg["save"] != true {
				payloadErr = fmt.Errorf("missing save config payload")
			}
			mu.Unlock()
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, err := NewClientWithHTTPClient(server.URL, "user", "pass", &http.Client{})
	if err != nil {
		t.Fatalf("NewClientWithHTTPClient: %v", err)
	}
	if err := client.DeleteAllRoutes(); err != nil {
		t.Fatalf("DeleteAllRoutes: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if payloadErr != nil {
		t.Fatalf("payload error: %v", payloadErr)
	}
	if deletePayloadLen != 1 {
		t.Fatalf("unexpected payload length: %d", deletePayloadLen)
	}
}
