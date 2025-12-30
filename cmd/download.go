package cmd

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alexferrari88/sbstck-dl/lib"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
)

// downloadCmd represents the download command
var (
	downloadUrl    string
	format         string
	outputFolder   string
	dryRun         bool
	addSourceURL   bool
	downloadImages bool
	imageQuality   string
	imagesDir      string
	downloadFiles  bool
	fileExtensions string
	filesDir       string
	createArchive  bool
	downloadCmd    = &cobra.Command{
		Use:   "download",
		Short: "Download individual posts or the entire public archive",
		Long:  `You can provide the url of a single post or the main url of the Substack you want to download.`,
		Run: func(cmd *cobra.Command, args []string) {
			startTime := time.Now()

			manifestPath := filepath.Join(outputFolder, lib.ManifestFilename)
			manifest, err := lib.LoadManifest(manifestPath)
			if err != nil {
				log.Printf("Error loading manifest: %v\n", err)
				manifest = lib.NewManifest()
			}

			updateManifest := func(post lib.Post, path string, downloadedAt time.Time) {
				if manifest == nil {
					return
				}
				canonicalURL := post.CanonicalUrl
				if canonicalURL == "" {
					if verbose {
						log.Printf("Skipping manifest entry for %s: missing canonical URL\n", post.Slug)
					}
					return
				}
				if err := manifest.UpdateEntry(canonicalURL, path, outputFolder, format, downloadedAt); err != nil {
					log.Printf("Error updating manifest for %s: %v\n", canonicalURL, err)
					return
				}
				if err := manifest.Save(manifestPath); err != nil {
					log.Printf("Error saving manifest: %v\n", err)
				}
			}

			// Create archive instance if flag is set
			var archive *lib.Archive
			if createArchive {
				archive = lib.NewArchive()
			}

			// if url contains "/p/", we are downloading a single post
			if strings.Contains(downloadUrl, "/p/") {
				if verbose {
					fmt.Printf("Downloading post %s\n", downloadUrl)
				}
				if dryRun {
					fmt.Println("Dry run, exiting...")
					return
				}
				if (beforeDate != "" || afterDate != "") && verbose {
					fmt.Println("Warning: --before and --after flags are ignored when downloading a single post")
				}

				post, err := extractor.ExtractPost(ctx, downloadUrl)
				if err != nil {
					log.Fatalln(err)
				}
				downloadTime := time.Since(startTime)
				if verbose {
					fmt.Printf("Downloaded post %s in %s\n", downloadUrl, downloadTime)
				}

				path := makePath(post, outputFolder, format)
				if verbose {
					fmt.Printf("Writing post to file %s\n", path)
				}

				var writeErr error
				if downloadImages || downloadFiles {
					imageQualityEnum := lib.ImageQuality(imageQuality)
					// Parse file extensions if specified
					var fileExtensionsSlice []string
					if fileExtensions != "" {
						fileExtensionsSlice = strings.Split(strings.ReplaceAll(fileExtensions, " ", ""), ",")
					}
					imageResult, err := post.WriteToFileWithImages(ctx, path, format, addSourceURL, downloadImages, imageQualityEnum, imagesDir, downloadFiles, fileExtensionsSlice, filesDir, fetcher)
					writeErr = err
					if writeErr != nil {
						log.Printf("Error writing file %s: %v\n", path, writeErr)
					} else if verbose && imageResult.Success > 0 {
						fmt.Printf("Downloaded %d images (%d failed) for post %s\n", imageResult.Success, imageResult.Failed, post.Slug)
					}
				} else {
					writeErr = post.WriteToFile(path, format, addSourceURL)
					if writeErr != nil {
						log.Printf("Error writing file %s: %v\n", path, writeErr)
					}
				}

				if writeErr == nil {
					updateManifest(post, path, time.Now())
				}

				// Add to archive if enabled
				if archive != nil {
					archive.AddEntry(post, path, startTime)
				}

				if verbose {
					fmt.Println("Done in ", time.Since(startTime))
				}
			} else {
				// we are downloading the entire archive
				var downloadedPostsCount int
				dateFilterfunc := makeDateFilterFunc(beforeDate, afterDate)
				urls, err := extractor.GetAllPostsURLs(ctx, downloadUrl, dateFilterfunc)
				urlsCount := len(urls)
				if err != nil {
					log.Fatalln(err)
				}
				if urlsCount == 0 {
					if verbose {
						fmt.Println("No posts found, exiting...")
					}
					return
				}
				if verbose {
					fmt.Printf("Found %d posts\n", urlsCount)
				}
				if dryRun {
					fmt.Printf("Found %d posts\n", urlsCount)
					fmt.Println("Dry run, exiting...")
					return
				}
				urls, err = filterExistingPosts(urls, outputFolder, format)
				if err != nil {
					if verbose {
						fmt.Println("Error filtering existing posts:", err)
					}
				}
				if len(urls) == 0 {
					if verbose {
						fmt.Println("No new posts found, exiting...")
					}
					return
				}
				bar := progressbar.NewOptions(len(urls),
					progressbar.OptionSetWidth(25),
					progressbar.OptionSetDescription("downloading"),
					progressbar.OptionShowBytes(true))
				for result := range extractor.ExtractAllPosts(ctx, urls) {
					select {
					case <-ctx.Done():
						log.Fatalln("context cancelled")
					default:
					}
					if result.Err != nil {
						if verbose {
							fmt.Printf("Error downloading post %s: %s\n", result.Post.CanonicalUrl, result.Err)
							fmt.Println("Skipping...")
						}
						continue
					}
					bar.Add(1)
					downloadedPostsCount++
					if verbose {
						fmt.Printf("Downloading post %s\n", result.Post.CanonicalUrl)
					}
					post := result.Post

					path := makePath(post, outputFolder, format)
					if verbose {
						fmt.Printf("Writing post to file %s\n", path)
					}

					var writeErr error
					if downloadImages || downloadFiles {
						imageQualityEnum := lib.ImageQuality(imageQuality)
						// Parse file extensions if specified
						var fileExtensionsSlice []string
						if fileExtensions != "" {
							fileExtensionsSlice = strings.Split(strings.ReplaceAll(fileExtensions, " ", ""), ",")
						}
						imageResult, err := post.WriteToFileWithImages(ctx, path, format, addSourceURL, downloadImages, imageQualityEnum, imagesDir, downloadFiles, fileExtensionsSlice, filesDir, fetcher)
						writeErr = err
						if writeErr != nil {
							log.Printf("Error writing file %s: %v\n", path, writeErr)
						} else if verbose && imageResult.Success > 0 {
							fmt.Printf("Downloaded %d images (%d failed) for post %s\n", imageResult.Success, imageResult.Failed, post.Slug)
						}
					} else {
						writeErr = post.WriteToFile(path, format, addSourceURL)
						if writeErr != nil {
							log.Printf("Error writing file %s: %v\n", path, writeErr)
						}
					}

					if writeErr == nil {
						updateManifest(post, path, time.Now())
					}

					// Add to archive if enabled and post was successfully written
					if archive != nil {
						archive.AddEntry(post, path, time.Now())
					}
				}
				if verbose {
					fmt.Println("Downloaded", downloadedPostsCount, "posts, out of", len(urls))
					fmt.Println("Done in ", time.Since(startTime))
				}
			}

			// Generate archive page if enabled
			if archive != nil && len(archive.Entries) > 0 {
				if verbose {
					fmt.Printf("Generating archive page in %s format...\n", format)
				}

				var archiveErr error
				switch format {
				case "html":
					archiveErr = archive.GenerateHTML(outputFolder)
				case "md":
					archiveErr = archive.GenerateMarkdown(outputFolder)
				case "txt":
					archiveErr = archive.GenerateText(outputFolder)
				default:
					archiveErr = fmt.Errorf("unknown format for archive: %s", format)
				}

				if archiveErr != nil {
					log.Printf("Error generating archive page: %v\n", archiveErr)
				} else if verbose {
					fmt.Printf("Archive page generated: %s/index.%s\n", outputFolder, format)
				}
			}
		},
	}
)

