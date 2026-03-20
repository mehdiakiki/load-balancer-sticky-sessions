package loadbalancer

import (
	"net/http"
	"testing"

	"github.com/medvih/loadbalancer-sticky-sessions/pkg/backend"
)

func TestNewLoadBalancer(t *testing.T) {
	lb := NewLoadBalancer(300, "round-robin")
	if lb == nil {
		t.Fatal("Expected load balancer to not be nil")
	}
	if len(lb.Backends) != 0 {
		t.Errorf("Expected 0 backends, got %d", len(lb.Backends))
	}
}

func TestAddBackend(t *testing.T) {
	lb := NewLoadBalancer(300, "round-robin")
	backend1 := &backend.Backend{ServerID: "server-1"}
	backend2 := &backend.Backend{ServerID: "server-2"}

	lb.AddBackend(backend1)
	if len(lb.Backends) != 1 {
		t.Errorf("Expected 1 backend, got %d", len(lb.Backends))
	}

	lb.AddBackend(backend2)
	if len(lb.Backends) != 2 {
		t.Errorf("Expected 2 backends, got %d", len(lb.Backends))
	}
}

func TestRemoveBackend(t *testing.T) {
	lb := NewLoadBalancer(300, "round-robin")
	backend1 := &backend.Backend{ServerID: "server-1"}
	backend2 := &backend.Backend{ServerID: "server-2"}

	lb.AddBackend(backend1)
	lb.AddBackend(backend2)

	lb.RemoveBackend("server-1")
	if len(lb.Backends) != 1 {
		t.Errorf("Expected 1 backend after removal, got %d", len(lb.Backends))
	}

	if lb.Backends[0].ServerID != "server-2" {
		t.Errorf("Expected remaining backend to be server-2, got %s", lb.Backends[0].ServerID)
	}
}

func TestGenerateSessionID(t *testing.T) {
	lb := NewLoadBalancer(300, "round-robin")

	req1, _ := http.NewRequest("GET", "/", nil)
	req1.RemoteAddr = "192.168.1.1:1234"
	req1.Header.Set("User-Agent", "test-agent")

	id1 := lb.generateSessionID(req1)
	if id1 == "" {
		t.Error("Expected non-empty session ID")
	}

	req2, _ := http.NewRequest("GET", "/", nil)
	req2.RemoteAddr = "192.168.1.2:1234"
	req2.Header.Set("User-Agent", "test-agent")

	id2 := lb.generateSessionID(req2)

	if id1 == id2 {
		t.Error("Expected different session IDs for different requests")
	}
}
