package api

import (
	"crypto/rand"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
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

func openValidatedAsset(baseDir, relative string, allowedExt map[string]string) (*os.File, string, error) {
	if relative == "" || filepath.IsAbs(relative) {
		return nil, "", errInvalidAsset
	}
	clean := filepath.Clean(relative)
	if clean != relative || clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return nil, "", errInvalidAsset
	}

	root, err := os.OpenRoot(baseDir)
	if err != nil {
		return nil, "", errInvalidAsset
	}
	defer root.Close()

	f, err := root.OpenFile(clean, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, "", errInvalidAsset
	}
	mimeType, err := validateOpenedAsset(f, clean, allowedExt)
	if err != nil {
		f.Close()
		return nil, "", err
	}
	return f, mimeType, nil
}

func validateOpenedAsset(f *os.File, filename string, allowedExt map[string]string) (string, error) {
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return "", errInvalidAsset
	}
	mimeType, ok := allowedExt[strings.ToLower(filepath.Ext(filename))]
	if !ok {
		return "", errInvalidAsset
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return "", errInvalidAsset
	}
	if !assetContentMatches(f, mimeType) {
		return "", errInvalidAsset
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return "", errInvalidAsset
	}
	return mimeType, nil
}

func assetContentMatches(r io.Reader, expected string) bool {
	if expected == "image/svg+xml" {
		decoder := xml.NewDecoder(io.LimitReader(r, 64<<10))
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
	n, err := r.Read(buf)
	if err != nil && err != io.EOF {
		return false
	}
	return http.DetectContentType(buf[:n]) == expected
}

func writeValidatedUpload(baseDir, filename string, src io.Reader, allowedExt map[string]string) error {
	if _, ok := allowedExt[strings.ToLower(filepath.Ext(filename))]; !ok {
		return errInvalidAsset
	}
	root, err := os.OpenRoot(baseDir)
	if err != nil {
		return err
	}
	defer root.Close()

	tempName := ".upload-" + randomHex(16)
	out, err := root.OpenFile(tempName, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0640)
	if err != nil {
		return err
	}
	keep := false
	defer func() {
		out.Close()
		if !keep {
			_ = root.Remove(tempName)
		}
	}()

	if _, err := io.Copy(out, src); err != nil {
		return err
	}
	if _, err := validateOpenedAsset(out, filename, allowedExt); err != nil {
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	if err := root.Rename(tempName, filename); err != nil {
		return err
	}
	keep = true
	return nil
}

func removeStoredAsset(baseDir, filename string) error {
	root, err := os.OpenRoot(baseDir)
	if err != nil {
		return err
	}
	defer root.Close()
	return root.Remove(filename)
}

func (s *Server) handleMapAsset(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		http.NotFound(w, r)
		return
	}
	m, err := s.db.GetMap(id)
	if err != nil {
		serverError(w, r, err)
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
		serverError(w, r, err)
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
	f, mimeType, err := openValidatedAsset(filepath.Join(s.dataDir, directory), relative, allowedExt)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	serveOpenedAsset(w, r, f, filepath.Base(relative), mimeType)
}

func serveOpenedAsset(w http.ResponseWriter, r *http.Request, f *os.File, filename, mimeType string) {
	info, err := f.Stat()
	if err != nil {
		http.NotFound(w, r)
		return
	}

	disposition := mime.FormatMediaType("inline", map[string]string{"filename": filepath.Base(filename)})
	if disposition == "" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", mimeType)
	w.Header().Set("Content-Security-Policy", "sandbox; default-src 'none'")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Disposition", disposition)
	http.ServeContent(w, r, filepath.Base(filename), info.ModTime(), f)
}

func (s *Server) handleUploadMap(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid campaign id", http.StatusBadRequest)
		return
	}
	if err := parseMultipartForm(w, r, imageUploadLimit); err != nil {
		respondBodyError(w, err)
		return
	}
	name := r.FormValue("name")
	if name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	file, header, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "image is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(filepath.Base(header.Filename)))
	if _, ok := mapAssetTypes[ext]; !ok {
		http.Error(w, "unsupported image format", http.StatusBadRequest)
		return
	}
	filename := randomHex(16) + ext
	destDir := filepath.Join(s.dataDir, "maps")
	if err := os.MkdirAll(destDir, 0750); err != nil {
		serverError(w, r, err)
		return
	}
	if err := writeValidatedUpload(destDir, filename, file, mapAssetTypes); err != nil {
		if err == errInvalidAsset {
			http.Error(w, "image content does not match its format", http.StatusBadRequest)
			return
		}
		serverError(w, r, err)
		return
	}

	imagePath := "maps/" + filename
	mapID, err := s.db.CreateMap(id, name, imagePath)
	if err != nil {
		if cleanupErr := removeStoredAsset(destDir, filename); cleanupErr != nil {
			log.Printf("map upload cleanup failed: %v", cleanupErr)
		}
		serverError(w, r, err)
		return
	}
	m, err := s.db.GetMap(mapID)
	if err != nil {
		serverError(w, r, err)
		return
	}
	if m == nil {
		serverErrorText(w, r, "fetch created map")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(m) //nolint:errcheck
}

// randomHex returns n random hex bytes as a hex string.
func randomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b) //nolint:errcheck // crypto/rand.Read never returns an error on supported platforms
	return fmt.Sprintf("%x", b)
}
