package cmdutil

import (
	"context"
	"fmt"
	"io"
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
	Dir string
	// FailureKeyWords is a list of keywords that, if found in the output of the command, will be considered as a failure.
	// The command will return an error in this case.
	FailureKeyWords []string
	// Deprecated: This does not do anything.
	IsLoudCommand bool // If true, means that the command logs its output generally as errors (e.g. docker commands) and hence we'll log them all as non-error/warn
	ExtraEnvs     []string
	OutWriter     io.Writer
	ErrWriter     io.Writer
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
	timerStart := time.Now()
	defer func() {
		log.Debug(ctx, "Command execution time", "command", cmd.String(), "time", time.Since(timerStart))
	}()

	startOpts := StartOptions{
		ExecOptions:        opts,
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
				log.Warn(ctx, "Command received a kill signal (by the user?)", "command", cmd.String())
				return errutil.Wrap(exitErr, "Command received a kill signal")
			}
		}
		return errutil.Wrap(err, "Command did not run successfully")
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
	ExecOptions
	RefFailureDetected *atomic.Bool
}

func StartOSCmd(ctx context.Context, cmd *exec.Cmd, opts StartOptions) error {
	log.Debug(ctx, "Running command...", "command", cmd.String(), "opts", jsonutil.MustPrettyPrint(opts))
	if len(opts.ExtraEnvs) > 0 {
		if len(cmd.Env) == 0 {
			cmd.Env = os.Environ()
		}
		cmd.Env = append(cmd.Env, opts.ExtraEnvs...)
	}
	if opts.Dir != "" {
		if cmd.Dir != "" {
			log.Warn(ctx, "Command already has a working directory set. Overwriting it...", "oldDir", cmd.Dir, "newDir", opts.Dir)
		}
		cmd.Dir = opts.Dir
	}

	hasFailed := opts.RefFailureDetected
	if hasFailed == nil {
		hasFailed = &atomic.Bool{}
	}

	detectFailureKeywords := func(ctx context.Context, s string) {
		// if command has already failed, don't do anything
		if hasFailed.Load() {
			return
		}
		for _, kw := range opts.FailureKeyWords {
			if !strings.Contains(s, kw) {
				continue
			}
			if hasFailed.Load() { // check again
				return
			}
			// Kill the command
			hasFailed.Store(true)
			log.Info(ctx, "While running command, found failure keyword in output. Killing the command...", "keyword", kw, "log", s, "command", cmd.String())

			// Sleep for a bit to let the logs come though
			time.Sleep(2 * time.Second)

			killErr := KillCommand(ctx, cmd)
			if killErr != nil {
				log.Error(ctx, "Error killing the command", "command", cmd.String(), "error", killErr)
			}

		}
	}

	defaultOutWriter := GetDefaultStdOut(ctx)
	defaultErrWriter := GetDefaultStdErr(ctx)
	cmd.Stdout = logwriter.NewLogWriter(func(s string) {
		w := defaultOutWriter
		if opts.OutWriter != nil {
			w = opts.OutWriter
		}
		_, inerr := w.Write([]byte(s))
		if inerr != nil {
			log.Warn(ctx, "There was an error writing to the cmd's output writer", "cmd", cmd.String(), "error", inerr, "log", s)
		}
		detectFailureKeywords(ctx, s)
	})
	cmd.Stderr = logwriter.NewLogWriter(func(s string) {
		w := defaultErrWriter
		if opts.IsLoudCommand {
			w = defaultOutWriter
		}
		if opts.ErrWriter != nil {
			w = opts.ErrWriter
		}
		_, inerr := w.Write([]byte(s))
		if inerr != nil {
			log.Warn(ctx, "There was an error writing to the cmd's error writer", "cmd", cmd.String(), "error", inerr, "log", s)
		}
		detectFailureKeywords(ctx, s)
	})

	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("Starting command: %w", err)
	}

	return nil
}

