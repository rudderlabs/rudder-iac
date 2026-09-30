package dev

import (
	"errors"
	"regexp"
	"slices"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// flagError turns a cobra or pflag parse failure into the usage error
// object. Its next is a command that parses: the same command with a close
// flag, or the sibling command that has the flag.
func flagError(cmd *cobra.Command, err error) error {
	jsonFlag := cmd.Flags().Lookup("json")
	machine := jsonFlag != nil && jsonFlag.Changed || !isTerminal(cmd.ErrOrStderr())
	out := output{stdout: cmd.OutOrStdout(), stderr: cmd.ErrOrStderr(), flags: clientFlags{json: machine}}
	return out.fail(usageFor(cmd, err))
}

func usageFor(cmd *cobra.Command, err error) *cliError {
	var (
		notExist *pflag.NotExistError
		invalid  *pflag.InvalidValueError
		required *pflag.ValueRequiredError
	)
	switch {
	case errors.As(err, &notExist):
		return unknownFlag(cmd, strings.TrimLeft(notExist.GetSpecifiedName(), "-"))
	case errors.As(err, &invalid):
		return badValue(cmd, invalid.GetFlag(), err)
	case errors.As(err, &required):
		return badValue(cmd, required.GetFlag(), err)
	}
	return &cliError{Code: "usage", Message: err.Error(), Next: firstExample(cmd)}
}

func unknownFlag(cmd *cobra.Command, name string) *cliError {
	e := &cliError{Code: "usage", Message: "unknown flag: --" + name, Param: name,
		Details: map[string]any{"validFlags": flagNames(cmd)}}
	if close := closestFlag(cmd, name); close != nil {
		e.Message += "; did you mean --" + close.Name + "?"
		e.Next = rebuild(cmd, cmd, close)
		return e
	}
	if sibling, flag := siblingWithFlag(cmd, name); sibling != nil {
		e.Message += "; " + cliPath(sibling) + " has it"
		e.Next = rebuild(cmd, sibling, flag)
		return e
	}
	e.Next = firstExample(cmd)
	return e
}

func badValue(cmd *cobra.Command, flag *pflag.Flag, err error) *cliError {
	e := &cliError{Code: "usage", Message: err.Error(), Next: firstExample(cmd)}
	if flag != nil {
		e.Param = flag.Name
		e.Details = map[string]any{"usage": flag.Usage}
		e.Next = rebuild(cmd, cmd, flag)
	}
	return e
}

// rebuild writes the target command with the flags the caller set that the
// target also has, then add with a placeholder value, then --json. Values
// are the caller's own arguments, quoted for a POSIX shell.
func rebuild(from, target *cobra.Command, add *pflag.Flag) string {
	parts := []string{cliPath(target)}
	jsonSet := false
	from.Flags().Visit(func(f *pflag.Flag) {
		switch {
		case f.Name == "json":
			jsonSet = true
		case f.Name == add.Name || target.Flags().Lookup(f.Name) == nil:
		default:
			parts = append(parts, flagArgs(f)...)
		}
	})
	if add.Value.Type() == "bool" {
		parts = append(parts, "--"+add.Name)
	} else {
		parts = append(parts, "--"+add.Name, placeholder(add))
	}
	if jsonSet || target.Flags().Lookup("json") != nil {
		parts = append(parts, "--json")
	}
	return strings.Join(parts, " ")
}

// commandLine rebuilds the command the caller ran from its set flags.
func commandLine(cmd *cobra.Command, args []string) string {
	parts := append([]string{cliPath(cmd)}, args...)
	cmd.Flags().Visit(func(f *pflag.Flag) { parts = append(parts, flagArgs(f)...) })
	return strings.Join(parts, " ")
}

// placeholder is the flag's backquoted name from its usage, else a name for
// its type.
func placeholder(f *pflag.Flag) string {
	name, _ := pflag.UnquoteUsage(f)
	switch name {
	case "uint", "int":
		return "N"
	case "duration":
		return "DURATION"
	case "", "value", "string", "stringArray", "strings":
		return "VALUE"
	}
	return strings.ToUpper(name)
}

// flagArgs renders one set flag as arguments.
func flagArgs(f *pflag.Flag) []string {
	if sv, ok := f.Value.(pflag.SliceValue); ok {
		var out []string
		for _, v := range sv.GetSlice() {
			out = append(out, "--"+f.Name, shellWord(v))
		}
		return out
	}
	if f.Value.Type() == "bool" {
		if f.Value.String() == "true" {
			return []string{"--" + f.Name}
		}
		return []string{"--" + f.Name + "=false"}
	}
	return []string{"--" + f.Name, shellWord(f.Value.String())}
}

var plainWord = regexp.MustCompile(`^[A-Za-z0-9._/:=@%+-]+$`)

func shellWord(s string) string {
	if plainWord.MatchString(s) {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// closestFlag is a flag of cmd within two edits of name, or a flag whose
// name starts with name or name with it (--event for --events).
func closestFlag(cmd *cobra.Command, name string) *pflag.Flag {
	var best *pflag.Flag
	bestDist := 3
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if f.Hidden {
			return
		}
		d := editDistance(name, f.Name)
		if strings.HasPrefix(f.Name, name+"-") || strings.HasPrefix(name, f.Name) {
			d = min(d, 1)
		}
		if d < bestDist {
			best, bestDist = f, d
		}
	})
	return best
}

// siblingWithFlag finds another command of the dev tree with the flag.
func siblingWithFlag(cmd *cobra.Command, name string) (*cobra.Command, *pflag.Flag) {
	root := cmd
	for root.HasParent() && root.Name() != "dev" {
		root = root.Parent()
	}
	for _, c := range allCommands(root) {
		if c == cmd {
			continue
		}
		if f := c.Flags().Lookup(name); f != nil && !f.Hidden {
			return c, f
		}
	}
	return nil, nil
}

// cliPath is the command as a user types it, whatever the test root is.
func cliPath(cmd *cobra.Command) string {
	var names []string
	for c := cmd; c != nil; c = c.Parent() {
		names = append([]string{c.Name()}, names...)
		if c.Name() == "dev" {
			break
		}
	}
	return "rudder-cli " + strings.Join(names, " ")
}

func allCommands(root *cobra.Command) []*cobra.Command {
	out := []*cobra.Command{root}
	for _, c := range root.Commands() {
		out = append(out, allCommands(c)...)
	}
	return out
}

func flagNames(cmd *cobra.Command) []string {
	names := []string{}
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if !f.Hidden {
			names = append(names, "--"+f.Name)
		}
	})
	slices.Sort(names)
	return names
}

// firstExample is the first example line of cmd, or its path with --help.
func firstExample(cmd *cobra.Command) string {
	for _, line := range strings.Split(cmd.Example, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "rudder-cli ") {
			return line
		}
	}
	return cliPath(cmd) + " --help"
}

func editDistance(a, b string) int {
	prev := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur := make([]int, len(b)+1)
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(b)]
}
