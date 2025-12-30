package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	logFormatText = "text"
	logFormatJSON = "json"
)

func normalizeLogFormat(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func logEvent(event string, fields map[string]any) {
	if logFormat != logFormatJSON {
		return
	}

	entry := map[string]any{
		"time":  time.Now().UTC().Format(time.RFC3339),
		"event": event,
	}
	for key, value := range fields {
		entry[key] = value
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return
	}

	fmt.Println(string(data))
}
