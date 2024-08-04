package sclog

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/teejays/gokutil/clog/decoration"
)

var colorMap = map[slog.Level]decoration.Decoration{
	slog.LevelDebug: decoration.FG_GRAY_LIGHT,
	slog.LevelInfo:  decoration.FG_GREEN,
	slog.LevelWarn:  decoration.FG_YELLOW,
	slog.LevelError: decoration.FG_RED,
}

// Handler is a struct that implements the slog.Handler interface.
type Handler struct {
	color     bool
	timestamp bool
	out       io.Writer
	level     slog.Level

	commonAttrs    []slog.Attr
	prefixHeadings []string
}

type NewHandlerRequest struct {
	Out       io.Writer
	Level     slog.Level
	Color     bool
	Timestamp bool
}

func NewHandler(req NewHandlerRequest) Handler {
	return Handler{
		color:          req.Color,
		timestamp:      req.Timestamp,
		out:            req.Out,
		level:          req.Level,
		commonAttrs:    []slog.Attr{},
		prefixHeadings: []string{},
	}
}

func (l Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return l.level <= level
}

// Handle implements the slog.Handler interface. It is the main method that logs the message.
// Sample output:
//
//	2024/08/04 17:24:19 [DEBUG] GetUserIDsSelectQueryBuilder
//	  - Request: {Filter:{ID:<nil> Name:<nil> Email:0x14000420040 OrganizationID:<nil> HavingAddresses:<nil> PastOrganizationIDs:<nil> AuthCredential:<nil> CreatedAt:<nil> UpdatedAt:<nil> DeletedAt:<nil> And:[] Or:[]}}
//	  - OtherAttr: Value
func (l Handler) Handle(ctx context.Context, rec slog.Record) error {

	// [LEVEL]
	msg := "[" + rec.Level.String() + "]"

	// [LEVEL] [Prefix 1] [Prefix 2] ...
	for _, prefix := range l.prefixHeadings {
		msg += " " + "[" + prefix + "]"
	}

	// [LEVEL] [Prefix 1] [Prefix 2] ... Message
	msg = msg + " " + rec.Message

	if rec.NumAttrs() > 0 {
		spliter := "\n"
		attrMsg := ""
		rec.Attrs(func(a slog.Attr) bool {
			attrMsg += fmt.Sprintf("  - %s: %v%s", a.Key, a.Value, spliter)
			return true
		})
		attrMsg = strings.TrimSuffix(attrMsg, spliter)
		msg += ("\n" + attrMsg)
	}

	if l.color {
		msg = decoration.Decorate(msg, colorMap[rec.Level])
	}

	if l.timestamp {
		msg = fmt.Sprintf("%s %s", rec.Time.Format("2006/01/02 15:04:05"), msg)
	}

	_, err := l.out.Write([]byte(msg + "\n"))
	return err
}

func (l Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	l.commonAttrs = append(l.commonAttrs, attrs...)
	return l
}

func (l Handler) WithGroup(str string) slog.Handler {
	// Not implemented for now
	return l
}

// Custom Methods

// WithHeading adds a prefix to the log message. This is useful when you want to group logs together.
// It returns a new copy of the Handler with the prefix added.
func (l Handler) WithHeading(str string) Handler {
	lCopy := copy(l)
	lCopy.prefixHeadings = append(lCopy.prefixHeadings, str)
	return lCopy
}

func copy(l Handler) Handler {
	return Handler{
		color:          l.color,
		timestamp:      l.timestamp,
		out:            l.out,
		level:          l.level,
		commonAttrs:    l.commonAttrs,
		prefixHeadings: l.prefixHeadings,
	}
}
