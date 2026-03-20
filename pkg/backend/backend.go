package backend

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"
)

type Backend struct {
	URL           *url.URL
	Alive         bool
	mux           sync.RWMutex
	ReverseProxy  *httputil.ReverseProxy
	ServerID      string
	Weight        int
	CurrentWeight int
}

func NewBackend(serverID, target string, weight int) (*Backend, error) {
	parsedURL, err := url.Parse(target)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	if weight <= 0 {
		weight = 1
	}

	b := &Backend{
		URL:      parsedURL,
		Alive:    true,
		ServerID: serverID,
		Weight:   weight,
	}

	b.ReverseProxy = httputil.NewSingleHostReverseProxy(b.URL)

	return b, nil
}

func (b *Backend) SetAlive(alive bool) {
	b.mux.Lock()
	defer b.mux.Unlock()
	b.Alive = alive
}

func (b *Backend) IsAlive() bool {
	b.mux.RLock()
	defer b.mux.RUnlock()
	return b.Alive
}

func (b *Backend) HealthCheck(clientTimeout time.Duration) error {
	client := http.Client{
		Timeout: clientTimeout,
	}

	resp, err := client.Get(b.URL.String())
	if err != nil {
		b.SetAlive(false)
		return fmt.Errorf("health check failed for %s: %w", b.ServerID, err)
	}
	defer resp.Body.Close()

	alive := resp.StatusCode >= 200 && resp.StatusCode < 500
	b.SetAlive(alive)

	if !alive {
		return fmt.Errorf("backend %s returned status %d", b.ServerID, resp.StatusCode)
	}

	return nil
}
