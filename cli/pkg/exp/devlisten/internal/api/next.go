package api

import (
	"strconv"
	"strings"
)

// command builds a `next` CLI command. Every value the caller sent is
// single-quoted for a POSIX shell. Values the server generates (cursors,
// view names) are bare. A captured value never goes into a command.
type command struct{ parts []string }

func newCommand(base string) *command { return &command{parts: []string{base}} }

func (c *command) flag(name string) *command {
	c.parts = append(c.parts, "--"+name)
	return c
}

// bare adds a server-generated value.
func (c *command) bare(name, value string) *command {
	c.parts = append(c.parts, "--"+name, value)
	return c
}

func (c *command) num(name string, n uint64) *command {
	return c.bare(name, strconv.FormatUint(n, 10))
}

// quoted adds one caller value per occurrence.
func (c *command) quoted(name string, values ...string) *command {
	for _, v := range values {
		c.parts = append(c.parts, "--"+name, shellQuote(v))
	}
	return c
}

func (c *command) String() string { return strings.Join(c.parts, " ") }

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func intStrings(ns []int) []string {
	out := make([]string, len(ns))
	for i, n := range ns {
		out[i] = strconv.Itoa(n)
	}
	return out
}
