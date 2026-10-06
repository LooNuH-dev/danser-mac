// Package lazer links osu!lazer's library into danser's osu!stable-like directories using the bundled lazer-bridge tool
package lazer

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/wieku/danser-go/app/settings"
	"github.com/wieku/danser-go/framework/env"
	"github.com/wieku/danser-go/framework/files"
)

var synced bool

// Sync links lazer's beatmaps, skins and replays into the configured Songs/Skins/Replays directories.
// It's a no-op if OsuLazerDir is empty or it already ran in this session.
func Sync() error {
	lazerDir := strings.TrimSpace(settings.General.OsuLazerDir)
	if lazerDir == "" || synced {
		return nil
	}

	// lazer-bridge creates Songs, Skins and Replays inside a single root, so they have to share a parent
	songs, _ := filepath.Abs(settings.General.GetSongsDir())
	skins, _ := filepath.Abs(settings.General.GetSkinsDir())
	replays, _ := filepath.Abs(settings.General.GetReplaysDir())

	root := filepath.Dir(songs)
	if filepath.Base(songs) != "Songs" || filepath.Join(root, "Skins") != skins || filepath.Join(root, "Replays") != replays {
		return errors.New("osu!lazer import requires Songs, Skins and Replays directories to be named so and share the same parent directory")
	}

	bridge, err := files.GetCommandExec("lazer-bridge", "lazer-bridge")
	if err != nil {
		return fmt.Errorf("lazer-bridge not found in %s: %w", env.LibDir(), err)
	}

	log.Println("Lazer: Syncing osu!lazer library from", lazerDir, "into", root)

	var stdout, stderr bytes.Buffer

	cmd := exec.Command(bridge, "--lazer", lazerDir, "--out", root)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err = cmd.Run(); err != nil {
		return fmt.Errorf("lazer-bridge failed: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	log.Println("Lazer:", strings.TrimSpace(stdout.String()))

	synced = true

	return nil
}
