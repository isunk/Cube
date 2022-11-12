package module

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"cube/internal/builtin"
)

func init() {
	register("file", func(ctx Context) interface{} {
		return &FileClient{}
	})
}

type FileClient struct{}

func (f *FileClient) getPath(name string) (string, error) {
	fp := path.Clean("files/" + name)
	if !strings.HasPrefix(fp+"/", "files/") {
		return "", errors.New("permission denied")
	}
	return fp, nil
}

func (f *FileClient) Read(name string) (builtin.Buffer, error) {
	fp, err := f.getPath(name)
	if err != nil {
		return nil, err
	}

	return os.ReadFile(fp)
}

func (f *FileClient) ReadRange(name string, offset int64, length int64) (builtin.Buffer, error) {
	fp, err := f.getPath(name)
	if err != nil {
		return nil, err
	}

	fd, err := os.Open(fp)
	if err != nil {
		return nil, err
	}
	defer fd.Close()

	if _, err := fd.Seek(offset, io.SeekStart); err != nil { // 设置光标的位置：距离文件开头 offset 个字节处
		return nil, err
	}

	data := make([]byte, length)
	if _, err := fd.Read(data); err != nil && err != io.EOF {
		return nil, err
	}

	return data, nil
}

func (f *FileClient) Write(name string, bytes []byte) error {
	fp, err := f.getPath(name)
	if err != nil {
		return err
	}

	paths, _ := filepath.Split(fp)
	os.MkdirAll(paths, os.ModePerm)
	return os.WriteFile(fp, bytes, 0o664)
}

func (f *FileClient) WriteRange(name string, offset int64, bytes []byte) error {
	fp, err := f.getPath(name)
	if err != nil {
		return err
	}

	fd, err := os.OpenFile(fp, os.O_WRONLY, 0o664)
	if err != nil {
		return err
	}
	defer fd.Close()

	if _, err := fd.Seek(offset, io.SeekStart); err != nil {
		return err
	}

	_, err = fd.Write(bytes)
	return err
}

func (f *FileClient) Stat(name string) (fs.FileInfo, error) {
	fp, err := f.getPath(name)
	if err != nil {
		return nil, err
	}

	return os.Stat(fp)
}

func (f *FileClient) List(name string) ([]string, error) {
	fp, err := f.getPath(name)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(fp)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0)
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names, nil
}

func (f *FileClient) Remove(name string) error {
	fp, err := f.getPath(name)
	if err != nil {
		return err
	}

	if fp == "files" || strings.HasSuffix(name, "/") {
		names, err := f.List(name)
		if err != nil {
			return err
		}
		for _, n := range names {
			if err := os.RemoveAll(path.Join(fp, n)); err != nil {
				return err
			}
		}
		return nil
	}

	return os.RemoveAll(fp)
}