func init() {
	downloadCmd.Flags().StringVarP(&downloadUrl, "url", "u", "", "Specify the Substack url")
	downloadCmd.Flags().StringVarP(&format, "format", "f", "html", "Specify the output format (options: \"html\", \"md\", \"txt\"")
	downloadCmd.Flags().StringVarP(&outputFolder, "output", "o", ".", "Specify the download directory")
	downloadCmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "Enable dry run")
	downloadCmd.Flags().BoolVar(&addSourceURL, "add-source-url", false, "Add the original post URL at the end of the downloaded file")
	downloadCmd.Flags().BoolVar(&downloadImages, "download-images", false, "Download images locally and update content to reference local files")
	downloadCmd.Flags().StringVar(&imageQuality, "image-quality", "high", "Image quality to download (options: \"high\", \"medium\", \"low\")")
	downloadCmd.Flags().StringVar(&imagesDir, "images-dir", "images", "Directory name for downloaded images")
	downloadCmd.Flags().BoolVar(&downloadFiles, "download-files", false, "Download file attachments locally and update content to reference local files")
	downloadCmd.Flags().StringVar(&fileExtensions, "file-extensions", "", "Comma-separated list of file extensions to download (e.g., 'pdf,docx,txt'). If empty, downloads all file types")
	downloadCmd.Flags().StringVar(&filesDir, "files-dir", "files", "Directory name for downloaded file attachments")
	downloadCmd.Flags().BoolVar(&createArchive, "create-archive", false, "Create an archive index page linking all downloaded posts")
	downloadCmd.MarkFlagRequired("url")
}

