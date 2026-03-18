package config

import (
	"github.com/sirupsen/logrus"
)

// ClientGroupEndpoints configures per-client-group DNS identification
// via subdomain hostnames, URL paths, and EDNS CPE-ID.
type ClientGroupEndpoints struct {
	// Domains lists base domains for subdomain-based client group identification.
	// A query to {group-slug}.{domain} extracts the group slug from TLS SNI or HTTP Host.
	Domains []string `yaml:"domains"`

	// CpeID enables EDNS CPE-ID (option 65074) extraction for plain DNS queries.
	CpeID bool `yaml:"cpeId" default:"true"`
}

// IsEnabled implements `config.Configurable`.
func (c *ClientGroupEndpoints) IsEnabled() bool {
	return len(c.Domains) > 0 || c.CpeID
}

// LogConfig implements `config.Configurable`.
func (c *ClientGroupEndpoints) LogConfig(logger *logrus.Entry) {
	logger.Infof("cpeId = %t", c.CpeID)

	if len(c.Domains) > 0 {
		logger.Info("domains:")

		for _, d := range c.Domains {
			logger.Infof("  - %s", d)
		}
	}
}
