package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/digitalghost404/inkandbone/internal/db"
)

// Only public scene text and the ruleset name cross the map worker boundary.
func (s *Server) illustrateMap(ctx context.Context, campaignID int64, name, scene string) (int64, error) {
	campaign, err := s.db.GetCampaign(campaignID)
	if err != nil {
		return 0, err
	}
	if campaign == nil {
		return 0, errors.New("unknown map campaign")
	}
	rule, err := s.db.GetRuleset(campaign.RulesetID)
	if err != nil {
		return 0, err
	}
	if rule == nil {
		return 0, errors.New("unknown map ruleset")
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "unix", s.mapSocket)
	if err != nil {
		return 0, fmt.Errorf("connect to map worker: %w", err)
	}
	defer conn.Close()
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stop()
	if err = json.NewEncoder(conn).Encode(map[string]string{"name": name, "ruleset": rule.Name, "scene": scene}); err != nil {
		return 0, err
	}
	var reply struct {
		Image []byte       `json:"image"`
		Zones []db.MapZone `json:"zones"`
		Error string       `json:"error"`
	}
	if err = json.NewDecoder(io.LimitReader(conn, 24<<20)).Decode(&reply); err != nil {
		return 0, fmt.Errorf("read map worker result: %w", err)
	}
	if reply.Error != "" || len(reply.Image) > 16<<20 || len(reply.Zones) == 0 || len(reply.Zones) > 16 {
		return 0, errors.New("map worker returned incomplete artwork or zones")
	}
	ext := ""
	switch http.DetectContentType(reply.Image) {
	case "image/png":
		ext = ".png"
	case "image/jpeg":
		ext = ".jpg"
	case "image/webp":
		ext = ".webp"
	default:
		return 0, errors.New("map worker returned an unsupported image")
	}
	for _, z := range reply.Zones {
		if strings.TrimSpace(z.Name) == "" || len(z.Name) > 256 || z.X < 0 || z.Y < 0 || z.Width <= 0 || z.Height <= 0 || z.X+z.Width > 1.000001 || z.Y+z.Height > 1.000001 {
			return 0, errors.New("map worker returned invalid zone bounds")
		}
	}
	dir := filepath.Join(s.dataDir, "maps")
	if err = os.MkdirAll(dir, 0750); err != nil {
		return 0, err
	}
	file := "map_" + randomHex(16) + ext
	if err = writeValidatedUpload(dir, file, bytes.NewReader(reply.Image), mapAssetTypes); err != nil {
		return 0, err
	}
	id, err := s.db.CreateIllustratedMap(ctx, campaignID, name, "maps/"+file, reply.Zones)
	if err != nil {
		_ = removeStoredAsset(dir, file)
	}
	return id, err
}
