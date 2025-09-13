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
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"

	"github.com/containers/toolbox/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	initContainerFlags struct {
		gid         int
		home        string
		homeLink    bool
		mediaLink   bool
		mntLink     bool
		monitorHost bool
		shell       string
		uid         int
		user        string
	}

	// macOS-specific container initialization mounts
	// These are simplified compared to Linux since macOS containers
	// run in VMs and have different filesystem layouts
	initContainerMounts = []struct {
		containerPath string
		source        string
		flags         string
	}{
		// Basic directory mappings that might exist in macOS containers
		{"/tmp", "/host/tmp", ""},
		// Note: Many Linux-specific paths like /run/systemd/* don't exist on macOS
	}
)

var initContainerCmd = &cobra.Command{
	Use:    "init-container",
	Short:  "Initialize a running container (macOS version)",
	Hidden: true,
	RunE:   initContainer,
}

func init() {
	flags := initContainerCmd.Flags()

	flags.IntVar(&initContainerFlags.gid,
		"gid",
		0,
		"GID to configure inside the Toolbx container")

	flags.StringVar(&initContainerFlags.home,
		"home",
		"",
		"Path to the user's home directory inside the Toolbx container")

	flags.BoolVar(&initContainerFlags.homeLink,
		"home-link",
		false,
		"Make /home a symbolic link to /var/home")

	flags.BoolVar(&initContainerFlags.mediaLink,
		"media-link",
		false,
		"Make /media a symbolic link to /run/media")

	flags.BoolVar(&initContainerFlags.mntLink,
		"mnt-link",
		false,
		"Make /mnt a symbolic link to /var/mnt")

	flags.BoolVar(&initContainerFlags.monitorHost,
		"monitor-host",
		false,
		"Monitor host configuration changes")

	flags.StringVar(&initContainerFlags.shell,
		"shell",
		"",
		"Path to the user's default shell inside the Toolbx container")

	flags.IntVar(&initContainerFlags.uid,
		"uid",
		0,
		"UID to configure inside the Toolbx container")

	flags.StringVar(&initContainerFlags.user,
		"user",
		"",
		"Username to configure inside the Toolbx container")

	initContainerCmd.Flags().MarkHidden("gid")
	initContainerCmd.Flags().MarkHidden("home")
	initContainerCmd.Flags().MarkHidden("home-link")
	initContainerCmd.Flags().MarkHidden("media-link")
	initContainerCmd.Flags().MarkHidden("mnt-link")
	initContainerCmd.Flags().MarkHidden("monitor-host")
	initContainerCmd.Flags().MarkHidden("shell")
	initContainerCmd.Flags().MarkHidden("uid")
	initContainerCmd.Flags().MarkHidden("user")
}

func initContainer(cmd *cobra.Command, args []string) error {
	logrus.Debug("Starting macOS container initialization")

	if !utils.IsInsideContainer() {
		return errors.New("init-container is only intended to be run inside a container")
	}

	// Create toolbox environment marker for macOS
	if err := createToolboxEnvironmentFile(); err != nil {
		return err
	}

	// Set up basic user configuration
	if err := setupUser(); err != nil {
		return err
	}

	// Create necessary directory structure
	if err := setupDirectories(); err != nil {
		return err
	}

	// Configure hostname if needed
	if err := setupHostname(); err != nil {
		return err
	}

	logrus.Debug("macOS container initialization completed")
	return nil
}

func createToolboxEnvironmentFile() error {
	logrus.Debug("Creating toolbox environment marker")

	// Try both locations - container might have /run or might not
	markerPaths := []string{
		"/run/.toolboxenv",
		"/tmp/.toolboxenv",
	}

	for _, markerPath := range markerPaths {
		toolboxEnvFile, err := os.Create(markerPath)
		if err != nil {
			logrus.Debugf("Failed to create %s: %v", markerPath, err)
			continue
		}

		toolboxEnvFile.Close()
		logrus.Debugf("Created toolbox environment marker at %s", markerPath)
		return nil
	}

	return errors.New("failed to create toolbox environment marker")
}

func setupUser() error {
	if initContainerFlags.user == "" {
		return nil
	}

	logrus.Debugf("Setting up user: %s (UID: %d, GID: %d)",
		initContainerFlags.user,
		initContainerFlags.uid,
		initContainerFlags.gid)

	// On macOS containers, user setup is typically handled by the container runtime
	// We just need to ensure the user exists and has proper permissions

	if _, err := user.Lookup(initContainerFlags.user); err != nil {
		logrus.Debugf("User %s not found, this may be expected in macOS containers", initContainerFlags.user)
	}

	return nil
}

func setupDirectories() error {
	logrus.Debug("Setting up directory structure")

	// Create basic directories that toolbox expects
	dirs := []string{
		"/var/log",
		"/var/tmp",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			logrus.Debugf("Failed to create directory %s: %v", dir, err)
		}
	}

	// Handle symbolic links if requested
	if initContainerFlags.homeLink {
		if err := createSymlinkIfNeeded("/home", "/var/home"); err != nil {
			logrus.Debugf("Failed to create home symlink: %v", err)
		}
	}

	if initContainerFlags.mntLink {
		if err := createSymlinkIfNeeded("/mnt", "/var/mnt"); err != nil {
			logrus.Debugf("Failed to create mnt symlink: %v", err)
		}
	}

	// Media link is less relevant on macOS but handle it anyway
	if initContainerFlags.mediaLink {
		if err := createSymlinkIfNeeded("/media", "/var/media"); err != nil {
			logrus.Debugf("Failed to create media symlink: %v", err)
		}
	}

	return nil
}

func setupHostname() error {
	// On macOS containers, hostname is typically managed by the container runtime
	// Just log that we're skipping this
	logrus.Debug("Hostname configuration handled by container runtime on macOS")
	return nil
}

func createSymlinkIfNeeded(linkPath, targetPath string) error {
	// Check if link already exists and points to the right place
	if target, err := os.Readlink(linkPath); err == nil {
		if target == targetPath {
			return nil // Already correct
		}
		// Remove incorrect symlink
		os.Remove(linkPath)
	}

	// Check if target exists as a directory, if not create it
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		if err := os.MkdirAll(targetPath, 0755); err != nil {
			return fmt.Errorf("failed to create target directory %s: %w", targetPath, err)
		}
	}

	// Create the symlink
	if err := os.Symlink(targetPath, linkPath); err != nil {
		return fmt.Errorf("failed to create symlink %s -> %s: %w", linkPath, targetPath, err)
	}

	logrus.Debugf("Created symlink %s -> %s", linkPath, targetPath)
	return nil
}