package main

import (
	"external-dns-hetzner-webhook/internal/hetzner"
	"time"

	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/external-dns/provider/webhook"
)

func main() {
	srvOptions := struct {
		hostname string `env:"SERVER_HOST" envDefault:"0.0.0.0"`
	}{}

	// instantiate the configuration
	config := &hetzner.Configuration{}
	log.Infof("Starting server on %s", srvOptions.hostname)
	// instantiate the Hetzner provider
	provider, err := hetzner.NewHetznerProvider(config)
	if err != nil {
		panic(err)
	}

	startedChan := make(chan struct{})

	go webhook.StartHTTPApi(provider, startedChan, 5*time.Second, 5*time.Second, srvOptions.hostname)
	<-startedChan

	time.Sleep(100000 * time.Second)

}
