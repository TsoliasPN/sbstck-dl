package cmd

import (
	"context"
	"errors"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/alexferrari88/sbstck-dl/lib"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands

type cookieName string

const (
	substackSid cookieName = "substack.sid"
	connectSid  cookieName = "connect.sid"
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
	proxyURL       string
	verbose        bool
	ratePerSecond  int
	beforeDate     string
	afterDate      string
	idCookieName   cookieName
	idCookieVal    string
	ctx            = context.Background()
	parsedProxyURL *url.URL
	fetcher        *lib.Fetcher
	extractor      *lib.Extractor

	rootCmd = &cobra.Command{
		Use:   "sbstck-dl",
		Short: "Substack Downloader",
		Long:  `sbstck-dl is a command line tool for downloading Substack newsletters for archival purposes, offline reading, or data analysis.`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {

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

			fetcher = lib.NewFetcher(lib.WithRatePerSecond(ratePerSecond), lib.WithProxyURL(parsedProxyURL), lib.WithCookie(cookie))
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
	rootCmd.PersistentFlags().StringVar(&idCookieVal, "cookie_val", "", "The substack.sid/connect.sid cookie value (required for private newsletters)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().IntVarP(&ratePerSecond, "rate", "r", lib.DefaultRatePerSecond, "Specify the rate of requests per second")
	rootCmd.PersistentFlags().StringVar(&beforeDate, "before", "", "Download posts published before this date (format: YYYY-MM-DD)")
	rootCmd.PersistentFlags().StringVar(&afterDate, "after", "", "Download posts published after this date (format: YYYY-MM-DD)")
	rootCmd.MarkFlagsRequiredTogether("cookie_name", "cookie_val")

	rootCmd.AddCommand(downloadCmd)
	rootCmd.AddCommand(listCmd)
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
