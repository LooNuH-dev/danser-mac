// Package dialog mirrors the subset of github.com/sqweek/dialog used by danser. On macOS native dialogs are shown
// by osascript in a separate process, because a modal NSOpenPanel run inside SDL's event loop doesn't receive input.
package dialog

import "errors"

// ErrCancelled is returned when the user cancels a file or directory dialog
var ErrCancelled = errors.New("cancelled")

type filter struct {
	desc       string
	extensions []string
}

type FileBuilder struct {
	title    string
	startDir string
	filters  []filter
}

func File() *FileBuilder {
	return &FileBuilder{}
}

func (b *FileBuilder) Title(title string) *FileBuilder {
	b.title = title
	return b
}

func (b *FileBuilder) Filter(desc string, extensions ...string) *FileBuilder {
	b.filters = append(b.filters, filter{desc, extensions})
	return b
}

func (b *FileBuilder) SetStartDir(dir string) *FileBuilder {
	b.startDir = dir
	return b
}

func (b *FileBuilder) Load() (string, error) {
	return loadFile(b)
}

func (b *FileBuilder) LoadMultiple() ([]string, error) {
	return loadFiles(b)
}

type DirectoryBuilder struct {
	title    string
	startDir string
}

func Directory() *DirectoryBuilder {
	return &DirectoryBuilder{}
}

func (b *DirectoryBuilder) Title(title string) *DirectoryBuilder {
	b.title = title
	return b
}

func (b *DirectoryBuilder) SetStartDir(dir string) *DirectoryBuilder {
	b.startDir = dir
	return b
}

func (b *DirectoryBuilder) Browse() (string, error) {
	return browseDirectory(b)
}

type MsgBuilder struct {
	msg   string
	title string
}

func Message(format string, args ...any) *MsgBuilder {
	return &MsgBuilder{msg: sprintf(format, args...)}
}

func (b *MsgBuilder) Title(title string) *MsgBuilder {
	b.title = title
	return b
}

func (b *MsgBuilder) Info() {
	messageInfo(b)
}

func (b *MsgBuilder) Error() {
	messageError(b)
}

func (b *MsgBuilder) YesNo() bool {
	return messageYesNo(b, false)
}

func (b *MsgBuilder) ErrorYesNo() bool {
	return messageYesNo(b, true)
}
