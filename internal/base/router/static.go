package router

import (
	"io/fs"
	"net/http"
	"strings"

	webui "github.com/cccvno1/ledger-agent/web"
)

// registerStatic serves the Vue SPA from the embedded filesystem.
// Any path not matching a real file falls back to index.html so that
// Vue Router can handle client-side navigation.
func registerStatic(mux *http.ServeMux) {
	distFS, err := fs.Sub(webui.Files, "dist")
	if err != nil {
		panic("webui: dist subtree not found: " + err.Error())
	}

	fileServer := http.FileServer(http.FS(distFS))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		urlPath := strings.TrimPrefix(r.URL.Path, "/")
		if urlPath == "" {
			urlPath = "index.html"
		}

		f, err := distFS.Open(urlPath)
		if err != nil {
			// Path not found → SPA fallback so Vue Router takes over.
			http.ServeFileFS(w, r, distFS, "index.html")
			return
		}
		defer f.Close()

		stat, err := f.Stat()
		if err != nil || stat.IsDir() {
			http.ServeFileFS(w, r, distFS, "index.html")
			return
		}

		fileServer.ServeHTTP(w, r)
	})
}
