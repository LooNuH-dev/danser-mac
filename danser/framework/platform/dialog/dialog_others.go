//go:build !darwin

package dialog

import (
	"errors"
	"fmt"

	sq "github.com/sqweek/dialog"
)

func sprintf(format string, args ...any) string {
	if len(args) == 0 {
		return format
	}

	return fmt.Sprintf(format, args...)
}

func wrapErr(err error) error {
	if errors.Is(err, sq.ErrCancelled) {
		return ErrCancelled
	}

	return err
}

func fileBuilder(b *FileBuilder) *sq.FileBuilder {
	fb := sq.File().Title(b.title).SetStartDir(b.startDir)
	for _, f := range b.filters {
		fb = fb.Filter(f.desc, f.extensions...)
	}

	return fb
}

func loadFile(b *FileBuilder) (string, error) {
	p, err := fileBuilder(b).Load()
	return p, wrapErr(err)
}

func loadFiles(b *FileBuilder) ([]string, error) {
	p, err := fileBuilder(b).LoadMultiple()
	return p, wrapErr(err)
}

func browseDirectory(b *DirectoryBuilder) (string, error) {
	p, err := sq.Directory().Title(b.title).SetStartDir(b.startDir).Browse()
	return p, wrapErr(err)
}

func messageInfo(b *MsgBuilder) {
	sq.Message("%s", b.msg).Title(b.title).Info()
}

func messageError(b *MsgBuilder) {
	sq.Message("%s", b.msg).Title(b.title).Error()
}

func messageYesNo(b *MsgBuilder, isError bool) bool {
	if isError {
		return sq.Message("%s", b.msg).Title(b.title).ErrorYesNo()
	}

	return sq.Message("%s", b.msg).Title(b.title).YesNo()
}
