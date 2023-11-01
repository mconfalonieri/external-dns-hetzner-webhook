package main

import (
	"external-dns-hetzner-webhook/internal/hetzner"
	"external-dns-hetzner-webhook/internal/server"
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/external-dns/provider/webhook"

	"github.com/codingconcepts/env"
)

func loop(status *server.HealthStatus) {
	exitSignal := make(chan os.Signal, 1)
	signal.Notify(exitSignal, syscall.SIGINT, syscall.SIGTERM)
	signal := <-exitSignal

	log.Infof("Signal %s received. Shutting down the webhook.", signal.String())
	status.SetHealth(false)
	status.SetReady(false)
}

func main() {
	// Read server options
	serverOptions := &server.ServerOptions{}
	if err := env.Set(serverOptions); err != nil {
		log.Fatal(err)
	}

	// Start health server
	log.Infof("Starting liveness and readiness server on %s", serverOptions.GetHealthAddress())
	healthStatus := server.HealthStatus{}
	healthServer := server.HealthServer{}
	go healthServer.Start(&healthStatus, nil, *serverOptions)

	// Read provider configuration
	providerConfig := &hetzner.Configuration{}
	if err := env.Set(providerConfig); err != nil {
		log.Fatal(err)
	}

	// instantiate the Hetzner provider
	provider, err := hetzner.NewHetznerProvider(providerConfig)
	if err != nil {
		panic(err)
	}

	log.Infof("Starting webhook server on %s", serverOptions.GetWebhookAddress())
	startedChan := make(chan struct{})
	go webhook.StartHTTPApi(
		provider, startedChan,
		serverOptions.GetReadTimeout(),
		serverOptions.GetWriteTimeout(),
		serverOptions.GetWebhookAddress(),
	)
	<-startedChan
	healthStatus.SetHealth(true)
	healthStatus.SetReady(true)

	// Loops until a signal tells us to exit
	loop(&healthStatus)
}
