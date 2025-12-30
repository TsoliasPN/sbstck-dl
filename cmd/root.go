package cmd

import (
	"context"
	"errors"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/alexferrari88/sbstck-dl/lib"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands

type cookieName string

const (
	substackSid     cookieName = "substack.sid"
	connectSid      cookieName = "connect.sid"
	cookieValEnvVar            = "SBSTCK_COOKIE_VAL"
)

func (c *cookieName) String() string {
	return string(*c)
}

func (c *cookieName) Set(val string) error {
	switch val {
	case "substack.sid", "connect.sid":
		*c = cookieName(val)
	default:
		return errors.New("invalid cookie name: must be either substack.sid or connect.sid")
	}
	return nil
}

func (c *cookieName) Type() string {
	return "cookieName"
}

var (
	proxyURL         string
	verbose          bool
	ratePerSecond    int
	maxWorkers       int
	logFormat        string
	beforeDate       string
	afterDate        string
	idCookieName     cookieName
	idCookieVal      string
	cookieValFile    string
	cookieJarPath    string
	cookieKeychain   string
	notionLabelsPath string
	configPath       string
	ctx              = context.Background()
	parsedProxyURL   *url.URL
	fetcher          *lib.Fetcher
	extractor        *lib.Extractor

	rootCmd = &cobra.Command{
		Use:   "sbstck-dl",
		Short: "Substack Downloader",
		Long:  `sbstck-dl is a command line tool for downloading Substack newsletters for archival purposes, offline reading, or data analysis.`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if configPath != "" {
				cfg, err := LoadConfig(configPath)
				if err != nil {
					log.Fatalf("failed to load config: %v", err)
				}
				if err := applyConfigToCommand(cmd, cfg); err != nil {
					log.Fatalf("failed to apply config: %v", err)
				}
			}

			var cookie *http.Cookie

			if proxyURL != "" {
				var err error
				parsedProxyURL, err = parseURL(proxyURL)
				if err != nil {
					log.Fatal(err)
				}
			}

			if ratePerSecond == 0 {
				log.Fatal("rate must be greater than 0")
			}
			if maxWorkers <= 0 {
				log.Fatal("max-workers must be greater than 0")
			}
			logFormat = normalizeLogFormat(logFormat)
			if logFormat != logFormatText && logFormat != logFormatJSON {
				log.Fatal("log-format must be either \"text\" or \"json\"")
			}

			var err error
			idCookieVal, err = resolveCookieValue(idCookieVal, cookieValFile)
			if err != nil {
				log.Fatalf("failed to resolve cookie value: %v", err)
			}

			if idCookieVal == "" && cookieKeychain != "" {
				value, err := secretStore.Get(cookieKeychain)
				if err != nil {
					log.Fatalf("failed to read cookie from keychain: %v", err)
				}
				idCookieVal = value
			}

			domainHint := extractDomainHint(cmd)
			if idCookieVal == "" && cookieJarPath != "" {
				jarName, jarValue, err := readCookieFromJar(cookieJarPath, idCookieName, domainHint)
				if err != nil {
					log.Fatalf("failed to read cookie jar: %v", err)
				}
				if idCookieName == "" {
					idCookieName = jarName
				}
				idCookieVal = jarValue
			}

			if idCookieName != "" && idCookieVal == "" {
				log.Fatalf("cookie_val is required when cookie_name is set (or set %s, --cookie-val-file, or --cookie-jar)", cookieValEnvVar)
			}
			if idCookieName == "" && idCookieVal != "" {
				log.Fatal("cookie_name is required when cookie_val is set")
			}

			if idCookieVal != "" && idCookieName != "" {
				if idCookieName == substackSid {
					cookie = &http.Cookie{
						Name:  "substack.sid",
						Value: idCookieVal,
					}
				} else if idCookieName == connectSid {
					cookie = &http.Cookie{
						Name:  "connect.sid",
						Value: idCookieVal,
					}
				}
			}

			fetcher = lib.NewFetcher(
				lib.WithRatePerSecond(ratePerSecond),
				lib.WithMaxWorkers(maxWorkers),
				lib.WithProxyURL(parsedProxyURL),
				lib.WithCookie(cookie),
			)
			extractor = lib.NewExtractor(fetcher)
		},
	}
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&proxyURL, "proxy", "x", "", "Specify the proxy url")
	rootCmd.PersistentFlags().Var(&idCookieName, "cookie_name", "Either \"substack.sid\" or \"connect.sid\", based on the cookie you have (required for private newsletters)")
	rootCmd.PersistentFlags().StringVar(&idCookieVal, "cookie_val", "", "The substack.sid/connect.sid cookie value (required for private newsletters; or set SBSTCK_COOKIE_VAL)")
	rootCmd.PersistentFlags().StringVar(&cookieValFile, "cookie-val-file", "", "Read cookie value from a file (overrides SBSTCK_COOKIE_VAL)")
	rootCmd.PersistentFlags().StringVar(&cookieJarPath, "cookie-jar", "", "Read cookies from a Netscape cookie jar file (cookies.txt)")
	rootCmd.PersistentFlags().StringVar(&cookieKeychain, "cookie-keychain", "", "Read cookie value from OS keychain/credential manager")
	rootCmd.PersistentFlags().StringVar(&notionLabelsPath, "notion-labels", "", "Path to a YAML/JSON map of Notion URL to label for notion-links output")
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "Load options from a YAML/JSON config file")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().IntVarP(&ratePerSecond, "rate", "r", lib.DefaultRatePerSecond, "Specify the rate of requests per second")
	rootCmd.PersistentFlags().StringVar(&beforeDate, "before", "", "Download posts published before this date (format: YYYY-MM-DD)")
	rootCmd.PersistentFlags().StringVar(&afterDate, "after", "", "Download posts published after this date (format: YYYY-MM-DD)")
	rootCmd.PersistentFlags().IntVar(&maxWorkers, "max-workers", lib.DefaultMaxWorkers, "Maximum parallel workers for downloading posts (rate limiting still applies)")
	rootCmd.PersistentFlags().IntVar(&maxWorkers, "concurrency", lib.DefaultMaxWorkers, "Alias for --max-workers")
	rootCmd.PersistentFlags().StringVar(&logFormat, "log-format", logFormatText, "Log format (text or json)")

	rootCmd.AddCommand(downloadCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(archiveCmd)
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(versionCmd)
}

