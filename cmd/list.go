package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
)

// listCmd represents the list command
var (
	pubUrl           string
	listJSON         bool
	listWithMetadata bool
	listCmd          = &cobra.Command{
		Use:   "list",
		Short: "List the posts of a Substack",
		Long:  `List the posts of a Substack`,
		Run: func(cmd *cobra.Command, args []string) {
			if listWithMetadata && !listJSON {
				log.Fatal("--with-metadata requires --json")
			}

			parsedURL, err := parseURL(pubUrl)
			if err != nil {
				log.Fatal(err)
			}
			mainWebsite := fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host)
			statusf := func(format string, args ...any) {
				if !verbose {
					return
				}
				if listJSON {
					fmt.Fprintf(os.Stderr, format, args...)
					return
				}
				fmt.Printf(format, args...)
			}
			statusf("Main website: %s\n", mainWebsite)
			statusf("Getting all posts URLs...\n")
			dateFilterfunc := makeDateFilterFunc(beforeDate, afterDate)
			urls, err := extractor.GetAllPostsURLs(ctx, mainWebsite, dateFilterfunc)
			if err != nil {
				log.Fatal(err)
			}
			statusf("Found %d posts.\n", len(urls))

			if listJSON {
				output, err := buildListOutput(ctx, urls, listWithMetadata)
				if err != nil {
					log.Fatal(err)
				}
				if err := json.NewEncoder(os.Stdout).Encode(output); err != nil {
					log.Fatal(err)
				}
				return
			}

			for _, url := range urls {
				fmt.Println(url)
			}
		},
	}
)

func init() {
	listCmd.Flags().StringVarP(&pubUrl, "url", "u", "", "Specify the Substack url")
	listCmd.Flags().BoolVar(&listJSON, "json", false, "Output JSON")
	listCmd.Flags().BoolVar(&listWithMetadata, "with-metadata", false, "Include title and date in JSON output (fetches each post)")
	listCmd.MarkFlagRequired("url")
}

type listPostOutput struct {
	URL   string `json:"url"`
	Title string `json:"title,omitempty"`
	Date  string `json:"date,omitempty"`
	Error string `json:"error,omitempty"`
}

type listOutput struct {
	Posts []listPostOutput `json:"posts"`
}

func buildListOutput(ctx context.Context, urls []string, withMetadata bool) (listOutput, error) {
	posts := make([]listPostOutput, len(urls))
	for i, url := range urls {
		posts[i] = listPostOutput{URL: url}
	}

	if !withMetadata {
		return listOutput{Posts: posts}, nil
	}
	if extractor == nil {
		return listOutput{}, fmt.Errorf("extractor not initialized")
	}

	byURL := make(map[string]listPostOutput, len(urls))
	for _, url := range urls {
		byURL[url] = listPostOutput{URL: url}
	}

	for result := range extractor.ExtractAllPosts(ctx, urls) {
		entry := byURL[result.Url]
		if result.Err != nil {
			entry.Error = result.Err.Error()
		} else {
			entry.Title = result.Post.Title
			entry.Date = result.Post.PostDate
		}
		byURL[result.Url] = entry
	}

	for i, url := range urls {
		if entry, ok := byURL[url]; ok {
			posts[i] = entry
		}
	}

	return listOutput{Posts: posts}, nil
}
