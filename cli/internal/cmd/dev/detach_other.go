//go:build !unix

package dev

import "os/exec"

func setDetachAttrs(*exec.Cmd) {}

func detachStdio() {}
