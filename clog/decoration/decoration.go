package decoration

import (
	"fmt"
	"regexp"
)

/********************************************************************************
* D E C O R A T I O N 															*
*********************************************************************************/

// Decoration represents an ANSI escape sequence, that can be used to format a message
// logged to the standard out (terminal).
type Decoration string

const (
	RESET Decoration = "\x1b[0m"

	// decorations
	BRIGHT     Decoration = "\x1b[1m"
	DIM        Decoration = "\x1b[2m"
	UNDERSCORE Decoration = "\x1b[4m"
	BLINK      Decoration = "\x1b[5m"
	REVERSE    Decoration = "\x1b[7m"
	HIDDEN     Decoration = "\x1b[8m"

	// foreground colors represent the color of the logged text
	FG_BLACK         Decoration = "\x1b[30m"
	FG_RED           Decoration = "\x1b[31m"
	FG_GREEN         Decoration = "\x1b[32m"
	FG_YELLOW        Decoration = "\x1b[33m"
	FG_BLUE          Decoration = "\x1b[34m"
	FG_MAGENTA       Decoration = "\x1b[35m"
	FG_CYAN          Decoration = "\x1b[36m"
	FG_WHITE         Decoration = "\x1b[37m"
	FG_GRAY_LIGHT    Decoration = "\x1b[90m"
	FG_RED_LIGHT     Decoration = "\x1b[91m"
	FG_GREEN_LIGHT   Decoration = "\x1b[92m"
	FG_YELLOW_LIGHT  Decoration = "\x1b[93m"
	FG_BLUE_LIGHT    Decoration = "\x1b[94m"
	FG_MAGENTA_LIGHT Decoration = "\x1b[95m"
	FG_CYAN_LIGHT    Decoration = "\x1b[96m"
	FG_WHITE_LIGHT   Decoration = "\x1b[97m"

	// background colors represent the background color of the logged text
	BG_BLACK   Decoration = "\x1b[40m"
	BG_RED     Decoration = "\x1b[41m"
	BG_GREEN   Decoration = "\x1b[42m"
	BG_YELLOW  Decoration = "\x1b[43m"
	BG_BLUE    Decoration = "\x1b[44m"
	BG_MAGENTA Decoration = "\x1b[45m"
	BG_CYAN    Decoration = "\x1b[46m"
	BG_WHITE   Decoration = "\x1b[47m"
)

// NewDecoration takes a string representation of sgr code (ANSI), casts it as a Decoration, and returns it. It panics if the sgrCode is not
// a valid ansi escape sequence code.
func NewDecoration(sgrCode string) Decoration {
	// verify that it's an ansi code
	// regex from: https://superuser.com/questions/380772/removing-ansi-color-codes-from-text-stream
	reg := regexp.MustCompile(`^\x1b\[[0-9;]*[mG]$`)
	if !reg.MatchString(sgrCode) {
		panic(fmt.Sprintf("Clog.Decoration: invalid sgr code '%s' provided", sgrCode))
	}
	return Decoration(sgrCode)
}

// Decorate takes a message and a list of decorations, and returns the message with the decorations applied.
func Decorate(msg string, Decorations ...Decoration) string {
	var decorationsCode string
	for _, d := range Decorations {
		decorationsCode += string(d)
	}
	if decorationsCode == "" {
		return msg
	}
	return fmt.Sprintf("%s%s%s", decorationsCode, msg, RESET)
}
