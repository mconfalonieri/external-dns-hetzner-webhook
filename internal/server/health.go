package server

import (
	"net"
	"net/http"
	"sync"

	log "github.com/sirupsen/logrus"
)

// HealthStatus contains the health and ready statuses for the webhook.
type HealthStatus struct {
	m       sync.Mutex
	healthy bool
	ready   bool
}

// SetHealth sets the health status.
func (h *HealthStatus) SetHealth(v bool) {
	h.m.Lock()
	h.healthy = v
	h.m.Unlock()
}

// SetReady sets the readiness status.
func (h *HealthStatus) SetReady(v bool) {
	h.m.Lock()
	h.ready = v
	h.m.Unlock()
}

// IsHealthy returns the healthy flag.
func (h *HealthStatus) IsHealthy() bool {
	var healthy bool
	h.m.Lock()
	healthy = h.healthy
	h.m.Unlock()
	return healthy
}

// IsReady returns the readiness status.
func (h *HealthStatus) IsReady() bool {
	var ready bool
	h.m.Lock()
	ready = h.ready
	h.m.Unlock()
	return ready
}

// HealthServer is the liveness and readiness server.
type HealthServer struct {
	status *HealthStatus
}

// livenessHandler checks if the server is healthy. It writes 200/OK if the
// healthy flag is set to "true" and 503/Service Unavailable otherwise.
func (s HealthServer) livenessHandler(w http.ResponseWriter, r *http.Request) {
	healthy := s.status.IsHealthy()
	var err error
	if healthy {
		_, err = w.Write([]byte(http.StatusText(http.StatusOK)))
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, err = w.Write([]byte(http.StatusText(http.StatusServiceUnavailable)))
	}
	if err != nil {
		log.Warn("Could not answer to a liveness probe: ", err.Error())
	}
}

// readinessHandler checks if the server is ready. It writes 200/OK if the
// healthy flag is set to "true" and 503/Service Unavailable otherwise.
func (s HealthServer) readinessHandler(w http.ResponseWriter, r *http.Request) {
	ready := s.status.IsReady()
	var err error
	if ready {
		_, err = w.Write([]byte(http.StatusText(http.StatusOK)))
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, err = w.Write([]byte(http.StatusText(http.StatusServiceUnavailable)))
	}
	if err != nil {
		log.Warn("Could not answer to a readiness probe: ", err.Error())
	}
}

// Start starts the liveness and readiness server.
func (s *HealthServer) Start(status *HealthStatus, startedChan chan struct{}, options ServerOptions) {
	s.status = status

	mux := http.NewServeMux()

	mux.HandleFunc("/", s.readinessHandler)
	mux.HandleFunc("/ready", s.readinessHandler)
	mux.HandleFunc("/health", s.livenessHandler)

	address := options.GetHealthAddress()

	srv := &http.Server{
		Addr:         address,
		Handler:      mux,
		ReadTimeout:  options.GetReadTimeout(),
		WriteTimeout: options.GetWriteTimeout(),
	}

	l, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatal(err)
	}

	if startedChan != nil {
		startedChan <- struct{}{}
	}

	if err := srv.Serve(l); err != nil {
		log.Fatal(err)
	}
}
