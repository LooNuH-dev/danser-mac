//go:build !windows && !linux

package files

import "os"

// setPipeSize is a no-op, pipe buffer size can't be changed on macOS
func setPipeSize(_ *os.File, _ int) error {
	return nil
}
