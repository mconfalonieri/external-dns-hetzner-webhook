package main

import (
	"time"

	"external-dns-hetzner-webhook/internal/hetzner"

	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/external-dns/provider/webhook"

	"github.com/codingconcepts/env"
)

type serverOptions struct {
	Hostname string `env:"SERVER_HOST" default:"0.0.0.0:8888"`
}

func main() {
	srvOptions := serverOptions{}
	if err := env.Set(&srvOptions); err != nil {
		log.Fatal(err)
	}
	// instantiate the configuration
	config := &hetzner.Configuration{}
	if err := env.Set(&config); err != nil {
		log.Fatal(err)
	}
	log.Infof("Starting server on %s", srvOptions.Hostname)

	// instantiate the Hetzner provider
	provider, err := hetzner.NewHetznerProvider(config)
	if err != nil {
		panic(err)
	}

	startedChan := make(chan struct{})

	go webhook.StartHTTPApi(provider, startedChan, 5*time.Second, 5*time.Second, srvOptions.Hostname)
	<-startedChan

	time.Sleep(100000 * time.Second)
}
