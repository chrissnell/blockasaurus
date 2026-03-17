package server

import (
	"encoding/json"
	"net/http"

	"github.com/0xERR0R/blocky/util"
)

func handleVersion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	_ = json.NewEncoder(w).Encode(map[string]string{
		"version": util.Version,
	})
}
