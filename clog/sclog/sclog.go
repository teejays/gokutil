package sclog

import (
	"context"
	"fmt"
	"io"
	"log/slog"

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
	color       bool
	timestamp   bool
	out         io.Writer
	level       slog.Level
	commonAttrs []slog.Attr
}

type NewHandlerRequest struct {
	Out       io.Writer
	Level     slog.Level
	Color     bool
	Timestamp bool
}

func NewHandler(req NewHandlerRequest) Handler {
	return Handler{
		color:       req.Color,
		timestamp:   req.Timestamp,
		out:         req.Out,
		level:       req.Level,
		commonAttrs: []slog.Attr{},
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
	msg := fmt.Sprintf("[%s] %s", rec.Level, rec.Message)

	if rec.NumAttrs() > 0 {
		attrMsg := ""
		rec.Attrs(func(a slog.Attr) bool {
			attrMsg += fmt.Sprintf("\n  - %s: %v", a.Key, a.Value)
			return true
		})
		msg += attrMsg
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
