package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/medvih/loadbalancer-sticky-sessions/pkg/backend"
	"github.com/medvih/loadbalancer-sticky-sessions/pkg/config"
	"github.com/medvih/loadbalancer-sticky-sessions/pkg/loadbalancer"
	"github.com/medvih/loadbalancer-sticky-sessions/pkg/logging"
	"github.com/medvih/loadbalancer-sticky-sessions/pkg/metrics"
	"github.com/medvih/loadbalancer-sticky-sessions/pkg/ratelimit"
)

func main() {
	configFile := flag.String("config", "configs/config.toml", "Path to configuration file")
	generateConfig := flag.Bool("generate-config", false, "Generate default config file and exit")
	flag.Parse()

	if *generateConfig {
		if err := generateDefaultConfig(); err != nil {
			log.Fatalf("Failed to generate config: %v", err)
		}
		return
	}

	cfg, err := config.LoadFromFile(*configFile)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	logger := logging.NewLogger(
		cfg.Logging.Level,
		cfg.Logging.Format,
		cfg.Logging.Output,
	)

	logger.Info("Starting load balancer", map[string]interface{}{
		"port":      cfg.Server.Port,
		"algorithm": cfg.LoadBalancer.Algorithm,
	})

	_ = metrics.NewMetrics()

	lb := loadbalancer.NewLoadBalancer(
		cfg.LoadBalancer.SessionTTL,
		cfg.LoadBalancer.Algorithm,
	)

	for _, backendCfg := range cfg.LoadBalancer.Backends {
		if !backendCfg.Enabled {
			logger.Info("Backend disabled, skipping", map[string]interface{}{
				"backend_id": backendCfg.ID,
			})
			continue
		}

		b, err := backend.NewBackend(
			backendCfg.ID,
			strings.TrimSpace(backendCfg.URL),
			backendCfg.Weight,
		)
		if err != nil {
			logger.Error("Failed to create backend", map[string]interface{}{
				"backend_id": backendCfg.ID,
				"error":      err.Error(),
			})
			continue
		}

		lb.AddBackend(b)
		logger.Info("Added backend", map[string]interface{}{
			"backend_id": backendCfg.ID,
			"url":        backendCfg.URL,
			"weight":     backendCfg.Weight,
		})
	}

	go lb.HealthCheck(
		cfg.LoadBalancer.HealthCheckInterval,
		cfg.LoadBalancer.HealthCheckTimeout,
	)

	var handler http.Handler = lb

	if cfg.RateLimit.Enabled {
		rl := ratelimit.NewRateLimiter(
			cfg.RateLimit.RequestsPerSecond,
			cfg.RateLimit.Burst,
			cfg.RateLimit.CleanupInterval,
		)
		handler = rl.Middleware(handler)
		logger.Info("Rate limiting enabled", map[string]interface{}{
			"requests_per_second": cfg.RateLimit.RequestsPerSecond,
			"burst":               cfg.RateLimit.Burst,
		})
	}

	if cfg.Metrics.Enabled {
		go func() {
			metricsAddr := fmt.Sprintf(":%d", cfg.Metrics.Port)
			logger.Info("Starting metrics server", map[string]interface{}{
				"port": cfg.Metrics.Port,
				"path": cfg.Metrics.Path,
			})
			http.Handle(cfg.Metrics.Path, metrics.Handler())
			if err := http.ListenAndServe(metricsAddr, nil); err != nil {
				logger.Error("Metrics server failed", map[string]interface{}{
					"error": err.Error(),
				})
			}
		}()
	}

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	logger.Info("Load balancer started", map[string]interface{}{
		"address": addr,
	})

	server := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	if cfg.Server.TLS.Enabled {
		logger.Info("TLS enabled", map[string]interface{}{
			"cert_file": cfg.Server.TLS.CertFile,
		})
		if err := server.ListenAndServeTLS(cfg.Server.TLS.CertFile, cfg.Server.TLS.KeyFile); err != nil {
			logger.Error("Server failed", map[string]interface{}{"error": err.Error()})
		}
	} else {
		if err := server.ListenAndServe(); err != nil {
			logger.Error("Server failed", map[string]interface{}{"error": err.Error()})
		}
	}
}

func generateDefaultConfig() error {
	cfg := config.DefaultConfig()
	cfg.LoadBalancer.Backends = []config.BackendConfig{
		{ID: "backend-1", URL: "http://localhost:8081", Weight: 3, Enabled: true},
		{ID: "backend-2", URL: "http://localhost:8082", Weight: 2, Enabled: true},
		{ID: "backend-3", URL: "http://localhost:8083", Weight: 1, Enabled: true},
	}

	if err := cfg.SaveToFile("configs/config.toml"); err != nil {
		return err
	}

	log.Println("Default config generated: configs/config.toml")
	return nil
}
