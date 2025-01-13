package mainutil

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/alexflint/go-arg"
	"github.com/teejays/gokutil/errutil"
	"github.com/teejays/gokutil/gopi/json"
	"github.com/teejays/gokutil/log"
)

var ErrCleanExit = errors.New("clean exit")

type IArgs interface {
	Version() string
	ValidateAndProcess(ctx context.Context) error
	GetParentArgs() *ParentArgs
}

type ParentArgs struct {
	// Subcommands
	VersionSubCmd *struct{} `arg:"subcommand:version" help:"Print the version of the CLI. Also works with -v or --version."`
	HelpSubCmd    *struct{} `arg:"subcommand:help" help:"Print the help message. Also works with -h or --help."`

	// Flags
	LogLevelStr string `arg:"--log-level,env:GOKU_LOG_LEVEL" default:"info" help:"Set the log level. Options: trace, debug, info, warn, error"`
	LogLevel    slog.Level
}

func (a *ParentArgs) GetParentArgs() *ParentArgs {
	return a
}

func (a *ParentArgs) ValidateAndProcess(ctx context.Context) error {
	// Log level
	if a.LogLevelStr != "" {
		level, err := log.ParseLevel(a.LogLevelStr)
		if err != nil {
			return errutil.Wrap(err, "Parsing log level")
		}
		a.LogLevel = level
	}

	return nil
}

func ParseArgs(ctx context.Context, programName string, v IArgs) error {
	// Create a new args parser
	argParser, err := arg.NewParser(arg.Config{
		Program: programName,
	}, v)
	if err != nil {
		return errutil.Wrap(err, "Creating new args parser")
	}

	log.Debug(ctx, "Parsing command line args")

	parseErr := argParser.Parse(os.Args[1:])
	if parseErr != nil && !errors.Is(parseErr, arg.ErrHelp) && !errors.Is(parseErr, arg.ErrVersion) {
		return parseErr
	}

	// Process and validate. Parent args ValidateAndProcess should be called from within the outer ValidateAndProcess
	if err := v.ValidateAndProcess(ctx); err != nil {
		return errutil.Wrap(err, "Validating args")
	}

	parentArgs := v.GetParentArgs()
	if parentArgs.VersionSubCmd != nil || errors.Is(parseErr, arg.ErrVersion) {
		fmt.Print(v.Version())
		return ErrCleanExit
	}

	if parentArgs.HelpSubCmd != nil || errors.Is(parseErr, arg.ErrHelp) {
		argParser.WriteHelp(os.Stdout)
		return ErrCleanExit
	}

	log.Debug(ctx, "Parsed args", "args", json.MustPrettyPrint(v))

	return nil

}
