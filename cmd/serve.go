package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	servePort int
	serveCmd  = &cobra.Command{
		Use:   "serve",
		Short: "Launch a local-only web UI",
		Run: func(cmd *cobra.Command, args []string) {
			if servePort < 1 || servePort > 65535 {
				log.Fatalf("invalid --port %d (must be 1-65535)", servePort)
			}
			addr := fmt.Sprintf("127.0.0.1:%d", servePort)
			server := &http.Server{
				Addr:    addr,
				Handler: serveUIHandler(),
			}

			fmt.Printf("UI running at http://%s (Ctrl+C to stop)\n", addr)
			if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.Fatalf("UI server failed: %v", err)
			}
		},
	}
)

func init() {
	serveCmd.Flags().IntVar(&servePort, "port", 8787, "Port to bind the local UI server")
}

func serveUIHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", serveUIRoot)
	mux.HandleFunc("/api/test-connection", serveTestConnection)
	return mux
}

func serveUIRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, serveHTML)
}

type testConnectionRequest struct {
	PublicationURL string `json:"publication_url"`
	PrivateURL     string `json:"private_url"`
	CookieName     string `json:"cookie_name"`
	CookieVal      string `json:"cookie_val"`
	CookieValFile  string `json:"cookie_val_file"`
	CookieJar      string `json:"cookie_jar"`
	Proxy          string `json:"proxy"`
}

type testConnectionResponse struct {
	OK            bool     `json:"ok"`
	SitemapURL    string   `json:"sitemap_url"`
	SitemapStatus int      `json:"sitemap_status,omitempty"`
	SitemapOK     bool     `json:"sitemap_ok"`
	PrivateURL    string   `json:"private_url,omitempty"`
	PrivateStatus int      `json:"private_status,omitempty"`
	PrivateOK     bool     `json:"private_ok"`
	Errors        []string `json:"errors,omitempty"`
}

func serveTestConnection(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req testConnectionRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeTestResponse(w, testConnectionResponse{
			OK:     false,
			Errors: []string{fmt.Sprintf("Invalid request payload: %v", err)},
		})
		return
	}

	errorsList := make([]string, 0)
	pubURL, err := parseURL(strings.TrimSpace(req.PublicationURL))
	if err != nil || pubURL == nil {
		errorsList = append(errorsList, "Substack URL must be a valid http(s) URL.")
	}

	var proxyURL *url.URL
	if strings.TrimSpace(req.Proxy) != "" {
		parsedProxy, err := parseURL(strings.TrimSpace(req.Proxy))
		if err != nil {
			errorsList = append(errorsList, "Proxy URL must be a valid http(s) URL.")
		} else {
			proxyURL = parsedProxy
		}
	}

	domainHint := ""
	if pubURL != nil {
		domainHint = pubURL.Hostname()
	}

	cookie, cookieErrors := resolveTestCookie(req, domainHint)
	errorsList = append(errorsList, cookieErrors...)

	response := testConnectionResponse{
		SitemapURL: strings.TrimSpace(req.PublicationURL),
		PrivateURL: strings.TrimSpace(req.PrivateURL),
	}

	if pubURL != nil {
		response.SitemapURL = buildSitemapURL(pubURL)
	}

	client := newTestClient(proxyURL)

	if pubURL != nil {
		status, err := fetchStatus(client, response.SitemapURL, nil)
		if err != nil {
			errorsList = append(errorsList, fmt.Sprintf("Failed to fetch sitemap: %v", err))
		} else {
			response.SitemapStatus = status
			response.SitemapOK = status >= 200 && status < 300
			if !response.SitemapOK {
				errorsList = append(errorsList, fmt.Sprintf("Sitemap returned status %d.", status))
			}
		}
	}

	privateURL := strings.TrimSpace(req.PrivateURL)
	if privateURL != "" {
		if _, err := parseURL(privateURL); err != nil {
			errorsList = append(errorsList, "Private post URL must be a valid http(s) URL.")
		} else if cookie == nil {
			errorsList = append(errorsList, "Cookie name/value required to test a private post URL.")
		} else {
			status, err := fetchStatus(client, privateURL, cookie)
			if err != nil {
				errorsList = append(errorsList, fmt.Sprintf("Failed to fetch private post: %v", err))
			} else {
				response.PrivateStatus = status
				response.PrivateOK = status >= 200 && status < 300
				if !response.PrivateOK {
					errorsList = append(errorsList, fmt.Sprintf("Private post returned status %d.", status))
				}
			}
		}
	}

	response.Errors = errorsList
	response.OK = len(errorsList) == 0 && response.SitemapOK && (privateURL == "" || response.PrivateOK)
	writeTestResponse(w, response)
}

