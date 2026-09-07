package api_gateway

import (
	// "context"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

type ProxyRequest struct {
	time time.Time
}

type Service struct {
	URL string
	Alive bool
	Proxy *httputil.ReverseProxy
}

type Gateway struct {
	Cracker *Service
	Subscription *Service
}

func NewService(checkURL string) *Service {
	parsedURL, err := url.Parse(checkURL)
	if err != nil {
		panic("api_gateway: Wrong url given, cannot parse")
	}
	return &Service{
		URL: checkURL,
		Alive: false,
		Proxy: httputil.NewSingleHostReverseProxy(parsedURL),
	}
} 

func NewGateway(crackerURL, subsURL string) *Gateway {
	cracker := NewService(crackerURL)
	subs := NewService(subsURL)
	return &Gateway{
		Cracker: cracker,
		Subscription: subs,
	}
}

func HealthCheck(service *Service) {
	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(service.URL + "/health")
	if err != nil {
		service.Alive = false
		return
	}
	if resp.StatusCode != http.StatusOK {
		service.Alive = false
		return
	}
	service.Alive = true
}

func (g *Gateway) StartCheck() {
	ticker := time.NewTicker(10 * time.Second)
	go func() {
		for {
			HealthCheck(g.Cracker)
			HealthCheck(g.Subscription)
			<-ticker.C
		}
	}()
}
