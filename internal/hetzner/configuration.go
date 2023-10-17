package hetzner

// Configuration for the Hetzner provider.
type Configuration struct {
	APIToken       string `env:"HETZNER_API_TOKEN,notEmpty"`
	APIEndpointURL string `env:"HETZNER_API_URL" envDefault:"https://api.hosting.hetzner.com/dns"`
	APIVersion     string `env:"HETZNER_API_VERSION" envDefault:"v1"`
	Debug          bool   `env:"HETZNER_DEBUG" envDefault:"false"`
	DryRun         bool   `env:"DRY_RUN" envDefault:"false"`
}