func makeDateFilterFunc(beforeDate string, afterDate string) lib.DateFilterFunc {
	before, beforeSet := parseDateInput(beforeDate)
	if beforeDate != "" && !beforeSet {
		log.Printf("Invalid --before date %q, ignoring it\n", beforeDate)
	}

	after, afterSet := parseDateInput(afterDate)
	if afterDate != "" && !afterSet {
		log.Printf("Invalid --after date %q, ignoring it\n", afterDate)
	}

	if !beforeSet && !afterSet {
		return nil
	}

	return func(date string) bool {
		parsed, ok := parseDateInput(date)
		if !ok {
			return false
		}

		if beforeSet && !parsed.Before(before) {
			return false
		}
		if afterSet && !parsed.After(after) {
			return false
		}
		return true
	}
}

func parseDateInput(value string) (time.Time, bool) {
	if value == "" {
		return time.Time{}, false
	}
	layouts := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02",
	}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}

func extractDomainHint(cmd *cobra.Command) string {
	value, err := cmd.Flags().GetString("url")
	if err != nil || value == "" {
		return ""
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return ""
	}
	return parsed.Hostname()
}

func resolveCookieValue(flagValue string, filePath string) (string, error) {
	if flagValue != "" {
		return flagValue, nil
	}
	if filePath != "" {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(data)), nil
	}
	return os.Getenv(cookieValEnvVar), nil
}
