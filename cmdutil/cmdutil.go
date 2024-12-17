package cmdutil

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	"github.com/teejays/gokutil/errutil"
	jsonutil "github.com/teejays/gokutil/gopi/json"
	"github.com/teejays/gokutil/log"
	"github.com/teejays/gokutil/log/logwriter"
	"github.com/teejays/gokutil/sclog"
)

type ExecOptions struct {
	// FailureKeyWords is a list of keywords that, if found in the output of the command, will be considered as a failure.
	// The command will return an error in this case.
	FailureKeyWords []string
	IsLoudCommand   bool // If true, means that the command logs its output generally as errors (e.g. docker commands) and hence we'll log them all as non-error/warn
	ExtraEnvs       []string
}

func IsMacOS(ctx context.Context) bool {
	return runtime.GOOS == "darwin"
}

func ExecCmdFromDir(ctx context.Context, dir string, name string, req ...string) error {
	var err error
	cmd := exec.CommandContext(ctx, name, req...)
	cmd.Dir, err = filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("Getting absolute path of the specified working dir [%s]: %w", dir, err)
	}
	return ExecOSCmd(ctx, cmd)
}

func ExecCmd(ctx context.Context, name string, req ...string) error {
	cmd := exec.CommandContext(ctx, name, req...)
	return ExecOSCmd(ctx, cmd)
}

func ExecOSCmdWithOpts(ctx context.Context, cmd *exec.Cmd, opts ExecOptions) error {

	startOpts := StartOptions{
		IsLoudCommand:      opts.IsLoudCommand,
		ExtraEnvs:          opts.ExtraEnvs,
		FailureKeyWords:    opts.FailureKeyWords,
		RefFailureDetected: &atomic.Bool{},
	}

	err := StartOSCmd(ctx, cmd, startOpts)
	if err != nil {
		return fmt.Errorf("Starting cmd: %s\n%w\n%s", cmd.String(), err, cmd.Stderr)
	}

	log.Debug(ctx, "Waiting on command...")
	err = cmd.Wait()
	if err != nil {
		// If the error corresponds to the user signalling the kill command, then we can ignore it
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.Error() == "signal: killed" {
				log.Debug(ctx, "Command was killed by user.", "command", cmd.String())
				return nil
			}
		}
		return errutil.Wrap(err, "Command did not run successfully.")
	}

	if startOpts.RefFailureDetected.Load() {
		return fmt.Errorf("Command failed. Detected failure keyword in output.")
	}

	return nil
}

func ExecOSCmd(ctx context.Context, cmd *exec.Cmd) error {
	return ExecOSCmdWithOpts(ctx, cmd, ExecOptions{})
}

type StartOptions struct {
	FailureKeyWords    []string
	IsLoudCommand      bool // If true, means that the command logs its output generally as errors (e.g. docker commands) and hence we'll log them all as non-error/warn
	ExtraEnvs          []string
	RefFailureDetected *atomic.Bool
}

func StartOSCmd(ctx context.Context, cmd *exec.Cmd, opts StartOptions) error {
	log.Debug(ctx, "Running command...", "command", cmd.String(), "dir", cmd.Dir)
	if opts.ExtraEnvs != nil && len(opts.ExtraEnvs) > 0 {
		if cmd.Env == nil || len(cmd.Env) == 0 {
			cmd.Env = os.Environ()
		}
		cmd.Env = append(cmd.Env, opts.ExtraEnvs...)
	}
	if cmd.Env != nil && len(cmd.Env) > 0 {
		log.None(ctx, "Running command (with extras env)...", "command", cmd.String(), "dir", cmd.Dir, "extraEnv", jsonutil.MustPrettyPrint(opts.ExtraEnvs))
	}

	detectFailureKeywords := func(ctx context.Context, s string) {
		// if command has already failed, don't do anything
		if opts.RefFailureDetected == nil {
			return
		}
		if opts.RefFailureDetected.Load() {
			return
		}
		for _, kw := range opts.FailureKeyWords {
			if strings.Contains(s, kw) {
				// Kill the command
				log.Warn(ctx, "While running command, found failure keyword in output. Killing the command...", "keyword", kw, "log", s, "command", cmd.String())
				opts.RefFailureDetected.Store(true)
				// Sleep for a bit to let the logs come though
				time.Sleep(2 * time.Second)
				killErr := KillCommand(ctx, cmd)
				if killErr != nil {
					log.Error(ctx, "Error killing the command", "command", cmd.String(), "error", killErr)
				}
				break
			}
		}
	}

	// Log the output using the logger
	cmdLoggerCtx := sclog.ContextIndent(ctx, 8)
	cmdLoggerCtx = sclog.ContextSkipTimestamp(cmdLoggerCtx)
	cmdLoggerCtx = sclog.ContextSkipLevel(cmdLoggerCtx)

	outWriter := logwriter.NewLogWriter(strings.Repeat(sclog.IdentSpace, 2), "", func(s string) {
		s = RemoveANSI(s)
		log.Debug(cmdLoggerCtx, s)
		go detectFailureKeywords(cmdLoggerCtx, s)
	})
	errWriter := logwriter.NewLogWriter(strings.Repeat(sclog.IdentSpace, 2), "", func(s string) {
		s = RemoveANSI(s)
		log.Warn(cmdLoggerCtx, s)
		go detectFailureKeywords(cmdLoggerCtx, s)
	})
	if opts.IsLoudCommand {
		errWriter = outWriter
	}

	cmd.Stdout = outWriter // os.Stdout
	cmd.Stderr = errWriter // os.Stderr
	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("Starting command: %w", err)
	}

	return nil
}

