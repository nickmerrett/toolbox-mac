//go:build darwin

/*
 * Copyright © 2019 – 2025 Red Hat Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/containers/toolbox/pkg/podman"
	"github.com/containers/toolbox/pkg/shell"
	"github.com/containers/toolbox/pkg/skopeo"
	"github.com/containers/toolbox/pkg/term"
	"github.com/containers/toolbox/pkg/utils"
	"github.com/docker/go-units"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type promptForDownloadError struct {
	ImageSize string
}

const (
	alpha    = `abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ`
	num      = `0123456789`
	alphanum = alpha + num
)

var (
	createFlags struct {
		authFile  string
		container string
		distro    string
		image     string
		release   string
	}

	createToolboxShMounts = []struct {
		containerPath string
		source        string
	}{
		{"/etc/profile.d/toolbox.sh", "/etc/profile.d/toolbox.sh"},
		{"/etc/profile.d/toolbox.sh", "/usr/share/profile.d/toolbox.sh"},
	}
)

var createCmd = &cobra.Command{
	Use:               "create",
	Short:             "Create a new Toolbx container (macOS version)",
	RunE:              create,
	ValidArgsFunction: completionEmpty,
}

func init() {
	flags := createCmd.Flags()

	flags.StringVar(&createFlags.authFile,
		"authfile",
		"",
		"Path to a file with credentials for authenticating to the registry for private images")

	flags.StringVarP(&createFlags.container,
		"container",
		"c",
		"",
		"Assign a different name to the Toolbx container")

	flags.StringVarP(&createFlags.distro,
		"distro",
		"d",
		"",
		"Create a Toolbx container for a different operating system distribution than the host")

	flags.StringVarP(&createFlags.image,
		"image",
		"i",
		"",
		"Change the name of the base image used to create the Toolbx container")

	flags.StringVarP(&createFlags.release,
		"release",
		"r",
		"",
		"Create a Toolbx container for a different operating system release than the host")
}

func (err promptForDownloadError) Error() string {
	return fmt.Sprintf("prompt for download (size: %s)", err.ImageSize)
}

func create(cmd *cobra.Command, args []string) error {
	if utils.IsInsideContainer() {
		return errors.New("create is not supported inside a container")
	}

	container, image, release, err := utils.ResolveContainerAndImageNames(createFlags.container,
		createFlags.distro,
		createFlags.image,
		createFlags.release)
	if err != nil {
		return err
	}

	if err := createContainer(container, image, release, createFlags.authFile); err != nil {
		return err
	}

	return nil
}

func createContainer(container, image, release, authFile string) error {
	if container == "" {
		panic("container not specified")
	}

	if image == "" {
		panic("image not specified")
	}

	logrus.Debugf("Creating container %s from image %s", container, image)

	if containerExists, _ := podman.ContainerExists(container); containerExists {
		return fmt.Errorf("container %s already exists", container)
	}

	// Check if image exists locally, pull if not
	if imageExists, _ := podman.ImageExists(image); !imageExists {
		if err := pullImage(image, authFile); err != nil {
			return err
		}
	}

	// Validate it's a toolbox image
	if _, err := podman.IsToolboxImage(image); err != nil {
		fmt.Fprintf(os.Stderr, "WARNING: %s\n", err)
	}

	// Create the container with macOS-specific options
	if err := createContainerWithMacOSOptions(container, image, release); err != nil {
		return err
	}

	return nil
}

func createContainerWithMacOSOptions(container, image, release string) error {
	logrus.Debugf("Creating container %s with macOS-specific options", container)

	logLevelString := podman.LogLevel.String()

	// Basic container creation arguments for macOS
	createArgs := []string{
		"--log-level", logLevelString,
		"create",
		"--dns", "none",
		"--hostname", container,
		"--interactive",
		"--name", container,
		"--network", "slirp4netns",
		"--tty",
		"--user", "root:root",
	}

	// macOS-specific volume mounts (simplified from Linux version)
	// Note: On macOS, containers run in VMs so direct host filesystem access is limited
	homeDir := os.Getenv("HOME")
	if homeDir != "" {
		homeDirMountArg := fmt.Sprintf("%s:%s:rslave", homeDir, homeDir)
		createArgs = append(createArgs, "--volume", homeDirMountArg)
	}

	// Mount some common macOS directories if they exist
	macOSMounts := []struct {
		host      string
		container string
		options   string
	}{
		{"/Users", "/host/Users", "rslave"},
		{"/Applications", "/host/Applications", "ro"},
		{"/opt", "/host/opt", "rslave"},
		{"/usr/local", "/host/usr/local", "rslave"},
		{"/tmp", "/host/tmp", "rslave"},
	}

	for _, mount := range macOSMounts {
		if _, err := os.Stat(mount.host); err == nil {
			mountArg := fmt.Sprintf("%s:%s:%s", mount.host, mount.container, mount.options)
			createArgs = append(createArgs, "--volume", mountArg)
		}
	}

	// Security and capability options for macOS
	createArgs = append(createArgs,
		"--cap-add", "SYS_PTRACE",
		"--security-opt", "label=disable",
		"--ipc", "host",
		"--pid", "host",
	)

	// Add the image
	createArgs = append(createArgs, image)

	// Add initialization command
	createArgs = append(createArgs, "toolbox", "init-container",
		"--user", os.Getenv("USER"),
		"--uid", fmt.Sprintf("%d", os.Getuid()),
		"--gid", fmt.Sprintf("%d", os.Getgid()),
		"--home", homeDir,
		"--shell", os.Getenv("SHELL"))

	logrus.Debug("Creating container:")
	logrus.Debugf("%s %v", "podman", createArgs)

	if err := shell.Run("podman", nil, nil, nil, createArgs...); err != nil {
		return fmt.Errorf("failed to create container %s: %w", container, err)
	}

	return nil
}

func pullImage(image, authFile string) error {
	if image == "" {
		panic("image not specified")
	}

	logrus.Debugf("Pulling image %s", image)

	// Check if we need to prompt for download
	if shouldPromptForDownload(image) {
		if err := promptForDownload(image); err != nil {
			return err
		}
	}

	// Pull the image
	if err := podman.Pull(image, authFile); err != nil {
		return fmt.Errorf("failed to pull image %s: %w", image, err)
	}

	return nil
}

func shouldPromptForDownload(image string) bool {
	// For macOS, always check image size before pulling
	// This is especially important since macOS containers run in VMs
	// and may have limited bandwidth/storage
	return term.IsTerminal(os.Stdin)
}

func promptForDownload(image string) error {
	imageSize, err := getImageSize(image)
	if err != nil {
		logrus.Debugf("Failed to get image size: %v", err)
		// Continue anyway if we can't get size
		return nil
	}

	fmt.Printf("Image required to create container: %s (%s)\n", image, imageSize)
	fmt.Print("Continue? [y/N]: ")

	var response string
	fmt.Scanln(&response)
	
	response = strings.ToLower(strings.TrimSpace(response))
	if response != "y" && response != "yes" {
		return errors.New("download cancelled by user")
	}

	return nil
}

func getImageSize(image string) (string, error) {
	// Try to get image size using skopeo
	imageSizeFloat64, err := skopeo.Inspect(image, "size")
	if err != nil {
		return "", err
	}

	imageSizeInt64 := int64(imageSizeFloat64)
	imageSize := units.HumanSize(float64(imageSizeInt64))
	return imageSize, nil
}

func showSpinner(message string) *spinner.Spinner {
	if !term.IsTerminal(os.Stderr) {
		fmt.Fprintf(os.Stderr, "%s\n", message)
		return nil
	}

	s := spinner.New(spinner.CharSets[9], 500*time.Millisecond)
	s.Prefix = message + " "
	s.Start()
	return s
}

func stopSpinner(s *spinner.Spinner) {
	if s != nil {
		s.Stop()
	}
}