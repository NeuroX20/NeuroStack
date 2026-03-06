package handler

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const wwwRoot = "/root/neurostack/www"

type FileInfo struct {
	Name    string `json:"name"`
	IsDir   bool   `json:"is_dir"`
	Size    int64  `json:"size"`
	ModTime string `json:"mod_time"`
	Ext     string `json:"ext"`
}

func safePath(rel string) (string, error) {
	if rel == "" {
		rel = "/"
	}
	abs := filepath.Join(wwwRoot, filepath.Clean("/"+rel))
	if !strings.HasPrefix(abs, wwwRoot) {
		return "", fmt.Errorf("access denied")
	}
	return abs, nil
}

// FMList lists files in a directory.
func FMList(w http.ResponseWriter, r *http.Request) {
	rel := r.URL.Query().Get("path")
	abs, err := safePath(rel)
	if err != nil {
		writeJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}

	entries, err := os.ReadDir(abs)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}

	var files []FileInfo
	for _, e := range entries {
		info, _ := e.Info()
		size := int64(0)
		if info != nil {
			size = info.Size()
		}
		files = append(files, FileInfo{
			Name:    e.Name(),
			IsDir:   e.IsDir(),
			Size:    size,
			ModTime: info.ModTime().Format("2006-01-02 15:04"),
			Ext:     strings.ToLower(filepath.Ext(e.Name())),
		})
	}

	// Sort: dirs first, then files
	sort.Slice(files, func(i, j int) bool {
		if files[i].IsDir != files[j].IsDir {
			return files[i].IsDir
		}
		return files[i].Name < files[j].Name
	})

	writeJSON(w, 200, map[string]interface{}{
		"path":  rel,
		"files": files,
	})
}

// FMRead reads a text file.
func FMRead(w http.ResponseWriter, r *http.Request) {
	rel := r.URL.Query().Get("path")
	abs, err := safePath(rel)
	if err != nil {
		writeJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}

	content, err := os.ReadFile(abs)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, 200, map[string]string{"content": string(content)})
}

// FMWrite saves a text file.
func FMWrite(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid JSON"})
		return
	}

	abs, err := safePath(body.Path)
	if err != nil {
		writeJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}

	if err := os.WriteFile(abs, []byte(body.Content), 0644); err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, 200, map[string]string{"status": "saved"})
}

// FMDelete deletes a file or directory.
func FMDelete(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid JSON"})
		return
	}

	abs, err := safePath(body.Path)
	if err != nil {
		writeJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}

	if err := os.RemoveAll(abs); err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, 200, map[string]string{"status": "deleted"})
}

// FMMkdir creates a new directory.
func FMMkdir(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid JSON"})
		return
	}

	abs, err := safePath(body.Path)
	if err != nil {
		writeJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}

	if err := os.MkdirAll(abs, 0755); err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, 200, map[string]string{"status": "created"})
}

// FMUpload handles file uploads.
func FMUpload(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(64 << 20) // 64MB max

	dir := r.FormValue("path")
	abs, err := safePath(dir)
	if err != nil {
		writeJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}

	files := r.MultipartForm.File["files"]
	uploaded := []string{}

	for _, fh := range files {
		src, err := fh.Open()
		if err != nil {
			continue
		}
		defer src.Close()

		destPath := filepath.Join(abs, filepath.Base(fh.Filename))
		dst, err := os.Create(destPath)
		if err != nil {
			continue
		}
		defer dst.Close()

		io.Copy(dst, src)
		uploaded = append(uploaded, fh.Filename)

		// Auto-extract ZIP
		if strings.ToLower(filepath.Ext(fh.Filename)) == ".zip" {
			extractZip(destPath, abs)
		}
	}

	writeJSON(w, 200, map[string]interface{}{
		"status":   "uploaded",
		"files":    uploaded,
		"count":    len(uploaded),
	})
}

// FMDownload serves a file for download.
func FMDownload(w http.ResponseWriter, r *http.Request) {
	rel := r.URL.Query().Get("path")
	abs, err := safePath(rel)
	if err != nil {
		http.Error(w, "Access denied", 403)
		return
	}

	info, err := os.Stat(abs)
	if err != nil || info.IsDir() {
		http.Error(w, "File not found", 404)
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename="+filepath.Base(abs))
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeFile(w, r, abs)
}

// FMZip creates a ZIP of a directory.
func FMZip(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid JSON"})
		return
	}

	abs, err := safePath(body.Path)
	if err != nil {
		writeJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}

	zipName := filepath.Base(abs) + "_" + time.Now().Format("20060102_150405") + ".zip"
	zipPath := filepath.Join(filepath.Dir(abs), zipName)

	if err := createZip(abs, zipPath); err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, 200, map[string]string{"status": "zipped", "file": zipName})
}

func extractZip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)
		if !strings.HasPrefix(fpath, dest) {
			continue
		}
		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, 0755)
			continue
		}
		os.MkdirAll(filepath.Dir(fpath), 0755)
		out, err := os.Create(fpath)
		if err != nil {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			out.Close()
			continue
		}
		io.Copy(out, rc)
		out.Close()
		rc.Close()
	}
	return nil
}

func createZip(src, dest string) error {
	zf, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer zf.Close()

	zw := zip.NewWriter(zf)
	defer zw.Close()

	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(filepath.Dir(src), path)
		w, err := zw.Create(rel)
		if err != nil {
			return err
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(w, f)
		return err
	})
}
