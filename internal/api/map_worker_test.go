package api

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMapWorkerSuppliesCampaignMapAndZones(t *testing.T) {
	s := newTestServerWithAI(t, detectThenValidSVGCompleter{})
	cid, sid := seedCampaign(t, s.db)
	socket := filepath.Join(t.TempDir(), "map.sock")
	listener, err := net.Listen("unix", socket)
	require.NoError(t, err)
	defer listener.Close()
	requests := make(chan map[string]string, 2)
	go func() {
		for range 2 {
			c, e := listener.Accept()
			if e != nil {
				return
			}
			var scene map[string]string
			if json.NewDecoder(c).Decode(&scene) == nil {
				requests <- scene
				_ = json.NewEncoder(c).Encode(map[string]any{"image": validPNG, "zones": []map[string]any{{"name": "Watchpost", "x": 0.2, "y": 0.2, "width": 0.3, "height": 0.3}}})
			}
			c.Close()
		}
	}()
	s.mapSocket = socket
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/campaigns/%d/maps/generate", cid), strings.NewReader(`{"name":"Market","context":"public square"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	require.Equal(t, 201, rec.Code)
	require.Equal(t, "public square", (<-requests)["scene"])
	s.autoGenerateMap(t.Context(), sid, "The party enters Ashen Tower.")
	require.Equal(t, "ruin", (<-requests)["scene"])
	maps, err := s.db.ListMaps(cid)
	require.NoError(t, err)
	require.Len(t, maps, 2)
	for _, m := range maps {
		asset := getAsset(t, s, fmt.Sprintf("/api/assets/maps/%d", m.ID))
		require.Equal(t, 200, asset.Code)
		require.Equal(t, validPNG, asset.Body.Bytes())
		zones, err := s.db.ListMapZones(m.ID)
		require.NoError(t, err)
		require.Len(t, zones, 1)
		require.Equal(t, "Watchpost", zones[0].Name)
	}
}

func TestSubscriptionMapForNewRulesetRollsBackOnZoneFailure(t *testing.T) {
	s := newTestServer(t)
	rule, err := s.db.CreateRuleset("new-tabletop-system", "[]", "1")
	require.NoError(t, err)
	cid, err := s.db.CreateCampaign(rule, "Future table", "")
	require.NoError(t, err)
	socket := filepath.Join(t.TempDir(), "worker.sock")
	l, err := net.Listen("unix", socket)
	require.NoError(t, err)
	defer l.Close()
	s.mapSocket = socket
	rules := make(chan string, 2)
	go func() {
		for range 2 {
			c, e := l.Accept()
			if e != nil {
				return
			}
			var request map[string]string
			if json.NewDecoder(c).Decode(&request) == nil {
				rules <- request["ruleset"]
				_ = json.NewEncoder(c).Encode(map[string]any{"image": validPNG, "zones": []map[string]any{{"name": "Gate", "width": 0.25, "height": 0.5}}})
			}
			c.Close()
		}
	}()
	_, err = s.db.SQL().Exec("CREATE TRIGGER reject_zone BEFORE INSERT ON map_zones BEGIN SELECT RAISE(ABORT,'forced'); END")
	require.NoError(t, err)
	_, err = s.illustrateMap(t.Context(), cid, "Gate", "You see a narrow gate.")
	require.Error(t, err)
	files, err := filepath.Glob(filepath.Join(s.dataDir, "maps", "*"))
	require.NoError(t, err)
	require.Empty(t, files)
	maps, err := s.db.ListMaps(cid)
	require.NoError(t, err)
	require.Empty(t, maps)
	require.Equal(t, "new-tabletop-system", <-rules)
	_, err = s.db.SQL().Exec("DROP TRIGGER reject_zone")
	require.NoError(t, err)
	id, err := s.illustrateMap(t.Context(), cid, "Gate", "You see a narrow gate.")
	require.NoError(t, err)
	require.Positive(t, id)
	require.Equal(t, "new-tabletop-system", <-rules)
}
