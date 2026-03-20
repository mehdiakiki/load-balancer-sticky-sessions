package loadbalancer

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/medvih/loadbalancer-sticky-sessions/pkg/backend"
)

const (
	StickySessionCookie = "LB_SESSION"
)

type StickySession struct {
	BackendID string
	ExpiresAt time.Time
}

type LoadBalancer struct {
	Backends      []*backend.Backend
	Current       uint64
	mux           sync.RWMutex
	StickySession map[string]*StickySession
	sessionTTL    time.Duration
	Algorithm     string
}

func NewLoadBalancer(sessionTTL time.Duration, algorithm string) *LoadBalancer {
	if algorithm == "" {
		algorithm = "round-robin"
	}
	return &LoadBalancer{
		Backends:      make([]*backend.Backend, 0),
		sessionTTL:    sessionTTL,
		StickySession: make(map[string]*StickySession),
		Algorithm:     algorithm,
	}
}

func (lb *LoadBalancer) AddBackend(b *backend.Backend) {
	lb.mux.Lock()
	defer lb.mux.Unlock()
	lb.Backends = append(lb.Backends, b)
}

func (lb *LoadBalancer) RemoveBackend(serverID string) {
	lb.mux.Lock()
	defer lb.mux.Unlock()
	for i, b := range lb.Backends {
		if b.ServerID == serverID {
			lb.Backends = append(lb.Backends[:i], lb.Backends[i+1:]...)
			return
		}
	}
}

func (lb *LoadBalancer) getNextBackend() *backend.Backend {
	lb.mux.Lock()
	defer lb.mux.Unlock()

	if len(lb.Backends) == 0 {
		return nil
	}

	if lb.Algorithm == "weighted-round-robin" {
		return lb.getWeightedRoundRobinBackend()
	}

	next := int(lb.Current+1) % len(lb.Backends)
	lb.Current++

	return lb.Backends[next]
}

func (lb *LoadBalancer) getWeightedRoundRobinBackend() *backend.Backend {
	var selectedBackend *backend.Backend
	var totalWeight int

	for _, b := range lb.Backends {
		if !b.IsAlive() {
			continue
		}

		b.CurrentWeight += b.Weight
		totalWeight += b.Weight

		if selectedBackend == nil || b.CurrentWeight > selectedBackend.CurrentWeight {
			selectedBackend = b
		}
	}

	if selectedBackend != nil {
		selectedBackend.CurrentWeight -= totalWeight
	}

	return selectedBackend
}

func (lb *LoadBalancer) getBackendByID(serverID string) *backend.Backend {
	lb.mux.RLock()
	defer lb.mux.RUnlock()

	for _, b := range lb.Backends {
		if b.ServerID == serverID {
			return b
		}
	}
	return nil
}

func (lb *LoadBalancer) generateSessionID(r *http.Request) string {
	data := fmt.Sprintf("%s%s%d", r.RemoteAddr, r.UserAgent(), time.Now().UnixNano())
	hash := sha256.Sum256([]byte(data))
	return base64.URLEncoding.EncodeToString(hash[:])
}

func (lb *LoadBalancer) getStickyBackend(r *http.Request) *backend.Backend {
	cookie, err := r.Cookie(StickySessionCookie)
	if err != nil {
		return nil
	}

	lb.mux.RLock()
	session, exists := lb.StickySession[cookie.Value]
	lb.mux.RUnlock()

	if !exists {
		return nil
	}

	if time.Now().After(session.ExpiresAt) {
		lb.mux.Lock()
		delete(lb.StickySession, cookie.Value)
		lb.mux.Unlock()
		return nil
	}

	backend := lb.getBackendByID(session.BackendID)
	if backend != nil && backend.IsAlive() {
		return backend
	}

	return nil
}

func (lb *LoadBalancer) setStickySession(w http.ResponseWriter, r *http.Request, serverID string) {
	sessionID := lb.generateSessionID(r)

	lb.mux.Lock()
	lb.StickySession[sessionID] = &StickySession{
		BackendID: serverID,
		ExpiresAt: time.Now().Add(lb.sessionTTL),
	}
	lb.mux.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name:     StickySessionCookie,
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   int(lb.sessionTTL.Seconds()),
	})
}

func (lb *LoadBalancer) HealthCheck(interval, timeout time.Duration) {
	for _, b := range lb.Backends {
		go func(backend *backend.Backend) {
			for {
				_ = backend.HealthCheck(timeout)
				time.Sleep(interval)
			}
		}(b)
	}
}

func (lb *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	targetBackend := lb.getStickyBackend(r)

	if targetBackend == nil {
		targetBackend = lb.getNextBackend()
		if targetBackend == nil {
			http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
			return
		}

		lb.setStickySession(w, r, targetBackend.ServerID)
	}

	targetBackend.ReverseProxy.ServeHTTP(w, r)
}
