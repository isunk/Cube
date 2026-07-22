package handler

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
)

const FileDir = "files"

func HandleFile(w http.ResponseWriter, r *http.Request) {
	var (
		data interface{}
		err  error
	)
	switch r.Method {
	case http.MethodGet:
		if r.URL.Query().Get("download") != "" {
			handleFileDownload(w, r)
			return
		}
		data, err = handleFileList(r)
	case http.MethodPost:
		err = handleFileUpload(r)
	case http.MethodDelete:
		err = handleFileDelete(r)
	default:
		Error(w, http.StatusMethodNotAllowed)
		return
	}
	if err != nil {
		Error(w, err)
		return
	}
	Success(w, data)
}

func safePath(base, sub string) (string, error) {
	fp := path.Clean(path.Join(base, sub))
	if !strings.HasPrefix(fp+"/", base+"/") && fp != base {
		return "", errors.New("path traversal denied")
	}
	return fp, nil
}

func handleFileList(r *http.Request) (interface{}, error) {
	os.MkdirAll(FileDir, 0755)

	subPath := r.URL.Query().Get("path")
	fp, err := safePath(FileDir, subPath)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(fp)
	if err != nil {
		return nil, err
	}

	fp = strings.TrimPrefix(fp, FileDir)
	fp = strings.TrimPrefix(fp, "/")

	var files []map[string]interface{}
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		name := entry.Name()
		if fp != "" {
			name = fp + "/" + name
		}
		files = append(files, map[string]interface{}{
			"name": name,
			"size": info.Size(),
			"time": info.ModTime(),
			"dir":  entry.IsDir(),
		})
	}
	return files, nil
}

func handleFileUpload(r *http.Request) error {
	r.ParseMultipartForm(32 << 20)

	subPath := r.FormValue("path")

	rtype := r.FormValue("type")
	if rtype == "directory" {
		dirName := r.FormValue("name")
		if dirName == "" {
			return errors.New("name is required")
		}
		fp, err := safePath(FileDir, subPath)
		if err != nil {
			return err
		}
		return os.MkdirAll(path.Join(fp, dirName), 0755)
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		return err
	}
	defer file.Close()

	name := r.FormValue("name")
	if name == "" {
		name = header.Filename
	}

	dir, err := safePath(FileDir, subPath)
	if err != nil {
		return err
	}
	os.MkdirAll(dir, 0755)

	fp, err := safePath(dir, name)
	if err != nil {
		return err
	}

	dst, err := os.Create(fp)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, file)
	return err
}

func handleFileDelete(r *http.Request) error {
	name := r.URL.Query().Get("name")
	if name == "" {
		return errors.New("name is required")
	}

	fp, err := safePath(FileDir, name)
	if err != nil {
		return err
	}

	info, err := os.Stat(fp)
	if os.IsNotExist(err) {
		return errors.New("file not found")
	}

	if info.IsDir() {
		return os.RemoveAll(fp)
	}
	return os.Remove(fp)
}

func handleFileDownload(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("download")
	if name == "" {
		Error(w, errors.New("name is required"))
		return
	}

	fp, err := safePath(FileDir, name)
	if err != nil {
		Error(w, err)
		return
	}

	if _, err := os.Stat(fp); os.IsNotExist(err) {
		Error(w, errors.New("file not found"))
		return
	}

	baseName := path.Base(name)
	w.Header().Set("Content-Disposition", "attachment; filename=\""+baseName+"\"")
	http.ServeFile(w, r, fp)
}
