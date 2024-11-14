/*
 * Copyright 2023 Marco Confalonieri.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"external-dns-hetzner-webhook/internal/hetzner"
	"external-dns-hetzner-webhook/internal/server"

	"github.com/bsm/openmetrics"
	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/external-dns/provider/webhook/api"

	"github.com/codingconcepts/env"
)

var (
	// notify requires the SIGINT and SIGTERM signals to be sent to the caller.
	notify = func(sig chan os.Signal) {
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	}
)

// healthStatus is the interface used by loop.
type healthStatus interface {
	SetHealth(bool)
	SetReady(bool)
}

// loop waits for a SIGTERM or a SIGINT and then shuts down the server.
func loop(status healthStatus) {
	exitSignal := make(chan os.Signal, 1)
	notify(exitSignal)
	signal := <-exitSignal

	log.Infof("Signal %s received. Shutting down the webhook.", signal.String())
	status.SetHealth(false)
	status.SetReady(false)
}

// main function
func main() {
	// Read server options
	serverOptions := &server.ServerOptions{}
	if err := env.Set(serverOptions); err != nil {
		log.Fatal(err)
	}

	// Create metrics register
	reg := openmetrics.NewConsistentRegistry(time.Now)

	// Start health server
	log.Infof("Starting liveness and readiness server on %s", serverOptions.GetHealthAddress())
	healthStatus := server.HealthStatus{}
	publicServer := server.NewPublicServer(&healthStatus, reg)
	go publicServer.Start(nil, *serverOptions)

	// Read provider configuration
	providerConfig := &hetzner.Configuration{}
	if err := env.Set(providerConfig); err != nil {
		log.Fatal(err)
	}

	// instantiate the Hetzner provider
	provider, err := hetzner.NewHetznerProvider(providerConfig, reg)
	if err != nil {
		panic(err)
	}

	// Start the webhook
	log.Infof("Starting webhook server on %s", serverOptions.GetWebhookAddress())
	startedChan := make(chan struct{})
	go api.StartHTTPApi(
		provider, startedChan,
		serverOptions.GetReadTimeout(),
		serverOptions.GetWriteTimeout(),
		serverOptions.GetWebhookAddress(),
	)

	// Wait for the HTTP server to start and then set the healthy and ready flags
	<-startedChan
	healthStatus.SetHealth(true)
	healthStatus.SetReady(true)

	// Loops until a signal tells us to exit
	loop(&healthStatus)
}
