package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"

	log "github.com/sirupsen/logrus"

	"github.com/ionos-cloud/external-dns-ionos-plugin/cmd/plugin/init/configuration"

	"github.com/ionos-cloud/external-dns-ionos-plugin/pkg/plugin"
)

// Init server initialization function
func Init(config configuration.Config, p *plugin.Plugin) *http.Server {
	r := chi.NewRouter()
	r.Use(plugin.Health)
	r.Get("/records", p.Records)
	r.Post("/records", p.ApplyChanges)
	r.Post("/propertyvaluesequals", p.PropertyValuesEquals)
	r.Post("/adjustendpoints", p.AdjustEndpoints)

	srv := createHTTPServer(fmt.Sprintf("%s:%d", config.ServerHost, config.ServerPort), r)
	go func() {
		log.Infof("starting server on addr: '%s' ", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Errorf("can't serve on addr: '%s', error: %v", srv.Addr, err)
		}
	}()
	return srv
}

func createHTTPServer(addr string, hand http.Handler) *http.Server {
	return &http.Server{
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       120 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		Addr:              addr,
		Handler:           hand,
	}
}

// ShutdownGracefully gracefully shutdown the http server
func ShutdownGracefully(srv *http.Server) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	sig := <-sigCh
	log.Infof("shutting down server due to received signal: %v", sig)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	if err := srv.Shutdown(ctx); err != nil {
		log.Errorf("error shutting down server: %v", err)
	}
	cancel()
}