func writeTestResponse(w http.ResponseWriter, response testConnectionResponse) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

func resolveTestCookie(req testConnectionRequest, domainHint string) (*http.Cookie, []string) {
	errorsList := make([]string, 0)
	var cn cookieName
	if strings.TrimSpace(req.CookieName) != "" {
		if err := cn.Set(strings.TrimSpace(req.CookieName)); err != nil {
			errorsList = append(errorsList, err.Error())
		}
	}

	cookieVal, err := resolveCookieValue(strings.TrimSpace(req.CookieVal), strings.TrimSpace(req.CookieValFile))
	if err != nil {
		errorsList = append(errorsList, fmt.Sprintf("Failed to read cookie value: %v", err))
	}

	if cookieVal == "" && strings.TrimSpace(req.CookieJar) != "" {
		jarName, jarValue, err := readCookieFromJar(strings.TrimSpace(req.CookieJar), cn, domainHint)
		if err != nil {
			errorsList = append(errorsList, fmt.Sprintf("Failed to read cookie jar: %v", err))
		} else {
			if cn == "" {
				cn = jarName
			}
			cookieVal = jarValue
		}
	}

	if cn != "" && cookieVal == "" {
		errorsList = append(errorsList, "Cookie value is required when cookie name is set.")
	}
	if cn == "" && cookieVal != "" {
		errorsList = append(errorsList, "Cookie name is required when cookie value is set.")
	}

	if cn == "" || cookieVal == "" {
		return nil, errorsList
	}

	return &http.Cookie{
		Name:  cn.String(),
		Value: cookieVal,
	}, errorsList
}

func buildSitemapURL(pubURL *url.URL) string {
	normalized := *pubURL
	normalized.Path = ""
	normalized.RawQuery = ""
	normalized.Fragment = ""
	return normalized.ResolveReference(&url.URL{Path: "/sitemap.xml"}).String()
}

func newTestClient(proxyURL *url.URL) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if proxyURL != nil {
		transport.Proxy = http.ProxyURL(proxyURL)
	}
	return &http.Client{
		Transport: transport,
		Timeout:   15 * time.Second,
	}
}

