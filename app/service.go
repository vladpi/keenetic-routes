package app

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/vladpi/keenetic-routes/config"
	"github.com/vladpi/keenetic-routes/keenetic"
	"github.com/vladpi/keenetic-routes/routes"

	"golang.org/x/term"
)

// RoutesClient is a small interface for route operations used by the app layer.
type RoutesClient interface {
	GetRoutes() ([]routes.Route, error)
	AddRoutes([]routes.Route) error
	DeleteAllRoutes() error
}

// Service implements core app operations.
type Service struct {
	newClient func(*config.Config) (RoutesClient, error)
	in        io.Reader
	out       io.Writer
}

// NewService creates a service with default IO and client factory.
func NewService() *Service {
	return NewServiceWithClientFactory(nil, nil, nil)
}

// NewServiceWithClientFactory allows injecting client factory and IO for tests.
func NewServiceWithClientFactory(factory func(*config.Config) (RoutesClient, error), in io.Reader, out io.Writer) *Service {
	if factory == nil {
		factory = defaultClientFactory
	}
	if in == nil {
		in = os.Stdin
	}
	if out == nil {
		out = os.Stdout
	}
	return &Service{newClient: factory, in: in, out: out}
}

func defaultClientFactory(cfg *config.Config) (RoutesClient, error) {
	baseURL := "http://" + cfg.Host
	client, err := keenetic.NewClient(baseURL, cfg.User, cfg.Password)
	if err != nil {
		return nil, err
	}
	return &keeneticAdapter{client: client}, nil
}

type keeneticAdapter struct {
	client *keenetic.Client
}

func (k *keeneticAdapter) GetRoutes() ([]routes.Route, error) {
	return k.client.GetDomainRoutes()
}

func (k *keeneticAdapter) AddRoutes(entries []routes.Route) error {
	return k.client.AddRoutes(entries)
}

func (k *keeneticAdapter) DeleteAllRoutes() error {
	return k.client.DeleteAllRoutes()
}

// Upload parses a YAML file and uploads static routes to the router.
func (s *Service) Upload(file string, cfg *config.Config) error {
	if file == "" {
		return fmt.Errorf("file path is required")
	}
	if _, err := os.Stat(file); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("routes file not found: %s", file)
		}
		return fmt.Errorf("stat routes file: %w", err)
	}

	client, err := s.newClient(cfg)
	if err != nil {
		return err
	}

	rf, err := routes.LoadYAML(file)
	if err != nil {
		return fmt.Errorf("load YAML: %w", err)
	}
	entries, err := routes.FlattenToEntries(rf)
	if err != nil {
		return fmt.Errorf("parse routes: %w", err)
	}
	if len(entries) == 0 {
		fmt.Fprintln(s.out, "No entries to upload.")
		return nil
	}

	if err := client.AddRoutes(entries); err != nil {
		return fmt.Errorf("add routes: %w", err)
	}
	fmt.Fprintf(s.out, "Uploaded %d static routes and saved config.\n", len(entries))
	return nil
}

// ResolveDomains resolves route group domains and merges IPv4 results into hosts.
func (s *Service) ResolveDomains(file string) error {
	if file == "" {
		return fmt.Errorf("file path is required")
	}
	if _, err := os.Stat(file); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("routes file not found: %s", file)
		}
		return fmt.Errorf("stat routes file: %w", err)
	}

	rf, err := routes.LoadYAML(file)
	if err != nil {
		return fmt.Errorf("load YAML: %w", err)
	}
	summary, err := routes.ResolveDomains(rf)
	if err != nil {
		return err
	}
	if summary.Groups == 0 {
		fmt.Fprintln(s.out, "No domains to resolve.")
		return nil
	}
	if err := routes.SaveYAML(file, rf); err != nil {
		return fmt.Errorf("save YAML: %w", err)
	}
	fmt.Fprintf(s.out, "Resolved %d domains in %d groups, added %d IPs.\n", summary.Domains, summary.Groups, summary.IPsAdded)
	return nil
}

// Backup downloads routes and saves them to a YAML file.
func (s *Service) Backup(output string, cfg *config.Config) error {
	if output == "" {
		return fmt.Errorf("output path is required")
	}

	client, err := s.newClient(cfg)
	if err != nil {
		return err
	}

	routesList, err := client.GetRoutes()
	if err != nil {
		return fmt.Errorf("get routes: %w", err)
	}

	rf := routes.ToYAML(routesList)
	if err := routes.SaveYAML(output, rf); err != nil {
		return fmt.Errorf("backup: %w", err)
	}
	n := 0
	for _, g := range rf.Routes {
		n += len(g.Hosts)
	}
	fmt.Fprintf(s.out, "Backed up %d routes to %s\n", n, output)
	return nil
}

// Clear removes all static routes from the router and saves config.
func (s *Service) Clear(cfg *config.Config) error {
	client, err := s.newClient(cfg)
	if err != nil {
		return err
	}

	if err := client.DeleteAllRoutes(); err != nil {
		return fmt.Errorf("clear routes: %w", err)
	}
	fmt.Fprintln(s.out, "Static routes cleared and config saved.")
	return nil
}

// InitConfig interactively creates configuration file.
func (s *Service) InitConfig() error {
	scanner := bufio.NewScanner(s.in)
	var cfg config.Config

	fmt.Fprint(s.out, "Enter Keenetic router host (e.g., 192.168.100.1:280): ")
	if scanner.Scan() {
		cfg.Host = strings.TrimSpace(scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read input: %w", err)
	}

	fmt.Fprint(s.out, "Enter username: ")
	if scanner.Scan() {
		cfg.User = strings.TrimSpace(scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read input: %w", err)
	}

	fmt.Fprint(s.out, "Enter password: ")
	if f, ok := s.in.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		passBytes, err := term.ReadPassword(int(f.Fd()))
		if err != nil {
			return fmt.Errorf("read password: %w", err)
		}
		fmt.Fprintln(s.out)
		cfg.Password = strings.TrimSpace(string(passBytes))
	} else {
		if scanner.Scan() {
			cfg.Password = strings.TrimSpace(scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			return fmt.Errorf("read input: %w", err)
		}
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if err := config.SaveConfig(&cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Fprintf(s.out, "Configuration saved to %s\n", config.GetConfigFilePath())
	return nil
}
