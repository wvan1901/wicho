package devlog

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"runtime"
	"strconv"
	"sync"
)

const (
	// FG: Foreground, BG: background
	ANSI_RESET_COLOR     = "\033[0m"
	ANSI_FG_BLACK        = 30
	ANSI_FG_RED          = 31
	ANSI_FG_GREEN        = 32
	ANSI_FG_YELLOW       = 33
	ANSI_FG_BLUE         = 34
	ANSI_FG_MAGENTA      = 35
	ANSI_FG_CYAN         = 36
	ANSI_FG_LIGHTGRAY    = 37
	ANSI_FG_DARKGRAY     = 90
	ANSI_FG_LIGHTRED     = 91
	ANSI_FG_LIGHTGREEN   = 92
	ANSI_FG_LIGHTYELLOW  = 93
	ANSI_FG_LIGHTBLUE    = 94
	ANSI_FG_LIGHTMAGENTA = 95
	ANSI_FG_LIGHTCYAN    = 96
	ANSI_FG_WHITE        = 97
	ANSI_BG_BLACK        = 40
	ANSI_BG_RED          = 41
	ANSI_BG_GREEN        = 42
	ANSI_BG_YELLOW       = 43
	ANSI_BG_BLUE         = 44
	ANSI_BG_MAGENTA      = 45
	ANSI_BG_CYAN         = 46
	ANSI_BG_LIGHTGRAY    = 47
	ANSI_BG_DARKGRAY     = 100
	ANSI_BG_LIGHTRED     = 101
	ANSI_BG_LIGHTGREEN   = 102
	ANSI_BG_LIGHTYELLOW  = 103
	ANSI_BG_LIGHTBLUE    = 104
	ANSI_BG_LIGHTMAGENTA = 105
	ANSI_BG_LIGHTCYAN    = 106
	ANSI_BG_WHITE        = 107
	TIME_FORMAT          = "[15:04:05.000]"
)

func colorString(colorAnsiCode int, v string) string {
	return fmt.Sprintf("\033[%sm%s%s", strconv.Itoa(colorAnsiCode), v, ANSI_RESET_COLOR)
}

func colorSimple(fgColor int, bgColor int, v string) string {
	if (fgColor < 30 || fgColor > 97) || (fgColor > 37 && fgColor < 90) {
		fgColor = 39
	}
	if (bgColor < 40 || bgColor > 107) || (bgColor > 47 && bgColor < 100) {
		bgColor = 49
	}
	return fmt.Sprintf("\033[%s;%sm%s%s", strconv.Itoa(fgColor), strconv.Itoa(bgColor), v, ANSI_RESET_COLOR)
}

type DevLogHandler struct {
	opts Options
	out  io.Writer
	goas []groupOrAttrs
	mu   *sync.Mutex
}

type Options struct {
	// Level reports the minimum level to log.
	// Levels with lower levels are discarded.
	// If nil, the Handler uses [slog.LevelInfo].
	Level slog.Leveler
	// Enables or disables source code location
	AddSource bool
	// TODO: Add color custumization
}

func New(out io.Writer, opts *Options) *DevLogHandler {
	h := &DevLogHandler{out: out, mu: &sync.Mutex{}}
	if opts != nil {
		h.opts = *opts
	}

	if h.opts.Level == nil {
		h.opts.Level = slog.LevelInfo
	}

	return h
}

func (h *DevLogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.opts.Level.Level()
}

func (h *DevLogHandler) Handle(ctx context.Context, r slog.Record) error {
	buf := make([]byte, 0, 1024)
	if !r.Time.IsZero() {
		buf = h.appendAttr(buf, slog.Time(slog.TimeKey, r.Time))
	}

	buf = fmt.Append(buf, handleLvl(r.Level)+" ")

	if r.PC != 0 && h.opts.AddSource {
		fs := runtime.CallersFrames([]uintptr{r.PC})
		f, _ := fs.Next()
		sourceStr := fmt.Sprintf(" %s:%d ", f.File, f.Line)
		colorVal := colorSimple(ANSI_FG_BLACK, ANSI_BG_LIGHTMAGENTA, sourceStr)
		buf = fmt.Append(buf, colorVal+" ")
	}
	buf = h.appendAttr(buf, slog.String(slog.MessageKey, r.Message))

	// Handle state from WithGroup and WithAttrs.
	goas := h.goas
	if r.NumAttrs() == 0 {
		// If the record has no Attrs, remove groups at the end of the list; they are empty.
		for len(goas) > 0 && goas[len(goas)-1].group != "" {
			goas = goas[:len(goas)-1]
		}
	}
	for _, goa := range goas {
		if goa.group != "" {
			buf = fmt.Appendf(buf, "%*s%s:\n", 4, "", goa.group)
		} else {
			for _, a := range goa.attrs {
				buf = h.appendAttr(buf, a)
			}
		}
	}
	r.Attrs(func(a slog.Attr) bool {
		buf = h.appendAttr(buf, a)
		return true
	})

	buf = append(buf, "\n"...)
	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.out.Write(buf)

	return err
}

