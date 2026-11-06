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
}

var Download = Kind{Prefix: "ws-cli-download-"}

var (
	exit       = os.Exit
	executable = os.Executable
	currentUID = os.Getuid
	pollEvery  = 100 * time.Millisecond
)

func (k Kind) Create(name string) (*os.File, error) {
	if name == "" || filepath.Base(name) != name || name == "." || name == ".." {
		return nil, fmt.Errorf("%q is not a file name", name)
	}

	folder, err := os.MkdirTemp("", k.Prefix+"*")
	if err != nil {
		return nil, err
	}

	file, err := os.OpenFile(filepath.Join(folder, name), os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		_ = os.Remove(folder)
		return nil, err
	}

	return file, nil
}

func (k Kind) Owns(path string) bool {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path || !k.ownsFolder(filepath.Dir(path)) {
		return false
	}

	info, err := os.Lstat(path)

	return err == nil && info.Mode().IsRegular()
}

func (k Kind) Remove(path string) error {
	if !k.Owns(path) {
		return k.refusal("remove", path)
	}

	if err := os.Remove(path); err != nil {
		return err
	}

	return os.Remove(filepath.Dir(path))
}

func (k Kind) Sweep(age time.Duration) {
	folders, _ := filepath.Glob(filepath.Join(os.TempDir(), k.Prefix+"*"))

	for _, folder := range folders {
		info, err := os.Lstat(folder)
		if err != nil || time.Since(info.ModTime()) <= age || !k.ownsFolder(folder) {
			continue
		}

		entries, err := os.ReadDir(folder)
		if err != nil || len(entries) > 1 {
			continue
		}

		if len(entries) == 1 {
			_ = k.Remove(filepath.Join(folder, entries[0].Name()))
		} else {
			_ = os.Remove(folder)
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

	for deadline := time.Now().Add(limit); time.Now().Before(deadline) && !isOpen(path) && !wasRead(path); {
		time.Sleep(pollEvery)
	}

	return k.Remove(path)
}

func (k Kind) ownsFolder(folder string) bool {
	if filepath.Dir(folder) != filepath.Clean(os.TempDir()) {
		return false
	}

	if matched, _ := filepath.Match(k.Prefix+"*", filepath.Base(folder)); !matched {
		return false
	}

	info, err := os.Lstat(folder)
	if err != nil || !info.IsDir() {
		return false
	}

	stat, ok := info.Sys().(*syscall.Stat_t)

	return ok && int(stat.Uid) == currentUID()
}

func (k Kind) refusal(action, path string) error {
	return fmt.Errorf("refusing to %s %s: not a file in a %s* folder of this user in %s", action, path, k.Prefix, os.TempDir())
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

func wasRead(path string) bool {
	info, err := os.Lstat(path)
	if err != nil {
		return false
	}

	stat, ok := info.Sys().(*syscall.Stat_t)

	return ok && time.Unix(stat.Atim.Unix()).After(time.Unix(stat.Mtim.Unix()))
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
