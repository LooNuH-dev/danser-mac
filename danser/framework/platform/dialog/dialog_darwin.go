package dialog

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func sprintf(format string, args ...any) string {
	if len(args) == 0 {
		return format
	}

	return fmt.Sprintf(format, args...)
}

// runScript runs AppleScript with user strings passed as argv, so nothing has to be escaped.
// Returns ErrCancelled when the user dismisses the dialog (AppleScript error -128).
func runScript(script string, args ...string) (string, error) {
	cmdArgs := []string{}
	for _, line := range strings.Split("on run argv\n"+script+"\nend run", "\n") {
		cmdArgs = append(cmdArgs, "-e", line)
	}

	cmdArgs = append(cmdArgs, args...)

	out, err := exec.Command("osascript", cmdArgs...).Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && strings.Contains(string(exitErr.Stderr), "-128") {
			return "", ErrCancelled
		}

		if errors.As(err, &exitErr) {
			return "", fmt.Errorf("osascript: %s", strings.TrimSpace(string(exitErr.Stderr)))
		}

		return "", err
	}

	return strings.TrimSuffix(string(out), "\n"), nil
}

func existingDir(dir string) string {
	if st, err := os.Stat(dir); err == nil && st.IsDir() {
		return dir
	}

	return ""
}

func fileScript(b *FileBuilder, multiple bool) (string, []string) {
	args := []string{b.title}

	cmd := "choose file with prompt (item 1 of argv)"

	if dir := existingDir(b.startDir); dir != "" {
		args = append(args, dir)
		cmd += " default location (POSIX file (item 2 of argv))"
	}

	var exts []string
	for _, f := range b.filters {
		for _, e := range f.extensions {
			if e = strings.Trim(strings.TrimSpace(e), ".*\""); e != "" {
				exts = append(exts, "\""+e+"\"")
			}
		}
	}

	if len(exts) > 0 {
		cmd += " of type {" + strings.Join(exts, ", ") + "}"
	}

	var sb strings.Builder
	sb.WriteString("activate\n")

	if multiple {
		sb.WriteString("set res to (" + cmd + " with multiple selections allowed)\n")
		sb.WriteString("set out to \"\"\nrepeat with f in res\nset out to out & POSIX path of f & linefeed\nend repeat\nreturn out")
	} else {
		sb.WriteString("return POSIX path of (" + cmd + ")")
	}

	return sb.String(), args
}

func loadFile(b *FileBuilder) (string, error) {
	script, args := fileScript(b, false)
	return runScript(script, args...)
}

func loadFiles(b *FileBuilder) ([]string, error) {
	script, args := fileScript(b, true)

	out, err := runScript(script, args...)
	if err != nil {
		return nil, err
	}

	var paths []string
	for _, p := range strings.Split(out, "\n") {
		if p = strings.TrimSpace(p); p != "" {
			paths = append(paths, p)
		}
	}

	return paths, nil
}

func browseDirectory(b *DirectoryBuilder) (string, error) {
	args := []string{b.title}
	cmd := "choose folder with prompt (item 1 of argv)"

	if dir := existingDir(b.startDir); dir != "" {
		args = append(args, dir)
		cmd += " default location (POSIX file (item 2 of argv))"
	}

	p, err := runScript("activate\nreturn POSIX path of ("+cmd+")", args...)

	return strings.TrimSuffix(p, "/"), err
}

func title(b *MsgBuilder) string {
	if b.title != "" {
		return b.title
	}

	return "danser"
}

func messageInfo(b *MsgBuilder) {
	_, _ = runScript("activate\ndisplay dialog (item 1 of argv) with title (item 2 of argv) buttons {\"OK\"} default button \"OK\" with icon note", b.msg, title(b))
}

func messageError(b *MsgBuilder) {
	_, _ = runScript("activate\ndisplay dialog (item 1 of argv) with title (item 2 of argv) buttons {\"OK\"} default button \"OK\" with icon stop", b.msg, title(b))
}

func messageYesNo(b *MsgBuilder, isError bool) bool {
	icon := "note"
	if isError {
		icon = "stop"
	}

	out, err := runScript("activate\nset r to display dialog (item 1 of argv) with title (item 2 of argv) buttons {\"No\", \"Yes\"} default button \"Yes\" cancel button \"No\" with icon "+icon+"\nreturn button returned of r", b.msg, title(b))

	return err == nil && out == "Yes"
}
