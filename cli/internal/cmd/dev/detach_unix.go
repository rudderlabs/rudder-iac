//go:build unix

package dev

import (
	"os"
	"os/exec"
	"syscall"

	"golang.org/x/sys/unix"
)

// setDetachAttrs starts the child in its own session, so a terminal hangup
// or Ctrl-C in the parent's shell does not reach it.
func setDetachAttrs(c *exec.Cmd) {
	c.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}

// detachStdio points stdin, stdout and stderr at /dev/null after the ready
// line, so the parent can exit and close its pipe ends without the child
// dying of SIGPIPE on a later write.
func detachStdio() {
	null, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		log.Error("opening /dev/null for the detached child", "error", err)
		return
	}
	for _, fd := range []int{0, 1, 2} {
		_ = unix.Dup2(int(null.Fd()), fd)
	}
}
