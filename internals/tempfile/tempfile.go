package tempfile

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

type Kind struct {
	Prefix string
	Suffix string
}

var Download = Kind{Prefix: "ws-cli-download-", Suffix: ".zip"}

var (
	exit       = os.Exit
	executable = os.Executable
	currentUID = os.Getuid
	pollEvery  = 100 * time.Millisecond
)

func (k Kind) Create(label string) (*os.File, error) {
	return os.CreateTemp("", k.Prefix+label+"-*"+k.Suffix)
}

func (k Kind) Owns(path string) bool {
	if !filepath.IsAbs(path) || filepath.Dir(path) != filepath.Clean(os.TempDir()) {
		return false
	}

	if matched, _ := filepath.Match(k.Prefix+"*"+k.Suffix, filepath.Base(path)); !matched {
		return false
	}

	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return false
	}

	stat, ok := info.Sys().(*syscall.Stat_t)

	return ok && int(stat.Uid) == currentUID()
}

func (k Kind) Remove(path string) error {
	if !k.Owns(path) {
		return k.refusal("remove", path)
	}

	return os.Remove(path)
}

func (k Kind) Sweep(age time.Duration) {
	matches, _ := filepath.Glob(filepath.Join(os.TempDir(), k.Prefix+"*"+k.Suffix))

	for _, path := range matches {
		if info, err := os.Lstat(path); err == nil && time.Since(info.ModTime()) > age {
			_ = k.Remove(path)
		}
	}
}

func (k Kind) Trap(path string) (release func()) {
	signals, done := make(chan os.Signal, 1), make(chan struct{})
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)

	go func() {
		select {
		case received := <-signals:
			_ = k.Remove(path)
			exit(128 + int(received.(syscall.Signal)))
		case <-done:
		}
	}()

	return func() {
		signal.Stop(signals)
		close(done)
	}
}

func (k Kind) Reap(path string, limit time.Duration) error {
	if !k.Owns(path) {
		return k.refusal("reap", path)
	}

	for deadline := time.Now().Add(limit); time.Now().Before(deadline) && !isOpen(path); {
		time.Sleep(pollEvery)
	}

	return k.Remove(path)
}

func (k Kind) refusal(action, path string) error {
	return fmt.Errorf("refusing to %s %s: not a %s*%s file of this user in %s", action, path, k.Prefix, k.Suffix, os.TempDir())
}

func Detach(args ...string) error {
	path, err := executable()
	if err != nil {
		return err
	}

	command := exec.Command(path, args...)
	command.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := command.Start(); err != nil {
		return err
	}

	return command.Process.Release()
}

func isOpen(path string) bool {
	links, _ := filepath.Glob("/proc/[0-9]*/fd/*")

	for _, link := range links {
		if target, err := os.Readlink(link); err == nil && target == path {
			return true
		}
	}

	return false
}
