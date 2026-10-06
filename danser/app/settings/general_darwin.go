package settings

import (
	"os"
	"path/filepath"
)

// getOsuInstallation points to the osu!stable-like tree synced from osu!lazer's database (relative to danser's data dir)
func getOsuInstallation() string {
	return "lazer"
}

func getLazerInstallation() string {
	dir, _ := os.UserHomeDir()
	return existingLazerDir(filepath.Join(dir, "Library", "Application Support", "osu"))
}
