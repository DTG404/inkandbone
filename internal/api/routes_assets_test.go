package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var validPNG = []byte("\x89PNG\r\n\x1a\nasset-data")
var validJPEG = []byte("\xff\xd8\xff\xdbasset-data")

func createAssetMap(t *testing.T, s *Server, campaignID int64, imagePath string) int64 {
	t.Helper()
	mapID, err := s.db.CreateMap(campaignID, "Asset Map", imagePath)
	require.NoError(t, err)
	return mapID
}

func createAssetCharacter(t *testing.T, s *Server, campaignID int64, portraitPath string) int64 {
	t.Helper()
	characterID, err := s.db.CreateCharacter(campaignID, "Asset Hero")
	require.NoError(t, err)
	require.NoError(t, s.db.UpdateCharacterPortrait(characterID, portraitPath))
	return characterID
}

func writeAssetFile(t *testing.T, dataDir, relative string, content []byte) string {
	t.Helper()
	fullPath := filepath.Join(dataDir, filepath.FromSlash(relative))
	require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0750))
	require.NoError(t, os.WriteFile(fullPath, content, 0640))
	return fullPath
}

func getAsset(t *testing.T, s *Server, url string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	s.ServeHTTP(w, httptest.NewRequest(http.MethodGet, url, nil))
	return w
}

func TestAssetMapServesOnlyReferencedRecordWithSafeHeaders(t *testing.T) {
	dataDir := t.TempDir()
	writeAssetFile(t, dataDir, "maps/world.png", validPNG)
	s := newTestServerWithDir(t, dataDir)
	campaignID, _ := seedCampaign(t, s.db)
	mapID := createAssetMap(t, s, campaignID, "maps/world.png")

	w := getAsset(t, s, "/api/assets/maps/"+strconv.FormatInt(mapID, 10))

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, validPNG, w.Body.Bytes())
	assert.Equal(t, "image/png", w.Header().Get("Content-Type"))
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "sandbox; default-src 'none'", w.Header().Get("Content-Security-Policy"))
	assert.Equal(t, "inline; filename=world.png", w.Header().Get("Content-Disposition"))
}

func TestAssetMapServesReferencedSVGWithRestrictiveHeaders(t *testing.T) {
	dataDir := t.TempDir()
	svg := []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1 1"><path d="M0 0"/></svg>`)
	writeAssetFile(t, dataDir, "maps/world.svg", svg)
	s := newTestServerWithDir(t, dataDir)
	campaignID, _ := seedCampaign(t, s.db)
	mapID := createAssetMap(t, s, campaignID, "maps/world.svg")

	w := getAsset(t, s, "/api/assets/maps/"+strconv.FormatInt(mapID, 10))

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, svg, w.Body.Bytes())
	assert.Equal(t, "image/svg+xml", w.Header().Get("Content-Type"))
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "sandbox; default-src 'none'", w.Header().Get("Content-Security-Policy"))
	assert.Equal(t, "inline; filename=world.svg", w.Header().Get("Content-Disposition"))
}

func TestAssetPortraitServesRasterByCharacterID(t *testing.T) {
	dataDir := t.TempDir()
	jpeg := []byte("\xff\xd8\xff\xdbportrait-data")
	writeAssetFile(t, dataDir, "portraits/hero.jpg", jpeg)
	s := newTestServerWithDir(t, dataDir)
	campaignID, _ := seedCampaign(t, s.db)
	characterID := createAssetCharacter(t, s, campaignID, "portraits/hero.jpg")

	w := getAsset(t, s, "/api/assets/portraits/"+strconv.FormatInt(characterID, 10))

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, jpeg, w.Body.Bytes())
	assert.Equal(t, "image/jpeg", w.Header().Get("Content-Type"))
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "sandbox; default-src 'none'", w.Header().Get("Content-Security-Policy"))
	assert.Equal(t, "inline; filename=hero.jpg", w.Header().Get("Content-Disposition"))
}

func TestAssetRouteRejectsMissingDatabaseRecord(t *testing.T) {
	s := newTestServer(t)
	assert.Equal(t, http.StatusNotFound, getAsset(t, s, "/api/assets/maps/999999").Code)
	assert.Equal(t, http.StatusNotFound, getAsset(t, s, "/api/assets/portraits/999999").Code)
}