func convertDateTime(datetime string) string {
	// Parse the datetime string
	parsedTime, err := time.Parse(time.RFC3339, datetime)
	if err != nil {
		// Return an empty string or an error message if parsing fails
		return ""
	}

	// Format the datetime to the desired format
	formattedDateTime := fmt.Sprintf("%d%02d%02d_%02d%02d%02d",
		parsedTime.Year(), parsedTime.Month(), parsedTime.Day(),
		parsedTime.Hour(), parsedTime.Minute(), parsedTime.Second())

	return formattedDateTime
}

func parseURL(toTest string) (*url.URL, error) {
	_, err := url.ParseRequestURI(toTest)
	if err != nil {
		return nil, err
	}

	u, err := url.Parse(toTest)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return nil, err
	}

	return u, err
}

func makePath(post lib.Post, outputFolder string, format string) string {
	return fmt.Sprintf("%s/%s_%s.%s", outputFolder, convertDateTime(post.PostDate), post.Slug, format)
}

// extractSlug extracts the slug from a Substack post URL
// e.g. https://example.substack.com/p/this-is-the-post-title -> this-is-the-post-title
func extractSlug(url string) string {
	split := strings.Split(url, "/")
	return split[len(split)-1]
}

// filterExistingPosts filters out posts that already exist in the output folder.
// It prefers a manifest match by canonical URL, then falls back to slug matching.
func filterExistingPosts(urls []string, outputFolder string, format string) ([]string, error) {
	var firstErr error
	manifestPath := filepath.Join(outputFolder, lib.ManifestFilename)
	manifest, err := lib.LoadManifest(manifestPath)
	if err != nil {
		firstErr = err
		manifest = nil
	}

	slugIndex, err := indexExistingSlugs(outputFolder, format)
	if err != nil {
		if firstErr == nil {
			firstErr = err
		}
		slugIndex = map[string]struct{}{}
	}

	var filtered []string
	for _, url := range urls {
		if manifest != nil {
			if entry, ok := manifest.Entries[url]; ok {
				if entry.Format == "" || entry.Format == format {
					entryPath := filepath.FromSlash(entry.FilePath)
					if entryPath != "" && !filepath.IsAbs(entryPath) {
						entryPath = filepath.Join(outputFolder, entryPath)
					}
					if entryPath != "" {
						if _, statErr := os.Stat(entryPath); statErr == nil {
							continue
						}
					}
				}
			}
		}

		slug := extractSlug(url)
		if _, exists := slugIndex[slug]; !exists {
			filtered = append(filtered, url)
		}
	}
	return filtered, firstErr
}

func indexExistingSlugs(outputFolder string, format string) (map[string]struct{}, error) {
	entries, err := os.ReadDir(outputFolder)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]struct{}{}, nil
		}
		return nil, err
	}

	slugIndex := make(map[string]struct{})
	ext := "." + format
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ext) {
			continue
		}
		base := strings.TrimSuffix(name, ext)
		slug := slugFromFilename(base)
		if slug == "" {
			continue
		}
		slugIndex[slug] = struct{}{}
	}

	return slugIndex, nil
}

func slugFromFilename(base string) string {
	if strings.HasPrefix(base, "_") {
		if len(base) > 1 {
			return base[1:]
		}
		return ""
	}
	if len(base) > 16 && base[8] == '_' && base[15] == '_' && isDigits(base[:8]) && isDigits(base[9:15]) {
		return base[16:]
	}
	return ""
}

func isDigits(value string) bool {
	for i := 0; i < len(value); i++ {
		if value[i] < '0' || value[i] > '9' {
			return false
		}
	}
	return value != ""
}
