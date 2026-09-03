// Package docsapi serves the panel's documentation site, embedded directly
// into the panel binary. Embedding it here — rather than a static file an
// operator has to scp onto the VDS by hand — means the docs update the same
// way everything else does: push to master, CI builds a new panel image,
// Watchtower deploys it. There is no separate docs deploy step to forget.
package docsapi

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed all:site
var siteFS embed.FS

// RegisterRoutes serves the docs site at /docs/.
func RegisterRoutes(mux *http.ServeMux) {
	sub, err := fs.Sub(siteFS, "site")
	if err != nil {
		panic(err) // embedded at build time; cannot fail at runtime
	}
	mux.Handle("GET /docs/", http.StripPrefix("/docs/", http.FileServerFS(sub)))
	mux.HandleFunc("GET /docs", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/docs/", http.StatusFound)
	})
}