func GetDefaultStdOut(ctx context.Context) io.Writer {
	// Log the output using the logger
	cmdLoggerCtx := sclog.ContextIndent(ctx, 8)
	cmdLoggerCtx = sclog.ContextSkipTimestamp(cmdLoggerCtx)
	cmdLoggerCtx = sclog.ContextSkipLevel(cmdLoggerCtx)
	return logwriter.NewLogWriter(
		func(s string) {
			s = RemoveANSI(s)
			// Strip any trailing newlines
			s = strings.TrimRight(s, "\n")
			log.Debug(cmdLoggerCtx, s)
		},
	)
}

func GetDefaultStdErr(ctx context.Context) io.Writer {
	// Log the output using the logger
	cmdLoggerCtx := sclog.ContextIndent(ctx, 8)
	cmdLoggerCtx = sclog.ContextSkipTimestamp(cmdLoggerCtx)
	cmdLoggerCtx = sclog.ContextSkipLevel(cmdLoggerCtx)
	return logwriter.NewLogWriter(
		func(s string) {
			s = RemoveANSI(s)
			// Strip any trailing newlines
			s = strings.TrimRight(s, "\n")
			log.Warn(cmdLoggerCtx, s)
		},
	)
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

type Creds struct {
	Username string
	Token    string
}
type DockerBuildReq struct {
	Creds           Creds  // Dockerhub creds, only used if passed
	DockerfilePath  string // Full path to the Dockerfile
	DockerImageRepo string // e.g. iamteejay/goku
	DockerImageTag  string // e.g. og-img-...
	BuildArgs       map[string]string
	NoPush          bool
	Platforms       []string // e.g. linux/amd64,linux/arm64.
}

func DockerLogin(ctx context.Context, username, password string) error {
	// Login using the docker login command but password using stdin, and pipe the password in
	cmd := exec.CommandContext(ctx, "docker", "login", "--username", username, "--password-stdin")
	cmd.Env = append(os.Environ(), "LC_CTYPE=C", "LANG=C")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("Getting stdin pipe: %w", err)
	}
	go func() {
		defer stdin.Close()
		_, inerr := io.WriteString(stdin, password)
		if inerr != nil {
			log.Error(ctx, "Error writing docker password to [docker login] stdin", "error", inerr)
		}
	}()

	err = ExecOSCmd(ctx, cmd)
	if err != nil {
		return err
	}

	return nil
}

func DockerImageBuild(ctx context.Context, req DockerBuildReq, opts ExecOptions) error {

	if req.Creds != (Creds{}) {
		log.Debug(ctx, "Logging into docker...", "username", req.Creds.Username)
		err := DockerLogin(ctx, req.Creds.Username, req.Creds.Token)
		if err != nil {
			return errutil.Wrap(err, "Logging into docker")
		}
	}

	cmdParts := []string{
		"docker", "buildx", "build",
	}
	if len(req.Platforms) > 0 {
		cmdParts = append(cmdParts, "--platform", strings.Join(req.Platforms, ","))
	} else {
		log.Warn(ctx, "No platforms specified for docker build. Using default linux/amd64,linux/arm64")
		cmdParts = append(cmdParts, "--platform", "linux/amd64,linux/arm64")
	}
	cmdParts = append(cmdParts,
		"-t", fmt.Sprintf("%s:%s", req.DockerImageRepo, req.DockerImageTag),
		"-f", req.DockerfilePath,
	)

	// Pass the envs
	for k, v := range req.BuildArgs {
		cmdParts = append(cmdParts, "--build-arg", fmt.Sprintf("%s=%s", k, v))
	}

	cmdParts = append(cmdParts, "--pull")
	if !req.NoPush {
		cmdParts = append(cmdParts, "--push")
	}
	cmdParts = append(cmdParts, ".")

	opts.IsLoudCommand = true

	cmd := exec.CommandContext(ctx, cmdParts[0], cmdParts[1:]...)
	// cmd.Dir = cfg.AppRootPath.Full
	err := ExecOSCmdWithOpts(ctx, cmd, opts)
	if err != nil {
		return errutil.Wrap(err, "Executing command: %v", cmd)
	}

	return nil
}
