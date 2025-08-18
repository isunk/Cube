package handler

import (
	"net/http"
	"strings"

	"cube/internal"
)

func HandleResource(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/resource/")

	var (
		content string
		lang    string
	)
	if err := internal.Db.QueryRow("select content, lang from source where url = ? and type = 'resource' and active = true", name).Scan(&content, &lang); err != nil {
		Error(w, err)
		return
	}
	switch lang {
	case "javascript":
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	case "html":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	}
	Success(w, content)
}
