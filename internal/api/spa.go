package api

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/ru-ace/nm-webui/internal/webui"
)

// spaHandler serves the embedded SPA with a client-side-router friendly
// fallback to index.html. /api routes are handled before this is mounted.
func (s *Server) spaHandler() http.Handler {
	fsys, err := fs.Sub(webui.Dist, webui.DistDir)
	if err != nil {
		panic("webui dist not embedded: " + err.Error())
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			writeErr(w, http.StatusNotFound, "endpoint not found")
			return
		}
		path := strings.TrimPrefix(r.URL.Path, "/")
		if r.URL.Path != "/" {
			if strings.Contains(r.URL.Path, "..") {
				http.NotFound(w, r)
				return
			}
			if f, err := fsys.Open(path); err == nil {
				_ = f.Close()
				w.Header().Set("Cache-Control", "no-cache")
				http.ServeFileFS(w, r, fsys, path)
				return
			}
		}
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFileFS(w, r, fsys, "index.html")
	})
}