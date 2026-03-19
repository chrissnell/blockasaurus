package server

import (
	"encoding/json"
	"net/http"

	"github.com/0xERR0R/blocky/config"
)

type endpointInfo struct {
	Domains          []string `json:"domains"`
	CpeID            bool     `json:"cpeId"`
	DOHPath          string   `json:"dohPath"`
	HasTLS           bool     `json:"hasTls"`
	HasHTTP          bool     `json:"hasHttp"`
	HasSelfSignedCert bool   `json:"hasSelfSignedCert"`
	AdvertiseAddress string   `json:"advertiseAddress,omitempty"`
}

func handleEndpointInfo(cfg *config.Config) http.HandlerFunc {
	info := endpointInfo{
		Domains:          cfg.ClientGroupEndpoints.Domains,
		CpeID:            cfg.ClientGroupEndpoints.CpeID,
		DOHPath:          cfg.Ports.DOHPath,
		HasTLS:           len(cfg.Ports.TLS) > 0 || len(cfg.Ports.HTTPS) > 0,
		HasHTTP:          len(cfg.Ports.HTTP) > 0,
		HasSelfSignedCert: cfg.CertFile != "" && hasSelfSignedRoot(cfg.CertFile),
		AdvertiseAddress: cfg.ClientGroupEndpoints.AdvertiseAddress,
	}

	if info.Domains == nil {
		info.Domains = []string{}
	}

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(info)
	}
}
