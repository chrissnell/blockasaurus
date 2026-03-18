package util

import (
	"regexp"
	"strings"
)

var (
	slugDisallowed    = regexp.MustCompile(`[^a-z0-9-]`)
	slugMultiHyphens  = regexp.MustCompile(`-{2,}`)
)

// SanitizeGroupSlug converts a human-readable client group name into a
// DNS-legal label suitable for use as a subdomain. The result is lowercase,
// alphanumeric + hyphens only, max 63 characters.
func SanitizeGroupSlug(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))

	// Replace common separators with hyphens
	s = strings.NewReplacer(" ", "-", "_", "-", ".", "-").Replace(s)

	// Strip anything not alphanumeric or hyphen
	s = slugDisallowed.ReplaceAllString(s, "")

	// Collapse multiple hyphens
	s = slugMultiHyphens.ReplaceAllString(s, "-")

	// Trim leading/trailing hyphens
	s = strings.Trim(s, "-")

	// DNS label limit
	if len(s) > 63 {
		s = s[:63]
		s = strings.TrimRight(s, "-")
	}

	return s
}
