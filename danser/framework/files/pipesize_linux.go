package files

import (
	"os"

	"golang.org/x/sys/unix"
)

func setPipeSize(file *os.File, size int) error {
	_, err := unix.FcntlInt(file.Fd(), unix.F_SETPIPE_SZ, size)
	return err
}
