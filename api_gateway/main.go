package api_gateway

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"key_cracker/api_gateway/config"
)

type Service struct {
	Name  string
	URL   string
	Alive atomic.Bool
	Proxy *httputil.ReverseProxy
}

type Gateway struct {
	services map[string]*Service
	client   *http.Client
	logger   *log.Logger
}

func NewService(name, checkURL string) *Service {
	parsedURL, err := url.Parse(checkURL)
	if err != nil {
		panic("api_gateway: Wrong url given, cannot parse")
	}
	return &Service{
		Name:  name,
		URL:   checkURL,
		Proxy: httputil.NewSingleHostReverseProxy(parsedURL),
	}
}

func NewGateway(client *http.Client, cfg *config.Config, logger *log.Logger) *Gateway {
	services := make(map[string]*Service, len(cfg.Services))
	for name, svcCfg := range cfg.Services {
		svc := NewService(name, svcCfg.URL)
		services[name] = svc
	}

	return &Gateway{services: services, client: client, logger: logger}
}

func (g *Gateway) HealthCheck(service *Service) {
	resp, err := g.client.Get(service.URL + "/health")
	if err != nil {
		service.Alive.Store(false)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		service.Alive.Store(false)
		return
	}
	service.Alive.Store(true)
}

func (g *Gateway) StartHealthCheck(ctx context.Context) {
	g.checkAll()
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				g.checkAll()
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (g *Gateway) checkAll() {
	for _, svc := range g.services {
		g.HealthCheck(svc)
	}
}

func (g *Gateway) ProxyTo(name string) http.Handler {
	svc, ok := g.services[name]
	if !ok {
		panic(fmt.Sprintf("gateway: unknown service %q", name))
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !svc.Alive.Load() {
			g.logger.Printf("ERROR: service unavailable: %s", name)
			http.Error(w, fmt.Sprintf("%s service unavailable", name), http.StatusServiceUnavailable)
			return
		}
		svc.Proxy.ServeHTTP(w, r)
	})
}

func (g *Gateway) Health(w http.ResponseWriter, r *http.Request) {
	status := "ok"
	checks := make(map[string]string, len(g.services))

	for name, svc := range g.services {
		if svc.Alive.Load() {
			checks[name] = "ok"
		} else {
			checks[name] = "unreachable"
			status = "error"
		}
	}

	code := http.StatusOK
	if status != "ok" {
		code = http.StatusServiceUnavailable
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]any{"status": status, "checks": checks})
}

// AuthMiddleware validates JWT tokens from the middleware service
func (g *Gateway) AuthMiddleware(middlewareURL string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Authorization header required", http.StatusUnauthorized)
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				http.Error(w, "Invalid authorization format", http.StatusUnauthorized)
				return
			}

			tokenString := parts[1]

			// Verify token with middleware service
			verifyURL := middlewareURL + "/verify"
			req, err := http.NewRequest("POST", verifyURL, strings.NewReader(tokenString))
			if err != nil {
				g.logger.Printf("ERROR: failed to create verify request: %v", err)
				http.Error(w, "Internal error", http.StatusInternalServerError)
				return
			}
			req.Header.Set("Content-Type", "text/plain")

			resp, err := g.client.Do(req)
			if err != nil {
				g.logger.Printf("ERROR: token verification failed: %v", err)
				http.Error(w, "Token verification failed", http.StatusUnauthorized)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func main() {
	var cfg_path string
	flag.StringVar(&cfg_path, "config", "config.yaml", "config path")
	flag.Parse()

	client := &http.Client{Timeout: 5 * time.Second}
	cfg, err := config.LoadConfig(cfg_path)
	if err != nil {
		panic(err)
	}
	logger := log.New(os.Stdout, "[API_GATEWAY] ", log.LstdFlags|log.Lmicroseconds)
	gateway := NewGateway(client, cfg, logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	gateway.StartHealthCheck(ctx)

	mux := http.NewServeMux()

	// Health endpoint
	mux.HandleFunc("/health", gateway.Health)

	// Route requests to appropriate services
	mux.Handle("/cracker/", http.StripPrefix("/cracker", gateway.ProxyTo("cracker")))
	mux.Handle("/middleware/", http.StripPrefix("/middleware", gateway.ProxyTo("middleware")))
	mux.Handle("/subscription/", http.StripPrefix("/subscription", gateway.ProxyTo("subscription")))

	// Protected routes example (requires auth)
	// mux.Handle("/api/protected", gateway.AuthMiddleware(cfg.Services["middleware"].URL)(http.HandlerFunc(protectedHandler)))

	logger.Printf("API Gateway listening on %s:%d", cfg.Gateway.Address, cfg.Gateway.Port)
	err = http.ListenAndServe(fmt.Sprintf("%s:%d", cfg.Gateway.Address, cfg.Gateway.Port), mux)
	fmt.Println(err)
}