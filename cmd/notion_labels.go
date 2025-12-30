package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alexferrari88/sbstck-dl/lib"
	"gopkg.in/yaml.v3"
)

func loadNotionLabelMap(path string) (map[string]string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	labels, err := decodeLabelMap(data, filepath.Ext(path))
	if err != nil {
		return nil, err
	}

	normalized := make(map[string]string)
	for key, value := range labels {
		normalizedKey, ok := lib.NormalizeNotionURL(strings.TrimSpace(key))
		if !ok {
			continue
		}
		label := strings.TrimSpace(value)
		if label == "" {
			continue
		}
		normalized[normalizedKey] = label
	}

	return normalized, nil
}

func decodeLabelMap(data []byte, ext string) (map[string]string, error) {
	ext = strings.ToLower(ext)
	var labels map[string]string
	switch ext {
	case ".json":
		if err := json.Unmarshal(data, &labels); err != nil {
			return nil, fmt.Errorf("parse JSON label map: %w", err)
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &labels); err != nil {
			return nil, fmt.Errorf("parse YAML label map: %w", err)
		}
	default:
		if err := json.Unmarshal(data, &labels); err != nil {
			if err := yaml.Unmarshal(data, &labels); err != nil {
				return nil, fmt.Errorf("parse label map: %w", err)
			}
		}
	}
	if labels == nil {
		labels = make(map[string]string)
	}
	return labels, nil
}
