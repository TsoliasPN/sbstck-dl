package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type Config struct {
	URL            *string `json:"url,omitempty" yaml:"url,omitempty"`
	Output         *string `json:"output,omitempty" yaml:"output,omitempty"`
	Format         *string `json:"format,omitempty" yaml:"format,omitempty"`
	DryRun         *bool   `json:"dry_run,omitempty" yaml:"dry_run,omitempty"`
	AddSourceURL   *bool   `json:"add_source_url,omitempty" yaml:"add_source_url,omitempty"`
	CreateArchive  *bool   `json:"create_archive,omitempty" yaml:"create_archive,omitempty"`
	DownloadImages *bool   `json:"download_images,omitempty" yaml:"download_images,omitempty"`
	ImageQuality   *string `json:"image_quality,omitempty" yaml:"image_quality,omitempty"`
	ImagesDir      *string `json:"images_dir,omitempty" yaml:"images_dir,omitempty"`
	DownloadFiles  *bool   `json:"download_files,omitempty" yaml:"download_files,omitempty"`
	FileExtensions *string `json:"file_extensions,omitempty" yaml:"file_extensions,omitempty"`
	FilesDir       *string `json:"files_dir,omitempty" yaml:"files_dir,omitempty"`
	Force          *bool   `json:"force,omitempty" yaml:"force,omitempty"`
	SkipExisting   *bool   `json:"skip_existing,omitempty" yaml:"skip_existing,omitempty"`
	RefreshUpdated *bool   `json:"refresh_updated,omitempty" yaml:"refresh_updated,omitempty"`
	Layout         *string `json:"layout,omitempty" yaml:"layout,omitempty"`
	WriteMetadata  *bool   `json:"write_metadata,omitempty" yaml:"write_metadata,omitempty"`
	FailFast       *bool   `json:"fail_fast,omitempty" yaml:"fail_fast,omitempty"`
	ContinueOnErr  *bool   `json:"continue_on_error,omitempty" yaml:"continue_on_error,omitempty"`
	Proxy          *string `json:"proxy,omitempty" yaml:"proxy,omitempty"`
	Rate           *int    `json:"rate,omitempty" yaml:"rate,omitempty"`
	MaxWorkers     *int    `json:"max_workers,omitempty" yaml:"max_workers,omitempty"`
	Before         *string `json:"before,omitempty" yaml:"before,omitempty"`
	After          *string `json:"after,omitempty" yaml:"after,omitempty"`
	CookieName     *string `json:"cookie_name,omitempty" yaml:"cookie_name,omitempty"`
	CookieVal      *string `json:"cookie_val,omitempty" yaml:"cookie_val,omitempty"`
	CookieValFile  *string `json:"cookie_val_file,omitempty" yaml:"cookie_val_file,omitempty"`
	CookieJar      *string `json:"cookie_jar,omitempty" yaml:"cookie_jar,omitempty"`
	CookieKeychain *string `json:"cookie_keychain,omitempty" yaml:"cookie_keychain,omitempty"`
	NotionLabels   *string `json:"notion_labels,omitempty" yaml:"notion_labels,omitempty"`
	Verbose        *bool   `json:"verbose,omitempty" yaml:"verbose,omitempty"`
	LogFormat      *string `json:"log_format,omitempty" yaml:"log_format,omitempty"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".json":
		return decodeConfigJSON(data)
	case ".yaml", ".yml":
		return decodeConfigYAML(data)
	default:
		cfg, err := decodeConfigJSON(data)
		if err == nil {
			return cfg, nil
		}
		return decodeConfigYAML(data)
	}
}

func decodeConfigJSON(data []byte) (*Config, error) {
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func decodeConfigYAML(data []byte) (*Config, error) {
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func applyConfigToCommand(cmd *cobra.Command, cfg *Config) error {
	if cfg == nil {
		return nil
	}

	type setter func(*cobra.Command, string, string) error
	setString := func(cmd *cobra.Command, name string, value *string) error {
		if value == nil {
			return nil
		}
		trimmed := strings.TrimSpace(*value)
		if trimmed == "" {
			return nil
		}
		return setFlagValue(cmd, name, trimmed)
	}
	setBool := func(cmd *cobra.Command, name string, value *bool) error {
		if value == nil {
			return nil
		}
		return setFlagValue(cmd, name, strconv.FormatBool(*value))
	}
	setInt := func(cmd *cobra.Command, name string, value *int) error {
		if value == nil || *value <= 0 {
			return nil
		}
		return setFlagValue(cmd, name, strconv.Itoa(*value))
	}

	if err := setString(cmd, "url", cfg.URL); err != nil {
		return err
	}
	if err := setString(cmd, "output", cfg.Output); err != nil {
		return err
	}
	if err := setString(cmd, "format", cfg.Format); err != nil {
		return err
	}
	if err := setBool(cmd, "dry-run", cfg.DryRun); err != nil {
		return err
	}
	if err := setBool(cmd, "add-source-url", cfg.AddSourceURL); err != nil {
		return err
	}
	if err := setBool(cmd, "create-archive", cfg.CreateArchive); err != nil {
		return err
	}
	if err := setBool(cmd, "download-images", cfg.DownloadImages); err != nil {
		return err
	}
	if err := setString(cmd, "image-quality", cfg.ImageQuality); err != nil {
		return err
	}
	if err := setString(cmd, "images-dir", cfg.ImagesDir); err != nil {
		return err
	}
	if err := setBool(cmd, "download-files", cfg.DownloadFiles); err != nil {
		return err
	}
	if err := setString(cmd, "file-extensions", cfg.FileExtensions); err != nil {
		return err
	}
	if err := setString(cmd, "files-dir", cfg.FilesDir); err != nil {
		return err
	}
	if err := setBool(cmd, "force", cfg.Force); err != nil {
		return err
	}
	if err := setBool(cmd, "skip-existing", cfg.SkipExisting); err != nil {
		return err
	}
	if err := setBool(cmd, "refresh-updated", cfg.RefreshUpdated); err != nil {
		return err
	}
	if err := setString(cmd, "layout", cfg.Layout); err != nil {
		return err
	}
	if err := setBool(cmd, "write-metadata", cfg.WriteMetadata); err != nil {
		return err
	}
	if err := setBool(cmd, "fail-fast", cfg.FailFast); err != nil {
		return err
	}
	if err := setBool(cmd, "continue-on-error", cfg.ContinueOnErr); err != nil {
		return err
	}
	if err := setString(cmd, "proxy", cfg.Proxy); err != nil {
		return err
	}
	if err := setInt(cmd, "rate", cfg.Rate); err != nil {
		return err
	}
	if err := setInt(cmd, "max-workers", cfg.MaxWorkers); err != nil {
		return err
	}
	if err := setString(cmd, "before", cfg.Before); err != nil {
		return err
	}
	if err := setString(cmd, "after", cfg.After); err != nil {
		return err
	}
	if err := setString(cmd, "cookie_name", cfg.CookieName); err != nil {
		return err
	}
	if err := setString(cmd, "cookie_val", cfg.CookieVal); err != nil {
		return err
	}
	if err := setString(cmd, "cookie-val-file", cfg.CookieValFile); err != nil {
		return err
	}
	if err := setString(cmd, "cookie-jar", cfg.CookieJar); err != nil {
		return err
	}
	if err := setString(cmd, "cookie-keychain", cfg.CookieKeychain); err != nil {
		return err
	}
	if err := setString(cmd, "notion-labels", cfg.NotionLabels); err != nil {
		return err
	}
	if err := setBool(cmd, "verbose", cfg.Verbose); err != nil {
		return err
	}
	if err := setString(cmd, "log-format", cfg.LogFormat); err != nil {
		return err
	}

	return nil
}

func setFlagValue(cmd *cobra.Command, name string, value string) error {
	flag := cmd.Flags().Lookup(name)
	if flag == nil || flag.Changed {
		return nil
	}
	if err := cmd.Flags().Set(name, value); err != nil {
		return fmt.Errorf("invalid %s in config: %w", name, err)
	}
	return nil
}