func TestAssetRouteRejectsInvalidStoredPaths(t *testing.T) {
	tests := []struct {
		name       string
		imagePath  func(dataDir string) string
		prepare    func(t *testing.T, dataDir string)
		assetRoute string
	}{
		{
			name:       "absolute map path",
			imagePath:  func(dataDir string) string { return filepath.Join(dataDir, "maps", "absolute.png") },
			prepare:    func(t *testing.T, dataDir string) { writeAssetFile(t, dataDir, "maps/absolute.png", validPNG) },
			assetRoute: "maps",
		},
		{
			name:       "map parent traversal",
			imagePath:  func(string) string { return "maps/../ttrpg.db" },
			prepare:    func(t *testing.T, dataDir string) { writeAssetFile(t, dataDir, "ttrpg.db", validPNG) },
			assetRoute: "maps",
		},
		{
			name:       "portrait parent traversal",
			imagePath:  func(string) string { return "portraits/../ttrpg.db" },
			prepare:    func(t *testing.T, dataDir string) { writeAssetFile(t, dataDir, "ttrpg.db", validPNG) },
			assetRoute: "portraits",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dataDir := t.TempDir()
			tt.prepare(t, dataDir)
			s := newTestServerWithDir(t, dataDir)
			campaignID, _ := seedCampaign(t, s.db)
			var id int64
			if tt.assetRoute == "maps" {
				id = createAssetMap(t, s, campaignID, tt.imagePath(dataDir))
			} else {
				id = createAssetCharacter(t, s, campaignID, tt.imagePath(dataDir))
			}
			w := getAsset(t, s, fmt.Sprintf("/api/assets/%s/%d", tt.assetRoute, id))
			assert.Equal(t, http.StatusNotFound, w.Code)
		})
	}
}

func TestAssetRouteRejectsSymlinkEscape(t *testing.T) {
	dataDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dataDir, "maps"), 0750))
	outsideDir := t.TempDir()
	outside := writeAssetFile(t, outsideDir, "outside.png", validPNG)
	require.NoError(t, os.Symlink(outside, filepath.Join(dataDir, "maps", "escape.png")))
	s := newTestServerWithDir(t, dataDir)
	campaignID, _ := seedCampaign(t, s.db)
	mapID := createAssetMap(t, s, campaignID, "maps/escape.png")

	w := getAsset(t, s, "/api/assets/maps/"+strconv.FormatInt(mapID, 10))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestOpenedAssetDescriptorRemainsStableAfterPathSwap(t *testing.T) {
	dataDir := t.TempDir()
	mapsDir := filepath.Join(dataDir, "maps")
	approvedPath := writeAssetFile(t, dataDir, "maps/world.png", validPNG)
	attackerDir := t.TempDir()
	attackerPath := writeAssetFile(t, attackerDir, "attacker.png", []byte("attacker-controlled bytes"))

	f, mimeType, err := openValidatedAsset(mapsDir, "world.png", mapAssetTypes)
	require.NoError(t, err)
	defer f.Close()

	require.NoError(t, os.Rename(approvedPath, filepath.Join(mapsDir, "approved-original.png")))
	require.NoError(t, os.Symlink(attackerPath, approvedPath))

	w := httptest.NewRecorder()
	serveOpenedAsset(w, httptest.NewRequest(http.MethodGet, "/api/assets/maps/1", nil), f, "world.png", mimeType)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, validPNG, w.Body.Bytes())
	assert.NotContains(t, w.Body.String(), "attacker-controlled")
}

func TestOpenValidatedAssetRejectsSymlinkEscapes(t *testing.T) {
	dataDir := t.TempDir()
	mapsDir := filepath.Join(dataDir, "maps")
	require.NoError(t, os.MkdirAll(mapsDir, 0750))
	outsideDir := t.TempDir()
	outsidePath := writeAssetFile(t, outsideDir, "outside.png", validPNG)

	require.NoError(t, os.Symlink(outsidePath, filepath.Join(mapsDir, "final.png")))
	f, _, err := openValidatedAsset(mapsDir, "final.png", mapAssetTypes)
	if f != nil {
		f.Close()
	}
	require.Error(t, err)

	require.NoError(t, os.Symlink(outsideDir, filepath.Join(mapsDir, "intermediate")))
	f, _, err = openValidatedAsset(mapsDir, "intermediate/outside.png", mapAssetTypes)
	if f != nil {
		f.Close()
	}
	require.Error(t, err)
}

