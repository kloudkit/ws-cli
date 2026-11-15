package proc

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"syscall"
	"time"
)

var (
	Executable = os.Executable
	ErrTimeout = errors.New("timed out")
	waitDelay  = time.Second
	kill       = syscall.Kill
)

type Cmd struct {
	Path    string
	Args    []string
	Env     []string
	Stdout  io.Writer
	Stderr  io.Writer
	Timeout time.Duration
}

func (c Cmd) Run(parent context.Context) error {
	ctx, cancel := context.WithTimeout(parent, c.Timeout)
	defer cancel()

	command := exec.CommandContext(ctx, c.Path, c.Args...)
	command.Env, command.Stdout, command.Stderr = c.Env, c.Stdout, c.Stderr
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	command.WaitDelay = waitDelay
	command.Cancel = func() error { return killGroup(command) }

	err := command.Run()
	_ = killGroup(command)

	switch {
	case err == nil, errors.Is(err, exec.ErrWaitDelay):
		return nil
	case parent.Err() != nil:
		return parent.Err()
	case ctx.Err() != nil:
		return ErrTimeout
	}

	return err
}

func Output(timeout time.Duration, path string, args ...string) ([]byte, error) {
	var stdout bytes.Buffer
	err := Cmd{Path: path, Args: args, Stdout: &stdout, Timeout: timeout}.Run(context.Background())

	return stdout.Bytes(), err
}

func killGroup(command *exec.Cmd) error {
	if command.Process == nil {
		return nil
	}

	err := kill(-command.Process.Pid, syscall.SIGKILL)
	if errors.Is(err, syscall.ESRCH) {
		return os.ErrProcessDone
	}

	return err
}
