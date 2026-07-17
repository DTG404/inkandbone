package api

import (
	"encoding/xml"
	"errors"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var errInvalidAsset = errors.New("invalid asset")

var mapAssetTypes = map[string]string{
	".svg":  "image/svg+xml",
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".webp": "image/webp",
}

var portraitAssetTypes = map[string]string{
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".gif":  "image/gif",
	".webp": "image/webp",
}

func storedAssetRelative(storedPath, directory string) (string, error) {
	if storedPath == "" || filepath.IsAbs(storedPath) {
		return "", errInvalidAsset
	}
	clean := filepath.Clean(filepath.FromSlash(storedPath))
	if filepath.ToSlash(clean) != storedPath {
		return "", errInvalidAsset
	}
	prefix := directory + string(filepath.Separator)
	if !strings.HasPrefix(clean, prefix) || clean == prefix {
		return "", errInvalidAsset
	}
	return strings.TrimPrefix(clean, prefix), nil
}

func resolveAsset(baseDir, relative string, allowedExt map[string]string) (string, string, error) {
	if relative == "" || filepath.IsAbs(relative) {
		return "", "", errInvalidAsset
	}
	clean := filepath.Clean(relative)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", "", errInvalidAsset
	}

	base, err := filepath.EvalSymlinks(baseDir)
	if err != nil {
		return "", "", errInvalidAsset
	}
	full, err := filepath.EvalSymlinks(filepath.Join(base, clean))
	if err != nil {
		return "", "", errInvalidAsset
	}
	rel, err := filepath.Rel(base, full)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", "", errInvalidAsset
	}

	info, err := os.Stat(full)
	if err != nil || !info.Mode().IsRegular() {
		return "", "", errInvalidAsset
	}
	mimeType, ok := allowedExt[strings.ToLower(filepath.Ext(full))]
	if !ok || !assetContentMatches(full, mimeType) {
		return "", "", errInvalidAsset
	}
	return full, mimeType, nil
}

func assetContentMatches(fullPath, expected string) bool {
	f, err := os.Open(fullPath)
	if err != nil {
		return false
	}
	defer f.Close()

	if expected == "image/svg+xml" {
		decoder := xml.NewDecoder(io.LimitReader(f, 64<<10))
		for {
			token, err := decoder.Token()
			if err != nil {
				return false
			}
			start, ok := token.(xml.StartElement)
			if !ok {
				continue
			}
			return strings.EqualFold(start.Name.Local, "svg") &&
				(start.Name.Space == "" || start.Name.Space == "http://www.w3.org/2000/svg")
		}
	}

	buf := make([]byte, 512)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return false
	}
	return http.DetectContentType(buf[:n]) == expected
}

func (s *Server) handleMapAsset(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		http.NotFound(w, r)
		return
	}
	m, err := s.db.GetMap(id)
	if err != nil {
		http.Error(w, "failed to load map", http.StatusInternalServerError)
		return
	}
	if m == nil {
		http.NotFound(w, r)
		return
	}
	s.serveTypedAsset(w, r, "maps", m.ImagePath, mapAssetTypes)
}

func (s *Server) handlePortraitAsset(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		http.NotFound(w, r)
		return
	}
	character, err := s.db.GetCharacter(id)
	if err != nil {
		http.Error(w, "failed to load character", http.StatusInternalServerError)
		return
	}
	if character == nil {
		http.NotFound(w, r)
		return
	}
	s.serveTypedAsset(w, r, "portraits", character.PortraitPath, portraitAssetTypes)
}

func (s *Server) serveTypedAsset(w http.ResponseWriter, r *http.Request, directory, storedPath string, allowedExt map[string]string) {
	relative, err := storedAssetRelative(storedPath, directory)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	fullPath, mimeType, err := resolveAsset(filepath.Join(s.dataDir, directory), relative, allowedExt)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	f, err := os.Open(fullPath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		http.NotFound(w, r)
		return
	}

	disposition := mime.FormatMediaType("inline", map[string]string{"filename": filepath.Base(fullPath)})
	if disposition == "" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", mimeType)
	w.Header().Set("Content-Security-Policy", "sandbox; default-src 'none'")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Disposition", disposition)
	http.ServeContent(w, r, filepath.Base(fullPath), info.ModTime(), f)
}

func (s *Server) handleLegacyMapAsset(w http.ResponseWriter, r *http.Request) {
	filename := r.PathValue("filename")
	if filename == "" || filename != filepath.Base(filename) || strings.ContainsAny(filename, `/\\`) {
		http.NotFound(w, r)
		return
	}
	if _, ok := mapAssetTypes[strings.ToLower(filepath.Ext(filename))]; !ok {
		http.NotFound(w, r)
		return
	}
	m, err := s.db.GetMapByImagePath("maps/" + filename)
	if err != nil {
		http.Error(w, "failed to load map", http.StatusInternalServerError)
		return
	}
	if m == nil {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, "/api/assets/maps/"+strconv.FormatInt(m.ID, 10), http.StatusTemporaryRedirect)
}