func TestAssetRouteEnforcesTypeSpecificExtensions(t *testing.T) {
	dataDir := t.TempDir()
	writeAssetFile(t, dataDir, "maps/animated.gif", []byte("GIF89aasset-data"))
	writeAssetFile(t, dataDir, "portraits/vector.svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`))
	s := newTestServerWithDir(t, dataDir)
	campaignID, _ := seedCampaign(t, s.db)
	mapID := createAssetMap(t, s, campaignID, "maps/animated.gif")
	characterID := createAssetCharacter(t, s, campaignID, "portraits/vector.svg")

	assert.Equal(t, http.StatusNotFound, getAsset(t, s, "/api/assets/maps/"+strconv.FormatInt(mapID, 10)).Code)
	assert.Equal(t, http.StatusNotFound, getAsset(t, s, "/api/assets/portraits/"+strconv.FormatInt(characterID, 10)).Code)
}

func TestRemoveStoredAssetReturnsCleanupError(t *testing.T) {
	err := removeStoredAsset(filepath.Join(t.TempDir(), "missing"), "unpublished.png")
	require.Error(t, err)
}

func TestAssetRouteRejectsMismatchedContent(t *testing.T) {
	dataDir := t.TempDir()
	writeAssetFile(t, dataDir, "maps/not-an-image.png", []byte("plain text"))
	writeAssetFile(t, dataDir, "portraits/wrong.jpg", validPNG)
	s := newTestServerWithDir(t, dataDir)
	campaignID, _ := seedCampaign(t, s.db)
	mapID := createAssetMap(t, s, campaignID, "maps/not-an-image.png")
	characterID := createAssetCharacter(t, s, campaignID, "portraits/wrong.jpg")

	assert.Equal(t, http.StatusNotFound, getAsset(t, s, "/api/assets/maps/"+strconv.FormatInt(mapID, 10)).Code)
	assert.Equal(t, http.StatusNotFound, getAsset(t, s, "/api/assets/portraits/"+strconv.FormatInt(characterID, 10)).Code)
}

func TestAssetRouteCannotServeDatabaseOrFilename(t *testing.T) {
	dataDir := t.TempDir()
	writeAssetFile(t, dataDir, "ttrpg.db", []byte("private database"))
	writeAssetFile(t, dataDir, "maps/world.png", validPNG)
	s := newTestServerWithDir(t, dataDir)
	campaignID, _ := seedCampaign(t, s.db)
	mapID := createAssetMap(t, s, campaignID, "maps/world.png")

	assert.Equal(t, http.StatusNotFound, getAsset(t, s, "/api/files/ttrpg.db").Code)
	assert.Equal(t, http.StatusNotFound, getAsset(t, s, "/api/assets/maps/world.png").Code)
	assert.Equal(t, http.StatusNotFound, getAsset(t, s, fmt.Sprintf("/api/assets/maps/%d/world.png", mapID)).Code)
}

func TestAssetLegacyFileSurfaceIsRemoved(t *testing.T) {
	dataDir := t.TempDir()
	writeAssetFile(t, dataDir, "maps/world.png", validPNG)
	s := newTestServerWithDir(t, dataDir)
	campaignID, _ := seedCampaign(t, s.db)
	createAssetMap(t, s, campaignID, "maps/world.png")

	tests := []string{
		"/api/files",
		"/api/files/",
		"/api/files/ttrpg.db",
		"/api/files/portraits/world.png",
		"/api/files/maps/world.png",
		"/api/files/maps/subdir/world.png",
		"/api/files/maps/%2e%2e%2fworld.png",
		"/api/files/maps/world.png/extra",
		"/api/files/maps/subdir/../world.png",
	}
	for _, url := range tests {
		t.Run(url, func(t *testing.T) {
			assert.Equal(t, http.StatusNotFound, getAsset(t, s, url).Code)
		})
	}
}
