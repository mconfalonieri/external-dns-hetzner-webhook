package main

import (
	"fmt"

	"github.com/ionos-cloud/external-dns-ionos-plugin/cmd/plugin/init/configuration"
	"github.com/ionos-cloud/external-dns-ionos-plugin/cmd/plugin/init/dnsprovider"
	"github.com/ionos-cloud/external-dns-ionos-plugin/cmd/plugin/init/logging"
	"github.com/ionos-cloud/external-dns-ionos-plugin/cmd/plugin/init/server"
	"github.com/ionos-cloud/external-dns-ionos-plugin/pkg/plugin"
)

const banner = `
  ___ ___  _  _  ___  ___  
 |_ _/ _ \| \| |/ _ \/ __| 
  | | (_) | .  | (_) \__ \
 |___\___/|_|\_|\___/|___/
 external-dns-ionos-plugin
 version: %s (%s)

`

var (
	Version = "local"
	Gitsha  = "?"
)

func main() {
	fmt.Printf(banner, Version, Gitsha)
	logging.Init()
	config := configuration.Init()
	srv := server.Init(config, plugin.New(dnsprovider.Init(config)))
	server.ShutdownGracefully(srv)
}
