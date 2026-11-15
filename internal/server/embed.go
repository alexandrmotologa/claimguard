package server

import (
	"io"
	"net/http"
	"strings"

	"github.com/alexandrmotologa/claimguard/web"
)

// SPAHandler wraps an fs.FS to provide single-page application (SPA) routing fallback to index.html.
type SPAHandler struct {
	fileSystem http.FileSystem
}

// NewSPAHandler initializes an SPA static file handler.
func NewSPAHandler() (http.Handler, error) {
	distFS, err := web.DistFS()
	if err != nil {
		return nil, err
	}
	return &SPAHandler{fileSystem: http.FS(distFS)}, nil
}

func (h *SPAHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		path = "index.html"
	}

	// Try opening requested file
	f, err := h.fileSystem.Open(path)
	if err != nil {
		// Fallback to index.html for client-side routing
		indexFile, err := h.fileSystem.Open("index.html")
		if err != nil {
			http.Error(w, "Studio index.html not found", http.StatusNotFound)
			return
		}
		defer indexFile.Close()

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		io.Copy(w, indexFile)
		return
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil || stat.IsDir() {
		indexFile, err := h.fileSystem.Open("index.html")
		if err != nil {
			http.Error(w, "Studio index.html not found", http.StatusNotFound)
			return
		}
		defer indexFile.Close()

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		io.Copy(w, indexFile)
		return
	}

	http.FileServer(h.fileSystem).ServeHTTP(w, r)
}
