package config

import (
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/0xERR0R/blocky/util"

	"github.com/sirupsen/logrus"
)

// RebindingProtection drops answers from the general upstream resolvers that contain
// private, loopback, link-local or unspecified IPs (DNS rebinding protection).
type RebindingProtection struct {
	// Enable DNS rebinding protection for answers from the general upstream resolvers.
	Enable bool `yaml:"enable"`
	// Domains (including their subdomains) that may resolve to non-public IPs (split-horizon DNS).
	AllowedDomains []string `yaml:"allowedDomains"`

	normalizedAllowedDomains []string
}

// IsEnabled implements `config.Configurable`.
func (c *RebindingProtection) IsEnabled() bool {
	return c.Enable
}

// LogConfig implements `config.Configurable`.
func (c *RebindingProtection) LogConfig(logger *logrus.Entry) {
	logger.Infof("allowed domains (%d):", len(c.AllowedDomains))

	for _, domain := range c.AllowedDomains {
		logger.Infof("  - %s", domain)
	}
}

// ValidateAllowedDomain returns an error if domain is not usable as an allowlist
// entry. Exported because the config file is not the only writer: the allowlist is
// also editable through the config API, which must reject exactly what a YAML load
// would reject, or the two paths drift.
func ValidateAllowedDomain(domain string) error {
	if strings.TrimSpace(domain) == "" {
		return errors.New("must not be empty")
	}

	// queryLog.ignore.domains supports wildcard/regex syntax; this list does not —
	// reject such entries (and other never-matching forms like padded strings or
	// degenerate dots) instead of silently ignoring them; whitespace is rejected
	// via unicode.IsSpace so the rule covers everything trimming would touch
	if strings.ContainsAny(domain, "*/") || strings.ContainsFunc(domain, unicode.IsSpace) ||
		strings.HasPrefix(domain, ".") || strings.Contains(domain, "..") {
		return errors.New("must be a plain domain (no wildcards, regexes or whitespace);" +
			" subdomains match automatically")
	}

	return nil
}

// SetAllowedDomains validates and replaces the allowlist, refreshing the cached
// normalized form. Assigning AllowedDomains directly would leave that cache holding
// the previous entries — the resolver reads the cache, so the change would appear to
// apply and silently not. Use this from anything that rewrites the list after load
// (the config store rebuilding from DB state, tests).
func (c *RebindingProtection) SetAllowedDomains(domains []string) error {
	for i, domain := range domains {
		if err := ValidateAllowedDomain(domain); err != nil {
			return fmt.Errorf("allowedDomains[%d] (%q) %w", i, domain, err)
		}
	}

	c.AllowedDomains = domains
	c.normalizedAllowedDomains = normalizeDomains(domains)

	return nil
}

// validate returns an error if the allowlist contains an empty or non-plain-domain
// entry. It runs even when the protection is disabled, so config errors surface
// before the user enables it.
func (c *RebindingProtection) validate() error {
	for i, domain := range c.AllowedDomains {
		if err := ValidateAllowedDomain(domain); err != nil {
			return fmt.Errorf("rebindingProtection.allowedDomains[%d] (%q) %w", i, domain, err)
		}
	}

	c.normalizedAllowedDomains = normalizeDomains(c.AllowedDomains)

	return nil
}

// NormalizedAllowedDomains returns the allowlist entries in canonical form
// (lowercase, no trailing dot) — the form resolvers must match against.
// validate caches the result; configs that were never validated (e.g.
// hand-built in tests) are normalized on the fly.
func (c *RebindingProtection) NormalizedAllowedDomains() []string {
	if c.normalizedAllowedDomains == nil {
		return normalizeDomains(c.AllowedDomains)
	}

	return c.normalizedAllowedDomains
}

func normalizeDomains(domains []string) []string {
	if domains == nil {
		return nil
	}

	normalized := make([]string, len(domains))
	for i, domain := range domains {
		normalized[i] = util.ExtractDomainOnly(domain)
	}

	return normalized
}
