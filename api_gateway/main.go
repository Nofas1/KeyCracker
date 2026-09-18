package api_gateway

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"sync/atomic"
	"time"

	"key_cracker/api_gateway/config"
	// mw "key_cracker/api_gateway/middleware"
)

type Service struct {
	Name string
	URL string
	Alive atomic.Bool
	Proxy *httputil.ReverseProxy
}

type Gateway struct {
	services map[string]*Service
	client *http.Client
	logger *slog.Logger
}

func NewService(name, checkURL string) *Service {
	parsedURL, err := url.Parse(checkURL)
	if err != nil {
		panic("api_gateway: Wrong url given, cannot parse")
	}
	return &Service{
		Name: name,
		URL: checkURL,
		Proxy: httputil.NewSingleHostReverseProxy(parsedURL),
	}
} 

// func NewGateway(crackerURL, subsURL string) *Gateway {
// 	cracker := NewService(crackerURL)
// 	subs := NewService(subsURL)
// 	return &Gateway{
// 		Cracker: cracker,
// 		Subscription: subs,
// 	}
// }

func NewGateway(client *http.Client, cfg *config.Config, logger *slog.Logger) *Gateway {
	// client := &http.Client{Timeout: 5 * time.Second}
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
		select {
		case <-ticker.C:
			g.checkAll()

		case <-ctx.Done():
			return
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
			g.logger.ErrorContext(r.Context(), "service unavailable", slog.String("service", name))
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

func main() {
	var cfg_path string
	flag.StringVar(&cfg_path, "config", "config.yaml", "config path")
	flag.Parse()

	client := &http.Client{}
	cfg, err := config.LoadConfig(cfg_path)
	if err != nil {
		panic(err)
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	gateway := NewGateway(client, cfg, logger)

	// authService, err := mw.New(time.Hour, logger, gateway.rep)
	if err != nil {
		panic(err)
	}

	logger.Info("Listening on", "Adr", cfg.Gateway.Address, "Port", cfg.Gateway.Port)
	err = http.ListenAndServe(fmt.Sprintf("%s:%d", cfg.Gateway.Address, cfg.Gateway.Port), nil)
	fmt.Println(err)
}
