package iac

import (
	"context"
	"embed"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/blang/semver"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
)

//go:embed data/credentials.json
var credentials embed.FS

//go:embed data/pulumi-resource-rudder-data-catalog
var provider embed.FS

type ErrAlreadyExists struct {
	error
}

type PulumiHelper struct {
	baseDIR       string
	workDIR       string
	versionsDIR   string
	pluginsDIR    string
	conf          *PulumiConfig
	pulumiCommand auto.PulumiCommand
}

func NewPulumiHelper(ctx context.Context, conf *PulumiConfig) (*PulumiHelper, error) {
	h := &PulumiHelper{
		baseDIR:     conf.PulumiHome,
		conf:        conf,
		versionsDIR: filepath.Join(conf.PulumiHome, "versions"),
		workDIR:     filepath.Join(conf.PulumiHome, "work"),
		pluginsDIR:  filepath.Join(conf.PulumiHome, "plugins"),
	}

	return h, nil
}

func (h *PulumiHelper) installPulumi(ctx context.Context, version string) (auto.PulumiCommand, error) {
	rootPath := filepath.Join(h.versionsDIR, version)

	// If I can successfully list the rootPath, return it as a
	// pulumi command else install it
	if _, err := os.Stat(rootPath); err == nil {
		return auto.NewPulumiCommand(&auto.PulumiCommandOptions{
			Root:    rootPath,
			Version: semver.MustParse(version),
		})
	}

	// Install the pulumi command from the options
	return auto.InstallPulumiCommand(ctx, &auto.PulumiCommandOptions{
		Version: semver.MustParse(version),
		Root:    rootPath,
	})
}

func (h *PulumiHelper) RemovePulumi(version string) error {
	return os.RemoveAll(h.versionsDIR)
}

func (h *PulumiHelper) InstallPlugin(url, name, version string, force bool) error {
	return nil
}

func (h *PulumiHelper) WorkDir() string {
	return h.workDIR
}

func (h *PulumiHelper) HomeDir() string {
	return h.baseDIR
}

func (h *PulumiHelper) Conf() *PulumiConfig {
	return h.conf
}

func (h *PulumiHelper) PulumiCommand() auto.PulumiCommand {
	return h.pulumiCommand
}

func (h *PulumiHelper) Setup(ctx context.Context) error {
	fmt.Println("Setting up the dependency")

	// 1. setup all the dirs
	for _, dir := range []string{h.baseDIR, h.versionsDIR, h.workDIR, h.pluginsDIR} {
		if _, err := os.Stat(dir); err == nil {
			continue // dir exists, continue
		}

		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("mkdir %s: %w", dir, err)
		}
	}

	// 2. locally install the pulumi command within appropriate directory
	pulumiCommand, err := h.installPulumi(ctx, h.conf.ToolVersion)
	if err != nil {
		return fmt.Errorf("installing pulumi: %s to be used: %w", h.conf.ToolVersion, err)
	}
	h.pulumiCommand = pulumiCommand

	// 3. setup the credentials file json into the main dir
	if err := copyEmbeds(credentials, "data/credentials.json", filepath.Join(h.baseDIR, "credentials.json")); err != nil {
		return fmt.Errorf("copying credentials file: %w", err)
	}

	if err := copyEmbeds(
		provider,
		"data/pulumi-resource-rudder-data-catalog",
		filepath.Join(
			h.pluginsDIR,
			"resource-rudder-data-catalog-v0.0.0",
			"pulumi-resource-rudder-data-catalog",
		),
	); err != nil {
		return fmt.Errorf("copying catalog provider binary: %w", err)
	}

	return nil
}

func copyEmbeds(source embed.FS, sourceName, dest string) error {
	if _, err := os.Stat(dest); err == nil {
		return nil
	}
	fmt.Printf("Copying %s to %s\n", sourceName, dest)

	sourceF, err := source.Open(sourceName)
	if err != nil {
		return fmt.Errorf("opening file to read: %s, err: %w", sourceName, err)
	}

	defer sourceF.Close()

	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return fmt.Errorf("creating dir: %s, err: %w", filepath.Dir(dest), err)
	}

	destF, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		return fmt.Errorf("opening file to write: %s, err: %w", dest, err)
	}

	defer destF.Close()

	if _, err := io.Copy(destF, sourceF); err != nil {
		return fmt.Errorf("copying contents from source: %s to dest: %s: %w", sourceName, dest, err)
	}

	return err
}
