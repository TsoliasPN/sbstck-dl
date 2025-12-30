package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

func TestLoadConfigJSON(t *testing.T) {
	data := `{"url":"https://example.substack.com","rate":5,"dry_run":true,"cookie_name":"substack.sid"}`
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.URL == nil || *cfg.URL != "https://example.substack.com" {
		t.Fatalf("unexpected url: %#v", cfg.URL)
	}
	if cfg.Rate == nil || *cfg.Rate != 5 {
		t.Fatalf("unexpected rate: %#v", cfg.Rate)
	}
	if cfg.DryRun == nil || *cfg.DryRun != true {
		t.Fatalf("unexpected dry_run: %#v", cfg.DryRun)
	}
	if cfg.CookieName == nil || *cfg.CookieName != "substack.sid" {
		t.Fatalf("unexpected cookie_name: %#v", cfg.CookieName)
	}
}

func TestLoadConfigYAML(t *testing.T) {
	data := "url: https://example.substack.com\nmax_workers: 7\nverbose: true\n"
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.MaxWorkers == nil || *cfg.MaxWorkers != 7 {
		t.Fatalf("unexpected max_workers: %#v", cfg.MaxWorkers)
	}
	if cfg.Verbose == nil || *cfg.Verbose != true {
		t.Fatalf("unexpected verbose: %#v", cfg.Verbose)
	}
}

func TestApplyConfigToCommand(t *testing.T) {
	cmd := &cobra.Command{}
	var cn cookieName
	cmd.Flags().Var(&cn, "cookie_name", "")
	cmd.Flags().String("url", "", "")
	cmd.Flags().Int("rate", 2, "")
	cmd.Flags().Bool("dry-run", false, "")

	cfg := &Config{
		URL:        strPtr("https://example.substack.com"),
		Rate:       intPtr(5),
		DryRun:     boolPtr(true),
		CookieName: strPtr("substack.sid"),
	}
	if err := applyConfigToCommand(cmd, cfg); err != nil {
		t.Fatalf("apply config: %v", err)
	}

	urlValue, _ := cmd.Flags().GetString("url")
	if urlValue != "https://example.substack.com" {
		t.Fatalf("unexpected url: %s", urlValue)
	}
	rateValue, _ := cmd.Flags().GetInt("rate")
	if rateValue != 5 {
		t.Fatalf("unexpected rate: %d", rateValue)
	}
	dryRunValue, _ := cmd.Flags().GetBool("dry-run")
	if !dryRunValue {
		t.Fatalf("expected dry_run to be true")
	}
	if cn.String() != "substack.sid" {
		t.Fatalf("unexpected cookie name: %s", cn.String())
	}
}

func TestApplyConfigRespectsFlags(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("url", "", "")
	cmd.Flags().String("format", "html", "")
	if err := cmd.Flags().Set("url", "https://from-flag.example"); err != nil {
		t.Fatalf("set url: %v", err)
	}

	cfg := &Config{
		URL:    strPtr("https://from-config.example"),
		Format: strPtr("md"),
	}
	if err := applyConfigToCommand(cmd, cfg); err != nil {
		t.Fatalf("apply config: %v", err)
	}
	urlValue, _ := cmd.Flags().GetString("url")
	if urlValue != "https://from-flag.example" {
		t.Fatalf("expected url to remain from flag, got %s", urlValue)
	}
	formatValue, _ := cmd.Flags().GetString("format")
	if formatValue != "md" {
		t.Fatalf("expected format from config, got %s", formatValue)
	}
}

func TestApplyConfigInvalidCookie(t *testing.T) {
	cmd := &cobra.Command{}
	var cn cookieName
	cmd.Flags().Var(&cn, "cookie_name", "")

	cfg := &Config{
		CookieName: strPtr("bad.cookie"),
	}
	if err := applyConfigToCommand(cmd, cfg); err == nil {
		t.Fatalf("expected error for invalid cookie name")
	}
}

func strPtr(value string) *string {
	return &value
}

func intPtr(value int) *int {
	return &value
}

func boolPtr(value bool) *bool {
	return &value
}
