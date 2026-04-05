package config

import (
	"fmt"

	"github.com/0xERR0R/blocky/log"
	"github.com/sirupsen/logrus"
)

const UpstreamDefaultCfgName = "default"

// upstreamsYAMLSentinel rejects any `upstreams:` section in YAML. Upstream
// configuration lives in the SQLite config store and is managed via the web UI.
// A hard error is returned with a pointer to the migration docs.
type upstreamsYAMLSentinel struct{}

func (upstreamsYAMLSentinel) UnmarshalYAML(_ func(any) error) error {
	return fmt.Errorf(
		"the 'upstreams:' section has moved to the SQLite config store: " +
			"remove the 'upstreams:' block from your YAML configuration and manage " +
			"upstream groups + settings via the web UI (see docs/migration-upstreams.md)",
	)
}

// Upstreams upstream servers configuration
type Upstreams struct {
	Init      Init             `yaml:"init"`
	Timeout   Duration         `default:"2s"            yaml:"timeout"` // always > 0
	Groups    UpstreamGroups   `yaml:"groups"`
	Strategy  UpstreamStrategy `default:"parallel_best" yaml:"strategy"`
	UserAgent string           `yaml:"userAgent"`
}

type UpstreamGroups map[string][]Upstream

func (c *Upstreams) validate(logger *logrus.Entry) {
	defaults := mustDefault[Upstreams]()

	if !c.Timeout.IsAboveZero() {
		logger.Warnf("upstreams.timeout <= 0, setting to %s", defaults.Timeout)
		c.Timeout = defaults.Timeout
	}
}

// IsEnabled implements `config.Configurable`.
func (c *Upstreams) IsEnabled() bool {
	return len(c.Groups) != 0
}

// LogConfig implements `config.Configurable`.
func (c *Upstreams) LogConfig(logger *logrus.Entry) {
	logger.Info("init:")
	log.WithIndent(logger, "  ", c.Init.LogConfig)

	logger.Info("timeout: ", c.Timeout)
	logger.Info("strategy: ", c.Strategy)
	logger.Info("groups:")

	for name, upstreams := range c.Groups {
		logger.Infof("  %s:", name)

		for _, upstream := range upstreams {
			logger.Infof("    - %s", upstream)
		}
	}
}

// UpstreamGroup represents the config for one group (upstream branch)
type UpstreamGroup struct {
	Upstreams

	Name string // group name
}

// NewUpstreamGroup creates an UpstreamGroup with the given name and upstreams.
//
// The upstreams from `cfg.Groups` are ignored.
func NewUpstreamGroup(name string, cfg Upstreams, upstreams []Upstream) UpstreamGroup {
	group := UpstreamGroup{
		Name:      name,
		Upstreams: cfg,
	}

	group.Groups = UpstreamGroups{name: upstreams}

	return group
}

func (c *UpstreamGroup) GroupUpstreams() []Upstream {
	return c.Groups[c.Name]
}

// IsEnabled implements `config.Configurable`.
func (c *UpstreamGroup) IsEnabled() bool {
	return len(c.GroupUpstreams()) != 0
}

// LogConfig implements `config.Configurable`.
func (c *UpstreamGroup) LogConfig(logger *logrus.Entry) {
	logger.Info("group: ", c.Name)
	logger.Info("upstreams:")

	for _, upstream := range c.GroupUpstreams() {
		logger.Infof("  - %s", upstream)
	}
}
