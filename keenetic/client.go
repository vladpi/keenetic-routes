package keenetic

import (
	"bytes"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

const defaultTimeout = 30 * time.Second

func newCookieJar() (http.CookieJar, error) {
	return cookiejar.New(nil)
}

// Client is an HTTP client for Keenetic NDMS RCI API with session auth.
type Client struct {
	baseURL    string
	login      string
	password   string
	httpClient *http.Client
	authed     bool
}

// NewClient creates a client. baseURL should be "http://host:port" (e.g. "http://192.168.100.1:280").
func NewClient(baseURL, login, password string) (*Client, error) {
	return NewClientWithHTTPClient(baseURL, login, password, nil)
}

// NewClientWithHTTPClient creates a client with a custom http.Client for testing.
func NewClientWithHTTPClient(baseURL, login, password string, httpClient *http.Client) (*Client, error) {
	jar, err := newCookieJar()
	if err != nil {
		return nil, fmt.Errorf("cookie jar: %w", err)
	}
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: defaultTimeout,
			Jar:     jar,
		}
	} else {
		if httpClient.Jar == nil {
			httpClient.Jar = jar
		}
		if httpClient.Timeout == 0 {
			httpClient.Timeout = defaultTimeout
		}
	}
	return &Client{
		baseURL:    strings.TrimSuffix(baseURL, "/"),
		login:      login,
		password:   password,
		httpClient: httpClient,
	}, nil
}

// auth performs NDMS auth: GET auth, on 401 compute MD5(login:realm:password) then SHA256(challenge+md5_hex), POST auth.
func (c *Client) auth() error {
	if c.authed {
		return nil
	}
	getResp, err := c.httpClient.Get(c.baseURL + "/auth")
	if err != nil {
		return fmt.Errorf("auth GET: %w", err)
	}
	defer getResp.Body.Close()

	if getResp.StatusCode == http.StatusOK {
		c.authed = true
		return nil
	}
	if getResp.StatusCode != http.StatusUnauthorized {
		return fmt.Errorf("auth GET: unexpected status %d", getResp.StatusCode)
	}
	realm := getResp.Header.Get("X-NDM-Realm")
	challenge := getResp.Header.Get("X-NDM-Challenge")
	if realm == "" || challenge == "" {
		return fmt.Errorf("auth: missing X-NDM-Realm or X-NDM-Challenge")
	}
	md5Sum := md5.Sum([]byte(c.login + ":" + realm + ":" + c.password))
	md5Hex := hex.EncodeToString(md5Sum[:])
	shaInput := challenge + md5Hex
	shaSum := sha256.Sum256([]byte(shaInput))
	shaHex := hex.EncodeToString(shaSum[:])

	body := map[string]string{"login": c.login, "password": shaHex}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("auth POST: marshal body: %w", err)
	}
	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/auth", bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("auth POST: new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	// Use same client so cookies from GET are sent and new ones from POST are stored
	postResp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("auth POST: %w", err)
	}
	defer postResp.Body.Close()
	if postResp.StatusCode != http.StatusOK {
		return fmt.Errorf("auth POST: status %d", postResp.StatusCode)
	}
	c.authed = true
	return nil
}

// Request performs a request after ensuring auth. GET if body is nil, POST with JSON body otherwise.
func (c *Client) Request(query string, body interface{}) ([]byte, error) {
	if err := c.auth(); err != nil {
		return nil, err
	}
	u, err := url.JoinPath(c.baseURL, query)
	if err != nil {
		return nil, fmt.Errorf("build request URL: %w", err)
	}
	var bodyBytes []byte
	if body != nil {
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
	}

	status, data, err := c.doRequest(u, query, bodyBytes)
	if err != nil {
		return nil, err
	}
	if status == http.StatusUnauthorized {
		c.authed = false
		if err := c.auth(); err != nil {
			return nil, err
		}
		status, data, err = c.doRequest(u, query, bodyBytes)
		if err != nil {
			return nil, err
		}
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("request %s: status %d: %s", query, status, string(data))
	}
	return data, nil
}

func (c *Client) doRequest(u, query string, bodyBytes []byte) (int, []byte, error) {
	var req *http.Request
	var err error
	if bodyBytes == nil {
		req, err = http.NewRequest(http.MethodGet, u, nil)
	} else {
		req, err = http.NewRequest(http.MethodPost, u, bytes.NewReader(bodyBytes))
		if err == nil {
			req.Header.Set("Content-Type", "application/json")
		}
	}
	if err != nil {
		return 0, nil, fmt.Errorf("new request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("request %s: %w", query, err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, err
	}
	return resp.StatusCode, data, nil
}