// RemoveANSI removes all ANSI escape codes from a given string
func RemoveANSI(input string) string {
	re := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return re.ReplaceAllString(input, "")
}

type FindSedDiff struct {
	Old string
	New string
}

// FindSed finds all files in the given directory and replaces the old string with the new string
// e.g. replace `{{.goku_app_name}}` -> AppName
func FindSed(ctx context.Context, diffs []FindSedDiff, dirPath string) error {
	// /usr/bin/find /Users/teejays/Development/goku-workspace/projects/tailgate/apps/.goku/static/admin -type f -exec sed -i '' 's|{{.goku_app_name}}|tailgate|g' {} \;

	cmdParts := []string{
		"find",
		dirPath,
		"-type", "f",
		"-exec",
		"sed",
	}
	// Check GO OS arch: if on mac, use ” for -i, if on linux, use nothing
	// cmdParts = append(cmdParts, "-n") // This is to suppress the output of the sed command
	// Note: Removing the -i flag -- since the sed was creating extra files (copy of og files)
	if IsMacOS(ctx) {
		cmdParts = append(cmdParts, "-i", "")
	} else {
		cmdParts = append(cmdParts, "-i")
	}
	// Combine multiple old/new pairs
	for _, diff := range diffs {
		cmdParts = append(cmdParts, "-e", fmt.Sprintf("s+%s+%s+g", diff.Old, diff.New))
	}
	cmdParts = append(cmdParts, "{}", "+")

	cmd := exec.CommandContext(ctx, cmdParts[0], cmdParts[1:]...)
	cmd.Env = append(os.Environ(), "LC_CTYPE=C", "LANG=C")
	err := ExecOSCmd(ctx, cmd)
	if err != nil {
		return err
	}
	return nil
}

func KillCommand(ctx context.Context, cmd *exec.Cmd) error {

	// Attempt 1
	log.Info(ctx, "Sending os.Interrupt signal to command...", "command", cmd.String())
	err := cmd.Process.Signal(os.Interrupt)
	if err != nil {
		return fmt.Errorf("Sending os.Interrupt signal to command: %w", err)
	}
	log.Info(ctx, "Waiting for the command to stop...", "command", cmd.String())
	err = cmd.Wait()
	if err == nil {
		log.Info(ctx, "Successfully stopped the command", "command", cmd.String())
		return nil
	}
	// If the error is not type *exec.ExitError, then we are okay since it means the process has already exited somehow
	if _, ok := err.(*exec.ExitError); ok {
		log.Debug(ctx, "Command server already stopped")
		return nil
	}

	log.Warn(ctx, "Command did not stop with os.Interrupt. Trying another approach...", "command", cmd.String(), "error", err)

	// Attempt 2
	log.Info(ctx, "Sending os.Kill signal to the command...", "command", cmd.String())
	err = cmd.Process.Signal(os.Kill)
	if err != nil {
		return fmt.Errorf("Sending os.Kill signal to the command: %w", err)
	}
	log.Info(ctx, "Waiting for the command to stop...", "command", cmd.String())
	err = cmd.Wait()
	if err == nil {
		log.Info(ctx, "Successfully stopped the command", "command", cmd.String())
		return nil
	}
	log.Error(ctx, "Error stopping the command", "error", err)

	// Attempt 3
	log.Info(ctx, "Killing process of the command...", "command", cmd.String())
	err = cmd.Process.Kill()
	if err != nil {
		return fmt.Errorf("Killing process of the command: %w", err)
	}
	log.Info(ctx, "Waiting for the command to stop...", "command", cmd.String())
	err = cmd.Wait()
	if err == nil {
		log.Info(ctx, "Successfully stopped the command", "command", cmd.String())
		return nil
	}
	log.Error(ctx, "Error stopping the command", "error", err, "command", cmd.String())

	return fmt.Errorf("Could not stop the command after multiple attempts")
}