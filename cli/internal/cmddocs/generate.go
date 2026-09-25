// Package cmddocs generates the public rudder-cli command reference bundle.
package cmddocs

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
	"github.com/spf13/pflag"
)

const (
	commandsDir = "commands"
	manDir      = "man"
)

// Generate writes Markdown, YAML, and section 1 man pages below outputDir.
func Generate(root *cobra.Command, outputDir string) error {
	prepareTree(root)

	var (
		commandOutputDir = filepath.Join(outputDir, commandsDir)
		manOutputDir     = filepath.Join(outputDir, manDir)
	)
	if err := resetOutputDirs(commandOutputDir, manOutputDir); err != nil {
		return err
	}

	commandPaths := markdownCommandPaths(root)
	prepender := func(filename string) string {
		return fmt.Sprintf("---\ncommand: %s\n---\n\n", commandPaths[filepath.Base(filename)])
	}
	linkHandler := func(filename string) string {
		return strings.ReplaceAll(filepath.Base(filename), "_", "-")
	}

	if err := doc.GenMarkdownTreeCustom(root, commandOutputDir, prepender, linkHandler); err != nil {
		return fmt.Errorf("generating Markdown command docs: %w", err)
	}
	if err := writeHelpMarkdown(root, commandOutputDir, linkHandler); err != nil {
		return err
	}
	if err := doc.GenYamlTree(root, commandOutputDir); err != nil {
		return fmt.Errorf("generating YAML command docs: %w", err)
	}
	if err := writeHelpYAML(root, commandOutputDir, linkHandler); err != nil {
		return err
	}
	if err := useHyphenatedCommandFilenames(commandOutputDir); err != nil {
		return err
	}

	epoch := time.Unix(0, 0).UTC()
	header := &doc.GenManHeader{
		Title:   "RUDDER-CLI",
		Section: "1",
		Date:    &epoch,
		Source:  "rudder-cli",
		Manual:  "Rudder CLI Manual",
	}
	if err := doc.GenManTree(root, header, manOutputDir); err != nil {
		return fmt.Errorf("generating man pages: %w", err)
	}
	if err := writeHelpMan(root, header, manOutputDir); err != nil {
		return err
	}

	return nil
}

func writeHelpMarkdown(root *cobra.Command, dir string, linkHandler func(string) string) error {
	helpCommand := initHelpCommand(root)
	filename := filepath.Join(dir, strings.ReplaceAll(helpCommand.CommandPath(), " ", "_")+".md")

	var buf bytes.Buffer
	if _, err := buf.WriteString(fmt.Sprintf("---\ncommand: %s\n---\n\n", helpCommand.CommandPath())); err != nil {
		return fmt.Errorf("writing help Markdown front matter: %w", err)
	}
	if err := doc.GenMarkdownCustom(helpCommand, &buf, linkHandler); err != nil {
		return fmt.Errorf("generating Markdown help docs: %w", err)
	}
	if err := os.WriteFile(filename, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("writing Markdown help docs: %w", err)
	}
	return nil
}

func writeHelpYAML(root *cobra.Command, dir string, linkHandler func(string) string) error {
	helpCommand := initHelpCommand(root)
	filename := filepath.Join(dir, strings.ReplaceAll(helpCommand.CommandPath(), " ", "_")+".yaml")

	var buf bytes.Buffer
	if err := doc.GenYamlCustom(helpCommand, &buf, linkHandler); err != nil {
		return fmt.Errorf("generating YAML help docs: %w", err)
	}
	if err := os.WriteFile(filename, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("writing YAML help docs: %w", err)
	}
	return nil
}

func writeHelpMan(root *cobra.Command, header *doc.GenManHeader, dir string) error {
	helpCommand := initHelpCommand(root)
	filename := filepath.Join(dir, strings.ReplaceAll(helpCommand.CommandPath(), " ", "-")+"."+header.Section)

	var buf bytes.Buffer
	if err := doc.GenMan(helpCommand, header, &buf); err != nil {
		return fmt.Errorf("generating man help docs: %w", err)
	}
	if err := os.WriteFile(filename, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("writing man help docs: %w", err)
	}
	return nil
}

func initHelpCommand(root *cobra.Command) *cobra.Command {
	root.InitDefaultHelpCmd()
	helpCommand, _, err := root.Find([]string{"help"})
	if err != nil || helpCommand == root {
		panic("help command not registered")
	}
	return helpCommand
}

func prepareTree(command *cobra.Command) {
	command.DisableAutoGenTag = true
	if command.HasPersistentFlags() {
		if configFlag := command.PersistentFlags().Lookup("config"); configFlag != nil {
			stableFlags := pflag.NewFlagSet("command-docs", pflag.ContinueOnError)
			stableFlags.String("config", "~/.rudder/config.json", "config file (default is '~/.rudder/config.json')")
			stableConfigFlag := stableFlags.Lookup("config")
			configFlag.Value = stableConfigFlag.Value
			configFlag.DefValue = stableConfigFlag.DefValue
			configFlag.Usage = stableConfigFlag.Usage
		}
	}
	for _, child := range command.Commands() {
		if child.Hidden || strings.TrimSpace(child.Deprecated) != "" {
			command.RemoveCommand(child)
			continue
		}
		prepareTree(child)
	}
}

func resetOutputDirs(dirs ...string) error {
	for _, dir := range dirs {
		if err := os.RemoveAll(dir); err != nil {
			return fmt.Errorf("clearing generated docs directory %q: %w", dir, err)
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("creating generated docs directory %q: %w", dir, err)
		}
	}
	return nil
}

func markdownCommandPaths(root *cobra.Command) map[string]string {
	paths := make(map[string]string)
	var collect func(*cobra.Command)
	collect = func(command *cobra.Command) {
		filename := strings.ReplaceAll(command.CommandPath(), " ", "_") + ".md"
		paths[filename] = command.CommandPath()
		for _, child := range command.Commands() {
			collect(child)
		}
	}
	collect(root)
	return paths
}

func useHyphenatedCommandFilenames(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("reading generated command docs directory: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.Contains(entry.Name(), "_") {
			continue
		}

		oldPath := filepath.Join(dir, entry.Name())
		newPath := filepath.Join(dir, strings.ReplaceAll(entry.Name(), "_", "-"))
		if err := os.Rename(oldPath, newPath); err != nil {
			return fmt.Errorf("renaming generated command doc %q: %w", entry.Name(), err)
		}
	}
	return nil
}
