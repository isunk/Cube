package handler

import (
	"net/http"
	"os"
	"path"
	"strings"
)

func HandleDocument(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/document/")

	fp := path.Clean("docs/" + name)
	if !strings.HasPrefix(fp+"/", "docs/") {
		Error(w, http.StatusForbidden)
		return
	}

	content, err := os.ReadFile(fp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	Success(w, string(content))
}
