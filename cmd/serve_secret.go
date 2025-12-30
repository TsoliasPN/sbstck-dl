package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type secretRequest struct {
	Key           string `json:"key"`
	CookieVal     string `json:"cookie_val"`
	CookieValFile string `json:"cookie_val_file"`
}

type secretResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

func serveSecret(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if !requireCSRF(w, r) {
		return
	}

	var req secretRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, secretResponse{
			OK:    false,
			Error: fmt.Sprintf("Invalid request payload: %v", err),
		})
		return
	}

	key := strings.TrimSpace(req.Key)
	if key == "" {
		writeJSON(w, http.StatusBadRequest, secretResponse{
			OK:    false,
			Error: "Keychain entry name is required.",
		})
		return
	}

	value, err := resolveCookieValue(strings.TrimSpace(req.CookieVal), strings.TrimSpace(req.CookieValFile))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, secretResponse{
			OK:    false,
			Error: fmt.Sprintf("Failed to read cookie value: %v", err),
		})
		return
	}
	if value == "" {
		writeJSON(w, http.StatusBadRequest, secretResponse{
			OK:    false,
			Error: "Cookie value is required.",
		})
		return
	}

	if err := secretStore.Set(key, value); err != nil {
		writeJSON(w, http.StatusInternalServerError, secretResponse{
			OK:    false,
			Error: fmt.Sprintf("Failed to store keychain entry: %v", err),
		})
		return
	}

	writeJSON(w, http.StatusOK, secretResponse{OK: true})
}