func (h *DevLogHandler) appendAttr(buf []byte, a slog.Attr) []byte {
	// Resolve the Attr's value before doing anything else
	a.Value = a.Value.Resolve()
	// Ignore empty Attrs
	if a.Equal(slog.Attr{}) {
		return buf
	}
	switch a.Value.Kind() {
	case slog.KindString:
		keyStr := colorSimple(ANSI_FG_LIGHTBLUE, ANSI_BG_BLACK, a.Key)
		buf = fmt.Append(buf, keyStr+"="+a.Value.String())
	case slog.KindTime:
		// Write the time in a standard way
		timeStr := fmt.Sprintf("%s", a.Value.Time().Format(TIME_FORMAT))
		colorStr := colorSimple(ANSI_FG_BLACK, ANSI_BG_LIGHTGREEN, timeStr)
		buf = fmt.Append(buf, colorStr)
	case slog.KindBool:
		keyStr := colorSimple(ANSI_FG_LIGHTRED, ANSI_BG_BLACK, a.Key)
		buf = fmt.Appendf(buf, "%s=%s", keyStr, a.Value)
	case slog.KindInt64:
		keyStr := colorSimple(ANSI_FG_LIGHTCYAN, ANSI_BG_BLACK, a.Key)
		buf = fmt.Appendf(buf, "%s=%s", keyStr, a.Value)
	case slog.KindGroup:
		attrs := a.Value.Group()
		// Ignore empty groups
		if len(attrs) == 0 {
			return buf
		}
		// If key is non empty, write it out
		// Otherwise inline the attrs
		if a.Key != "" {
			keyStr := colorSimple(0, ANSI_BG_BLUE, " "+a.Key+" ")
			startStr := colorSimple(0, ANSI_BG_GREEN, " START ")
			buf = fmt.Appendf(buf, "%s%s ", keyStr, startStr)
		}
		for _, ga := range attrs {
			buf = h.appendAttr(buf, ga)
		}
		if a.Key != "" {
			keyStr := colorSimple(0, ANSI_BG_BLUE, " "+a.Key+" ")
			endStr := colorSimple(0, ANSI_BG_RED, " END ")
			buf = fmt.Appendf(buf, "%s%s", keyStr, endStr)
		}
	default:
		keyStr := colorSimple(ANSI_FG_LIGHTGREEN, ANSI_BG_BLACK, a.Key)
		buf = fmt.Appendf(buf, "%s=%s", keyStr, a.Value)
	}

	buf = fmt.Append(buf, " ")

	return buf
}

// groupOrAttrs holds either a group name or a list of slog.Attrs.
type groupOrAttrs struct {
	group string      // group name if non-empty
	attrs []slog.Attr // attrs if non-empty
}

func (h *DevLogHandler) withGroupOrAttrs(goa groupOrAttrs) *DevLogHandler {
	h2 := *h
	h2.goas = make([]groupOrAttrs, len(h.goas)+1)
	copy(h2.goas, h.goas)
	h2.goas[len(h2.goas)-1] = goa
	return &h2
}

func (h *DevLogHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	return h.withGroupOrAttrs(groupOrAttrs{group: name})
}

func (h *DevLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	return h.withGroupOrAttrs(groupOrAttrs{attrs: attrs})
}

func handleLvl(lvl slog.Level) string {
	lvlStr := " " + lvl.String() + " "
	switch lvl {
	case slog.LevelDebug:
		return colorSimple(ANSI_FG_BLACK, ANSI_BG_DARKGRAY, lvlStr)
	case slog.LevelInfo:
		return colorSimple(ANSI_FG_BLACK, ANSI_BG_CYAN, lvlStr+" ")
	case slog.LevelWarn:
		return colorSimple(ANSI_FG_BLACK, ANSI_BG_LIGHTYELLOW, lvlStr+" ")
	case slog.LevelError:
		return colorSimple(ANSI_FG_BLACK, ANSI_BG_LIGHTRED, lvlStr)
	}
	return colorSimple(ANSI_FG_BLACK, ANSI_BG_WHITE, lvlStr)
}