func fetchStatus(client *http.Client, targetURL string, cookie *http.Cookie) (int, error) {
	req, err := http.NewRequest(http.MethodGet, targetURL, nil)
	if err != nil {
		return 0, err
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	return resp.StatusCode, nil
}

const serveHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Substack Downloader Wizard</title>
  <style>
    :root {
      color-scheme: light;
      --bg: #fdf6e3;
      --bg-2: #f8fafc;
      --card: #ffffff;
      --text: #0f172a;
      --muted: #475569;
      --accent: #f97316;
      --accent-dark: #b45309;
      --border: #e2e8f0;
      --ink: #1f2937;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      font-family: "Garamond", "Georgia", "Times New Roman", serif;
      background: radial-gradient(circle at 20% 10%, #fff7ed 0%, rgba(255, 247, 237, 0) 45%),
        linear-gradient(135deg, var(--bg-2) 0%, var(--bg) 100%);
      color: var(--text);
      min-height: 100vh;
    }
    .scene {
      max-width: 980px;
      margin: 0 auto;
      padding: 28px 18px 48px;
    }
    .hero {
      display: grid;
      gap: 8px;
      margin-bottom: 18px;
    }
    .pill {
      display: inline-flex;
      align-items: center;
      gap: 6px;
      padding: 6px 10px;
      border-radius: 999px;
      border: 1px solid #fed7aa;
      background: #fff7ed;
      color: var(--accent-dark);
      font-size: 12px;
      letter-spacing: 0.08em;
      text-transform: uppercase;
      width: fit-content;
    }
    h1 { margin: 0; font-size: 26px; letter-spacing: -0.02em; color: var(--ink); }
    p { margin: 0; line-height: 1.5; color: var(--muted); }
    .card {
      background: var(--card);
      border: 1px solid var(--border);
      border-radius: 16px;
      padding: 22px;
      box-shadow: 0 14px 40px rgba(15, 23, 42, 0.08);
      animation: lift 500ms ease-out;
    }
    .stepper {
      display: grid;
      grid-template-columns: repeat(3, minmax(0, 1fr));
      gap: 8px;
      margin-bottom: 16px;
    }
    .step-pill {
      border: 1px solid var(--border);
      background: #f8fafc;
      border-radius: 12px;
      padding: 10px 12px;
      font-size: 13px;
      text-align: left;
      cursor: pointer;
      transition: 0.2s ease;
      color: var(--muted);
    }
    .step-pill.active {
      border-color: #fdba74;
      background: #fff7ed;
      color: var(--ink);
      font-weight: 600;
    }
    .step-pill span {
      display: block;
      font-size: 12px;
      text-transform: uppercase;
      letter-spacing: 0.1em;
      color: var(--accent-dark);
    }
    .step-panel { display: none; }
    .step-panel.active { display: block; }
    .group {
      margin-top: 18px;
      padding-top: 14px;
      border-top: 1px dashed #e2e8f0;
    }
    .group:first-of-type {
      border-top: none;
      padding-top: 0;
      margin-top: 0;
    }
    .group h3 {
      margin: 0 0 8px;
      font-size: 16px;
      color: var(--ink);
    }
    .help {
      font-size: 12px;
      color: #64748b;
    }
    .tip {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      width: 16px;
      height: 16px;
      border-radius: 999px;
      border: 1px solid #fed7aa;
      background: #fff7ed;
      color: var(--accent-dark);
      font-size: 11px;
      margin-left: 6px;
      cursor: help;
    }
    .error {
      display: block;
      min-height: 16px;
      font-size: 12px;
      color: #b91c1c;
      margin-top: 4px;
    }
    .error-list {
      margin: 10px 0 0;
      padding-left: 18px;
      color: #b91c1c;
      font-size: 13px;
    }
    .test-status {
      margin-top: 10px;
      font-size: 13px;
      color: var(--muted);
    }
    .test-status.ok { color: #166534; }
    .test-status.bad { color: #b91c1c; }
    .field.invalid input,
    .field.invalid select,
    .field.invalid textarea {
      border-color: #fca5a5;
      box-shadow: 0 0 0 1px rgba(248, 113, 113, 0.2);
    }
    .grid {
      display: grid;
      gap: 12px;
      grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
    }
    label.field {
      display: grid;
      gap: 6px;
      font-size: 13px;
      color: var(--muted);
    }
    label.field input,
    label.field select,
    label.field textarea {
      width: 100%;
      padding: 10px 12px;
      border: 1px solid var(--border);
      border-radius: 10px;
      font-size: 14px;
      font-family: "Garamond", "Georgia", "Times New Roman", serif;
      color: var(--ink);
      background: #ffffff;
    }
    label.field input[type="checkbox"] {
      width: auto;
      margin-right: 8px;
    }
    .flag {
      display: inline-block;
      margin-left: 6px;
      padding: 2px 6px;
      border-radius: 6px;
      border: 1px solid #e2e8f0;
      background: #f8fafc;
      font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", monospace;
      font-size: 11px;
      color: #475569;
    }
    .preset-grid {
      display: grid;
      gap: 12px;
      grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
    }
    .preset-card {
      border: 1px solid var(--border);
      border-radius: 14px;
      padding: 16px;
      display: grid;
      gap: 6px;
      cursor: pointer;
      background: #fffaf5;
      transition: 0.2s ease;
    }
    .preset-card input { margin-right: 8px; }
    .preset-card:hover { border-color: #fdba74; }
    .preset-title { font-size: 16px; font-weight: 600; color: var(--ink); }
    .preset-desc { font-size: 13px; color: var(--muted); }
    .preset-meta { font-size: 12px; color: #64748b; }
    .advanced-only.is-hidden { display: none; }
    .toggle-row {
      display: flex;
      align-items: center;
      gap: 10px;
      padding: 8px 12px;
      border-radius: 10px;
      border: 1px solid var(--border);
      background: #ffffff;
      font-size: 13px;
      color: var(--muted);
    }
    .nav {
      margin-top: 18px;
      display: flex;
      justify-content: space-between;
      gap: 12px;
    }
    .nav button {
      padding: 10px 16px;
      border-radius: 10px;
      border: 1px solid var(--border);
      background: #ffffff;
      font-size: 14px;
      cursor: pointer;
      transition: 0.2s ease;
    }
    .nav button.primary {
      border-color: #fdba74;
      background: #fff7ed;
      color: var(--ink);
      font-weight: 600;
    }
    button.primary {
      border-color: #fdba74;
      background: #fff7ed;
      color: var(--ink);
      font-weight: 600;
    }
    .nav button:disabled { opacity: 0.5; cursor: not-allowed; }
    .review {
      display: grid;
      gap: 12px;
    }
    pre {
      background: #0f172a;
      color: #f8fafc;
      padding: 14px;
      border-radius: 12px;
      overflow-x: auto;
      font-size: 13px;
      line-height: 1.4;
    }
    .review-actions {
      display: flex;
      align-items: center;
      gap: 12px;
      flex-wrap: wrap;
    }
    .review-actions button {
      border: 1px solid #fdba74;
      background: #fff7ed;
      padding: 8px 14px;
      border-radius: 10px;
      font-size: 13px;
      cursor: pointer;
    }
    .hint {
      padding: 10px 12px;
      border-left: 3px solid var(--accent);
      background: #fff7ed;
      color: #7c2d12;
      border-radius: 8px;
      font-size: 13px;
    }
    .small { font-size: 12px; color: #64748b; }
    @media (max-width: 720px) {
      .scene { padding: 22px 16px 40px; }
      .stepper { grid-template-columns: 1fr; }
      .nav { flex-direction: column-reverse; align-items: stretch; }
    }
    @keyframes lift {
      from { opacity: 0; transform: translateY(8px); }
      to { opacity: 1; transform: translateY(0); }
    }
  </style>
</head>
<body>
  <div class="scene">
    <div class="hero">
      <span class="pill">Local only</span>
      <h1>Substack Downloader Wizard</h1>
      <p>Build a download command in three steps. Every field maps directly to a CLI flag.</p>
    </div>

    <div class="card">
      <div class="stepper">
        <button class="step-pill active" data-step-pill="1" type="button"><span>Step 1</span>Preset</button>
        <button class="step-pill" data-step-pill="2" type="button"><span>Step 2</span>Configure</button>
        <button class="step-pill" data-step-pill="3" type="button"><span>Step 3</span>Review</button>
      </div>

      <form id="wizardForm">
        <section class="step-panel active" data-step="1">
          <h2>Choose a preset</h2>
          <p class="small">Basic keeps only the essentials. Advanced exposes every download flag.</p>
          <div class="preset-grid">
            <label class="preset-card">
              <div>
                <input type="radio" name="preset" value="basic" checked />
                <span class="preset-title">Basic</span>
              </div>
              <span class="preset-desc">Quick download with archive and assets.</span>
              <span class="preset-meta">URL, output, format, archive, assets, rate, workers.</span>
            </label>
            <label class="preset-card">
              <div>
                <input type="radio" name="preset" value="advanced" />
                <span class="preset-title">Advanced</span>
              </div>
              <span class="preset-desc">Full control over filters, retries, and auth.</span>
              <span class="preset-meta">Includes cookies, layout, metadata, and safety flags.</span>
            </label>
          </div>
        </section>

        <section class="step-panel" data-step="2">
          <h2>Configure the download</h2>
          <p class="small">Recommended defaults: format html, rate 2, max-workers 10. Enable images/files for offline archives.</p>

          <div class="group">
            <h3>Core</h3>
            <div class="grid">
              <label class="field">
                Substack URL <span class="flag">--url</span><span class="tip" title="Paste a publication URL or a single post URL.">?</span>
                <input id="url" data-flag="--url" type="text" placeholder="https://example.substack.com" />
                <span class="help">Example: https://example.substack.com or https://example.substack.com/p/post-title</span>
                <span class="error" data-error-for="url"></span>
              </label>
              <label class="field">
                Output directory <span class="flag">--output</span><span class="tip" title="Where downloaded files will be saved.">?</span>
                <input id="output" data-flag="--output" type="text" placeholder="." />
                <span class="help">Use "." for the current folder.</span>
              </label>
              <label class="field">
                Format <span class="flag">--format</span><span class="tip" title="HTML is recommended for full fidelity.">?</span>
                <select id="format" data-flag="--format" data-default="html">
                  <option value="html" selected>html</option>
                  <option value="md">md</option>
                  <option value="txt">txt</option>
                </select>
                <span class="help">Recommended: html.</span>
              </label>
            </div>
          </div>

          <div class="group">
            <h3>Archive and assets</h3>
            <div class="grid">
              <label class="toggle-row">
                <input id="createArchive" data-flag="--create-archive" type="checkbox" />
                Create archive index <span class="flag">--create-archive</span>
              </label>
              <label class="toggle-row">
                <input id="downloadImages" data-flag="--download-images" type="checkbox" />
                Download images locally <span class="flag">--download-images</span><span class="tip" title="Recommended for offline browsing.">?</span>
              </label>
              <label class="toggle-row">
                <input id="downloadFiles" data-flag="--download-files" type="checkbox" />
                Download files locally <span class="flag">--download-files</span><span class="tip" title="Recommended if posts include attachments.">?</span>
              </label>
              <label class="field">
                Image quality <span class="flag">--image-quality</span><span class="tip" title="High is best quality but larger downloads.">?</span>
                <select id="imageQuality" data-flag="--image-quality" data-default="high" data-requires="downloadImages">
                  <option value="high" selected>high</option>
                  <option value="medium">medium</option>
                  <option value="low">low</option>
                </select>
                <span class="help">Recommended: high.</span>
              </label>
              <label class="field">
                Images directory <span class="flag">--images-dir</span>
                <input id="imagesDir" data-flag="--images-dir" data-default="images" data-requires="downloadImages" type="text" value="images" />
              </label>
              <label class="field">
                Files directory <span class="flag">--files-dir</span>
                <input id="filesDir" data-flag="--files-dir" data-default="files" data-requires="downloadFiles" type="text" value="files" />
              </label>
              <label class="field">
                File extensions <span class="flag">--file-extensions</span>
                <input id="fileExtensions" data-flag="--file-extensions" data-requires="downloadFiles" type="text" placeholder="pdf,docx" />
              </label>
            </div>
          </div>

          <div class="group advanced-only" data-advanced="true">
            <h3>Filters and layout</h3>
            <div class="grid">
              <label class="field">
                After date <span class="flag">--after</span>
                <input id="afterDate" data-flag="--after" type="text" placeholder="YYYY-MM-DD" />
                <span class="error" data-error-for="afterDate"></span>
              </label>
              <label class="field">
                Before date <span class="flag">--before</span>
                <input id="beforeDate" data-flag="--before" type="text" placeholder="YYYY-MM-DD" />
                <span class="error" data-error-for="beforeDate"></span>
              </label>
              <label class="field">
                Layout <span class="flag">--layout</span>
                <select id="layout" data-flag="--layout" data-default="flat">
                  <option value="flat" selected>flat</option>
                  <option value="year/month">year/month</option>
                  <option value="year/slug">year/slug</option>
                </select>
              </label>
              <label class="toggle-row">
                <input id="writeMetadata" data-flag="--write-metadata" type="checkbox" />
                Write metadata sidecars <span class="flag">--write-metadata</span>
              </label>
              <label class="toggle-row">
                <input id="addSourceUrl" data-flag="--add-source-url" type="checkbox" />
                Append source URL <span class="flag">--add-source-url</span>
              </label>
            </div>
          </div>

          <div class="group advanced-only" data-advanced="true">
            <h3>Run behavior</h3>
            <div class="grid">
              <label class="toggle-row">
                <input id="dryRun" data-flag="--dry-run" type="checkbox" />
                Dry run <span class="flag">--dry-run</span>
              </label>
              <label class="toggle-row">
                <input id="forceDownload" data-flag="--force" type="checkbox" data-exclusive="existing" />
                Force redownload <span class="flag">--force</span>
              </label>
              <label class="toggle-row">
                <input id="skipExisting" data-flag="--skip-existing" type="checkbox" data-exclusive="existing" />
                Skip existing <span class="flag">--skip-existing</span>
              </label>
              <label class="toggle-row">
                <input id="refreshUpdated" data-flag="--refresh-updated" type="checkbox" />
                Refresh updated posts <span class="flag">--refresh-updated</span>
              </label>
              <label class="toggle-row">
                <input id="failFast" data-flag="--fail-fast" type="checkbox" data-exclusive="errors" />
                Fail fast <span class="flag">--fail-fast</span>
              </label>
              <label class="toggle-row">
                <input id="continueOnError" data-flag="--continue-on-error" type="checkbox" data-exclusive="errors" />
                Continue on error <span class="flag">--continue-on-error</span>
              </label>
            </div>
          </div>

          <div class="group">
            <h3>Performance and logging</h3>
            <div class="grid">
              <label class="field">
                Rate per second <span class="flag">--rate</span><span class="tip" title="Lower rates reduce server load.">?</span>
                <input id="rate" data-flag="--rate" data-default="2" type="text" value="2" />
                <span class="help">Recommended: 2.</span>
              </label>
              <label class="field">
                Max workers <span class="flag">--max-workers</span><span class="tip" title="More workers increase parallel downloads.">?</span>
                <input id="maxWorkers" data-flag="--max-workers" data-default="10" type="text" value="10" />
                <span class="help">Recommended: 10.</span>
              </label>
              <label class="field advanced-only" data-advanced="true">
                Proxy URL <span class="flag">--proxy</span>
                <input id="proxy" data-flag="--proxy" type="text" placeholder="http://localhost:8080" />
                <span class="error" data-error-for="proxy"></span>
              </label>
              <label class="field advanced-only" data-advanced="true">
                Log format <span class="flag">--log-format</span>
                <select id="logFormat" data-flag="--log-format" data-default="text">
                  <option value="text" selected>text</option>
                  <option value="json">json</option>
                </select>
              </label>
              <label class="toggle-row advanced-only" data-advanced="true">
                <input id="verbose" data-flag="--verbose" type="checkbox" />
                Verbose output <span class="flag">--verbose</span>
              </label>
            </div>
          </div>

          <div class="group advanced-only" data-advanced="true">
            <h3>Private newsletter auth</h3>
            <div class="grid">
              <label class="field">
                Cookie name <span class="flag">--cookie_name</span>
                <select id="cookieName" data-flag="--cookie_name">
                  <option value="" selected>none</option>
                  <option value="substack.sid">substack.sid</option>
                  <option value="connect.sid">connect.sid</option>
                </select>
              </label>
              <label class="field">
                Cookie value <span class="flag">--cookie_val</span>
                <input id="cookieVal" data-flag="--cookie_val" type="text" placeholder="substack.sid value" />
              </label>
              <label class="field">
                Cookie value file <span class="flag">--cookie-val-file</span>
                <input id="cookieValFile" data-flag="--cookie-val-file" type="text" placeholder="path/to/cookie.txt" />
              </label>
              <label class="field">
                Cookie jar <span class="flag">--cookie-jar</span>
                <input id="cookieJar" data-flag="--cookie-jar" type="text" placeholder="path/to/cookies.txt" />
              </label>
              <label class="field">
                Notion labels <span class="flag">--notion-labels</span>
                <input id="notionLabels" data-flag="--notion-labels" type="text" placeholder="path/to/notion-labels.yaml" />
              </label>
            </div>
          </div>

          <div class="group">
            <h3>Test connection</h3>
            <p class="small">Fetches sitemap.xml and optionally tests a private post URL using your cookie.</p>
            <div class="grid">
              <label class="field">
                Private post URL
                <input id="privateUrl" type="text" placeholder="https://example.substack.com/p/private-post" />
                <span class="help">Optional. Requires cookie fields above.</span>
                <span class="error" data-error-for="privateUrl"></span>
              </label>
              <div>
                <button type="button" id="testBtn" class="primary">Test connection</button>
                <div id="testStatus" class="test-status">Run a test to verify access.</div>
              </div>
            </div>
          </div>
        </section>

        <section class="step-panel" data-step="3">
          <div class="review">
            <h2>Review command</h2>
            <p>Copy and run this command in your terminal.</p>
            <pre id="commandPreview">sbstck-dl download</pre>
            <div class="review-actions">
              <button type="button" id="copyBtn">Copy command</button>
              <span id="copyStatus" class="small"></span>
            </div>
            <div id="reviewStatus" class="hint">Tip: add a URL to target a single post or a publication.</div>
            <ul id="validationErrors" class="error-list" hidden></ul>
          </div>
        </section>
      </form>

      <div class="nav">
        <button type="button" id="backBtn" disabled>Back</button>
        <button type="button" id="nextBtn" class="primary">Continue</button>
      </div>
    </div>

    <div class="hint" style="margin-top: 18px;">
      This UI binds to 127.0.0.1 only. Share the generated command with your terminal to run it.
    </div>
  </div>

  <script>
    (function () {
      const stepPanels = Array.from(document.querySelectorAll('.step-panel'));
      const stepPills = Array.from(document.querySelectorAll('[data-step-pill]'));
      const backBtn = document.getElementById('backBtn');
      const nextBtn = document.getElementById('nextBtn');
      const form = document.getElementById('wizardForm');
      const commandPreview = document.getElementById('commandPreview');
      const copyBtn = document.getElementById('copyBtn');
      const copyStatus = document.getElementById('copyStatus');
      const reviewStatus = document.getElementById('reviewStatus');
      const errorList = document.getElementById('validationErrors');
      const presetInputs = Array.from(document.querySelectorAll('input[name="preset"]'));
      const advancedGroups = Array.from(document.querySelectorAll('[data-advanced="true"]'));
      const errorTargets = {};
      document.querySelectorAll('[data-error-for]').forEach(el => {
        errorTargets[el.dataset.errorFor] = el;
      });
      const urlInput = document.getElementById('url');
      const privateUrlInput = document.getElementById('privateUrl');
      const proxyInput = document.getElementById('proxy');
      const afterDateInput = document.getElementById('afterDate');
      const beforeDateInput = document.getElementById('beforeDate');
      const testBtn = document.getElementById('testBtn');
      const testStatus = document.getElementById('testStatus');
      const cookieNameInput = document.getElementById('cookieName');
      const cookieValInput = document.getElementById('cookieVal');
      const cookieValFileInput = document.getElementById('cookieValFile');
      const cookieJarInput = document.getElementById('cookieJar');
      let currentStep = 1;
      let activePreset = 'basic';

      function showStep(step) {
        currentStep = step;
        stepPanels.forEach(panel => {
          panel.classList.toggle('active', Number(panel.dataset.step) === step);
        });
        stepPills.forEach(pill => {
          pill.classList.toggle('active', Number(pill.dataset.stepPill) === step);
        });
        backBtn.disabled = step === 1;
        if (step === 3) {
          nextBtn.textContent = 'Done';
          nextBtn.disabled = true;
        } else {
          nextBtn.textContent = step === 2 ? 'Review' : 'Continue';
          nextBtn.disabled = false;
        }
        if (step === 3) {
          buildCommand();
        }
      }

      function toggleAdvanced(enabled) {
        advancedGroups.forEach(group => {
          group.classList.toggle('is-hidden', !enabled);
          group.querySelectorAll('input, select, textarea').forEach(el => {
            el.disabled = !enabled;
          });
        });
      }

      function quote(value) {
        if (/[\s"]/g.test(value)) {
          return '"' + value.replace(/"/g, '\\"') + '"';
        }
        return value;
      }

      function setError(key, message) {
        const target = errorTargets[key];
        if (!target) return;
        target.textContent = message || '';
        const field = target.closest('.field');
        if (field) field.classList.toggle('invalid', Boolean(message));
      }

      function validateURLField(value, label) {
        if (!value) return '';
        try {
          const parsed = new URL(value);
          if (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') {
            return label + ' must start with http:// or https://.';
          }
          return '';
        } catch (err) {
          return label + ' must be a valid URL (include http:// or https://).';
        }
      }

      function validateDateField(value, label) {
        if (!value) return '';
        if (!/^\d{4}-\d{2}-\d{2}$/.test(value)) {
          return label + ' must use YYYY-MM-DD (e.g. 2023-08-15).';
        }
        const parsed = Date.parse(value + 'T00:00:00');
        if (!Number.isFinite(parsed)) {
          return label + ' must be a valid calendar date.';
        }
        return '';
      }

      function validateAndRender() {
        const errors = [];
        const urlError = validateURLField((urlInput && !urlInput.disabled) ? urlInput.value.trim() : '', 'Substack URL');
        setError('url', urlError);
        if (urlError) errors.push(urlError);

        const proxyValue = (proxyInput && !proxyInput.disabled) ? proxyInput.value.trim() : '';
        const proxyError = validateURLField(proxyValue, 'Proxy URL');
        setError('proxy', proxyError);
        if (proxyError) errors.push(proxyError);

        const privateValue = (privateUrlInput && !privateUrlInput.disabled) ? privateUrlInput.value.trim() : '';
        const privateError = validateURLField(privateValue, 'Private post URL');
        setError('privateUrl', privateError);
        if (privateError) errors.push(privateError);

        const afterValue = (afterDateInput && !afterDateInput.disabled) ? afterDateInput.value.trim() : '';
        const afterError = validateDateField(afterValue, 'After date');
        setError('afterDate', afterError);
        if (afterError) errors.push(afterError);

        const beforeValue = (beforeDateInput && !beforeDateInput.disabled) ? beforeDateInput.value.trim() : '';
        const beforeError = validateDateField(beforeValue, 'Before date');
        setError('beforeDate', beforeError);
        if (beforeError) errors.push(beforeError);

        if (errorList) {
          if (errors.length) {
            errorList.hidden = false;
            errorList.innerHTML = errors.map(err => '<li>' + err + '</li>').join('');
          } else {
            errorList.hidden = true;
            errorList.innerHTML = '';
          }
        }
        return errors;
      }

      function buildCommand() {
        const parts = ['sbstck-dl', 'download'];
        const elements = Array.from(form.querySelectorAll('[data-flag]'));

        elements.forEach(el => {
          if (el.disabled) return;
          const flag = el.dataset.flag;
          if (el.dataset.requires) {
            const req = document.getElementById(el.dataset.requires);
            if (req && !req.checked) return;
          }
          if (el.type === 'checkbox') {
            if (el.checked) parts.push(flag);
            return;
          }
          if (el.type === 'radio') {
            if (el.checked) parts.push(flag);
            return;
          }
          const value = (el.value || '').trim();
          if (value === '') return;
          if (el.dataset.default && el.dataset.default === value) return;
          parts.push(flag, quote(value));
        });

        commandPreview.textContent = parts.join(' ');

        const errors = validateAndRender();
        const urlValue = (urlInput ? urlInput.value : '').trim();
        if (errors.length) {
          reviewStatus.textContent = 'Fix the highlighted fields to build a valid command.';
        } else if (!urlValue) {
          reviewStatus.textContent = 'Add --url to target a single post or a publication.';
        } else {
          reviewStatus.textContent = 'Ready to run. Paste the command into your terminal.';
        }
      }

      function handleExclusiveGroups(target) {
        const group = target.dataset.exclusive;
        if (!group || !target.checked) return;
        document.querySelectorAll('input[data-exclusive="' + group + '"]').forEach(el => {
          if (el !== target) el.checked = false;
        });
      }

      form.addEventListener('input', function (event) {
        const target = event.target;
        if (target && target.dataset && target.dataset.exclusive) {
          handleExclusiveGroups(target);
        }
        buildCommand();
      });

      stepPills.forEach(pill => {
        pill.addEventListener('click', function () {
          showStep(Number(pill.dataset.stepPill));
        });
      });

      presetInputs.forEach(input => {
        input.addEventListener('change', function () {
          if (input.checked) {
            activePreset = input.value;
            toggleAdvanced(activePreset === 'advanced');
            buildCommand();
          }
        });
      });

      backBtn.addEventListener('click', function () {
        if (currentStep > 1) showStep(currentStep - 1);
      });

      nextBtn.addEventListener('click', function () {
        if (currentStep < 3) showStep(currentStep + 1);
      });

      if (copyBtn) {
        copyBtn.addEventListener('click', function () {
          const text = commandPreview.textContent;
          if (!text) return;
          if (navigator.clipboard && navigator.clipboard.writeText) {
            navigator.clipboard.writeText(text).then(() => {
              copyStatus.textContent = 'Copied to clipboard.';
            }).catch(() => {
              copyStatus.textContent = 'Copy failed.';
            });
          } else {
            copyStatus.textContent = 'Copy not supported.';
          }
        });
      }

      if (testBtn) {
        testBtn.addEventListener('click', function () {
          const errors = validateAndRender();
          const urlValue = (urlInput ? urlInput.value : '').trim();
          if (errors.length) {
            if (testStatus) {
              testStatus.textContent = 'Fix the highlighted fields before testing.';
              testStatus.className = 'test-status bad';
            }
            return;
          }
          if (!urlValue) {
            if (testStatus) {
              testStatus.textContent = 'Add a Substack URL before testing.';
              testStatus.className = 'test-status bad';
            }
            return;
          }

          const payload = {
            publication_url: urlValue,
            private_url: privateUrlInput ? privateUrlInput.value.trim() : '',
            cookie_name: (cookieNameInput && !cookieNameInput.disabled) ? cookieNameInput.value : '',
            cookie_val: (cookieValInput && !cookieValInput.disabled) ? cookieValInput.value : '',
            cookie_val_file: (cookieValFileInput && !cookieValFileInput.disabled) ? cookieValFileInput.value : '',
            cookie_jar: (cookieJarInput && !cookieJarInput.disabled) ? cookieJarInput.value : '',
            proxy: (proxyInput && !proxyInput.disabled) ? proxyInput.value.trim() : ''
          };

          testBtn.disabled = true;
          if (testStatus) {
            testStatus.textContent = 'Testing...';
            testStatus.className = 'test-status';
          }

          fetch('/api/test-connection', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload)
          }).then(res => res.json()).then(data => {
            if (!testStatus) return;
            const parts = [];
            if (data.sitemap_ok) {
              parts.push('Sitemap OK (' + data.sitemap_status + ')');
            } else if (data.sitemap_status) {
              parts.push('Sitemap failed (' + data.sitemap_status + ')');
            }
            if (data.private_url) {
              if (data.private_ok) {
                parts.push('Private post OK (' + data.private_status + ')');
              } else if (data.private_status) {
                parts.push('Private post failed (' + data.private_status + ')');
              }
            }
            if (data.errors && data.errors.length) {
              parts.push(data.errors.join(' | '));
            }
            testStatus.textContent = parts.join(' | ') || 'No response details.';
            testStatus.className = 'test-status ' + (data.ok ? 'ok' : 'bad');
          }).catch(err => {
            if (!testStatus) return;
            testStatus.textContent = 'Test failed: ' + err;
            testStatus.className = 'test-status bad';
          }).finally(() => {
            testBtn.disabled = false;
          });
        });
      }

      toggleAdvanced(false);
      buildCommand();
      showStep(1);
    })();
  </script>
</body>
</html>`
