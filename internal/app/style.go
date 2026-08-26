package app

import (
	"fmt"
	"io"
	"os"
)

const (
	ansiReset  = "\x1b[0m"
	ansiBold   = "\x1b[1m"
	ansiDim    = "\x1b[2m"
	ansiRed    = "\x1b[31m"
	ansiGreen  = "\x1b[32m"
	ansiYellow = "\x1b[33m"
	ansiCyan   = "\x1b[36m"
)

// styler keeps terminal presentation separate from output content. Machine-
// readable YAML and JSON never pass through it.
type styler struct {
	enabled bool
}

func styleFor(w io.Writer) styler {
	if noColorRequested() || os.Getenv("TERM") == "dumb" {
		return styler{}
	}
	f, ok := w.(*os.File)
	if !ok {
		return styler{}
	}
	info, err := f.Stat()
	return styler{enabled: err == nil && info.Mode()&os.ModeCharDevice != 0}
}

func noColorRequested() bool {
	return os.Getenv("NO_COLOR") != ""
}

func (s styler) wrap(code, text string) string {
	if !s.enabled {
		return text
	}
	return code + text + ansiReset
}

func (s styler) bold(text string) string   { return s.wrap(ansiBold, text) }
func (s styler) dim(text string) string    { return s.wrap(ansiDim, text) }
func (s styler) red(text string) string    { return s.wrap(ansiRed, text) }
func (s styler) green(text string) string  { return s.wrap(ansiGreen, text) }
func (s styler) yellow(text string) string { return s.wrap(ansiYellow, text) }
func (s styler) cyan(text string) string   { return s.wrap(ansiCyan, text) }

func printError(w io.Writer, err error) {
	s := styleFor(w)
	fmt.Fprintf(w, "%s %v\n", s.red(s.bold("error:")), err)
}
