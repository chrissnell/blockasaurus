package server

import (
	"encoding/json"
	"net/http"

	"github.com/0xERR0R/blocky/config"
	"github.com/0xERR0R/blocky/pkg/advertise"
)

type endpointInfo struct {
	Domains []string `json:"domains"`
	CpeID   bool     `json:"cpeId"`
	DOHPath string   `json:"dohPath"`
	HasTLS  bool     `json:"hasTls"`
	HasHTTP bool     `json:"hasHttp"`
	// HasDoH3 reports that the DoH endpoint is also reachable over HTTP/3.
	// The HTTP/3 listener binds once, at startup, so reading it from the
	// startup config is reading the live state and not a stale copy.
	HasDoH3           bool   `json:"hasDoH3"`
	HasSelfSignedCert bool   `json:"hasSelfSignedCert"`
	AdvertiseAddress  string `json:"advertiseAddress,omitempty"`
}

func handleEndpointInfo(cfg *config.Config) http.HandlerFunc {
	info := endpointInfo{
		Domains:           cfg.ClientGroupEndpoints.Domains,
		CpeID:             cfg.ClientGroupEndpoints.CpeID,
		DOHPath:           cfg.Ports.DOHPath,
		HasTLS:            len(cfg.Ports.TLS) > 0 || len(cfg.Ports.HTTPS) > 0,
		HasHTTP:           len(cfg.Ports.HTTP) > 0,
		HasDoH3:           doh3Active(cfg),
		HasSelfSignedCert: cfg.CertFile != "" && hasSelfSignedRoot(cfg.CertFile),
	}

	// Resolve "auto" or explicit IP to an actual address for the frontend
	if ip, err := advertise.ResolveAddress(cfg.ClientGroupEndpoints.AdvertiseAddress); err == nil && ip != nil {
		info.AdvertiseAddress = ip.String()
	}

	if info.Domains == nil {
		info.Domains = []string{}
	}

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(info)
	}
}
