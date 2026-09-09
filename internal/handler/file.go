package handler

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path"
	"regexp"
	"strings"

	"cube/internal/util"
)

const ROOT_FOLDER = "files"

var REGEX_FILE_NAME = regexp.MustCompile(`\S+`) // 文件/文件夹名称必填校验（至少一个非空白字符）

func HandleFile(w http.ResponseWriter, r *http.Request) {
	var (
		data       interface{}
		returnless bool
		err        error
	)
	switch r.Method {
	case http.MethodGet:
		if r.URL.Query().Get("download") != "" {
			handleFileDownload(w, r)
			returnless = true
		} else {
			data, err = handleFileList(r)
		}
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
	if !returnless {
		Success(w, data)
	}
}

func handleFileList(r *http.Request) (interface{}, error) {
	os.MkdirAll(ROOT_FOLDER, 0o755)

	p := &util.QueryParams{Values: r.URL.Query()}
	fp, err := toPath(ROOT_FOLDER, p.Get("path"))
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(fp)
	if err != nil {
		return nil, err
	}

	var files []map[string]interface{}
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		files = append(files, map[string]interface{}{
			"name":   entry.Name(),
			"size":   info.Size(),
			"time":   info.ModTime(),
			"folder": entry.IsDir(),
		})
	}
	return files, nil
}

func handleFileUpload(r *http.Request) error {
	r.ParseMultipartForm(32 << 20) // 32<<20=32MB，multipart 表单内存缓冲上限，超过部分写入临时文件

	sub := r.FormValue("path")

	if r.FormValue("type") == "folder" {
		folder := r.FormValue("name")
		if err := util.ValidateString(folder, REGEX_FILE_NAME, "name is required"); err != nil {
			return err
		}
		fp, err := toPath(ROOT_FOLDER, sub)
		if err != nil {
			return err
		}
		// 校验文件夹名不逃逸 fp，防止跨目录创建
		fp, err = toPath(fp, folder)
		if err != nil {
			return err
		}
		return os.MkdirAll(fp, 0o755)
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

	folder, err := toPath(ROOT_FOLDER, sub)
	if err != nil {
		return err
	}
	os.MkdirAll(folder, 0o755)

	fp, err := toPath(folder, name)
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
	p := &util.QueryParams{Values: r.URL.Query()}
	name := p.Get("name")
	if err := util.ValidateString(name, REGEX_FILE_NAME, "name is required"); err != nil {
		return err
	}

	fp, err := toPath(ROOT_FOLDER, name)
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

	fp, err := toPath(ROOT_FOLDER, name)
	if err != nil {
		Error(w, err)
		return
	}

	if _, err := os.Stat(fp); os.IsNotExist(err) {
		Error(w, errors.New("file not found"))
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename=\""+path.Base(name)+"\"")
	http.ServeFile(w, r, fp)
}

func toPath(base, sub string) (string, error) {
	fp := path.Clean(path.Join(base, sub))
	if !strings.HasPrefix(fp+"/", base+"/") {
		return "", errors.New("path traversal denied")
	}
	return fp, nil
}
