//go:build linux

package api

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestOpenValidatedAssetRejectsSpecialFileWithoutBlocking(t *testing.T) {
	mapsDir := filepath.Join(t.TempDir(), "maps")
	require.NoError(t, os.MkdirAll(mapsDir, 0750))
	require.NoError(t, syscall.Mkfifo(filepath.Join(mapsDir, "pipe.png"), 0600))

	result := make(chan error, 1)
	go func() {
		f, _, err := openValidatedAsset(mapsDir, "pipe.png", mapAssetTypes)
		if f != nil {
			f.Close()
		}
		result <- err
	}()
	select {
	case err := <-result:
		require.Error(t, err)
	case <-time.After(time.Second):
		t.Fatal("opening a special file blocked")
	}
}
