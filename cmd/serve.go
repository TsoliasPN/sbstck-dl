package cmd

import (
	"errors"
	"fmt"
	"log"
	"net/http"

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

const serveHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Substack Downloader UI</title>
  <style>
    :root {
      color-scheme: light;
      --bg: #fef3c7;
      --bg-2: #f8fafc;
      --card: #ffffff;
      --text: #0f172a;
      --muted: #475569;
      --accent: #f97316;
      --border: #e2e8f0;
    }
    body {
      margin: 0;
      font-family: "Palatino Linotype", "Book Antiqua", Palatino, serif;
      background: linear-gradient(135deg, var(--bg-2) 0%, var(--bg) 100%);
      color: var(--text);
    }
    .container {
      max-width: 720px;
      margin: 0 auto;
      padding: 32px 20px 40px;
    }
    .card {
      background: var(--card);
      border: 1px solid var(--border);
      border-radius: 14px;
      padding: 20px;
      box-shadow: 0 8px 22px rgba(15, 23, 42, 0.08);
      animation: lift 480ms ease-out;
    }
    h1 { margin: 0 0 8px; font-size: 22px; }
    p { margin: 0 0 12px; line-height: 1.5; color: var(--muted); }
    code {
      display: inline-block;
      background: #fff7ed;
      border: 1px solid #fed7aa;
      border-radius: 6px;
      padding: 2px 6px;
      font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", monospace;
      font-size: 12px;
    }
    .hint {
      margin-top: 12px;
      padding: 10px 12px;
      border-left: 3px solid var(--accent);
      background: #fff7ed;
      color: #7c2d12;
      border-radius: 8px;
      font-size: 13px;
    }
    @keyframes lift {
      from { opacity: 0; transform: translateY(6px); }
      to { opacity: 1; transform: translateY(0); }
    }
  </style>
</head>
<body>
  <div class="container">
    <div class="card">
      <h1>Substack Downloader UI</h1>
      <p>This local-only UI is a starting point for guided runs. More guided steps will appear here soon.</p>
      <p>For now, use the CLI directly:</p>
      <p><code>sbstck-dl download --url https://example.substack.com --create-archive</code></p>
      <p class="hint">Tip: this server only binds to <strong>127.0.0.1</strong> for safety.</p>
    </div>
  </div>
</body>
</html>`
