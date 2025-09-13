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
	"os"
	"strings"

	"github.com/containers/toolbox/pkg/podman"
	"github.com/containers/toolbox/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func migrate(cmd *cobra.Command, args []string) error {
	logrus.Debug("Migrating to newer Podman (macOS)")
	
	if utils.IsInsideContainer() {
		logrus.Debug("Migration not needed: running inside a container")
		return nil
	}
	
	if cmdName, completionCmdName := cmd.Name(), completionCmd.Name(); cmdName == completionCmdName {
		logrus.Debugf("Migration not needed: command %s doesn't need it", cmdName)
		return nil
	}
	
	// On macOS, Podman migration is typically less critical since containers
	// run in a VM with different storage backends. We'll try a simpler approach.
	configDir, err := os.UserConfigDir()
	if err != nil {
		logrus.Debugf("Migrating to newer Podman: failed to get the user config directory: %s", err)
		return errors.New("failed to get the user config directory")
	}
	
	toolboxConfigDir := configDir + "/toolbox"
	stampPath := toolboxConfigDir + "/podman-system-migrate"
	logrus.Debugf("Toolbx config directory is %s", toolboxConfigDir)
	
	podmanVersion, err := podman.GetVersion()
	if err != nil {
		logrus.Debugf("Migrating to newer Podman: failed to get the Podman version: %s", err)
		return errors.New("failed to get the Podman version")
	}
	logrus.Debugf("Current Podman version is %s", podmanVersion)
	
	err = os.MkdirAll(toolboxConfigDir, 0775)
	if err != nil {
		logrus.Debugf("Migrating to newer Podman: failed to create configuration directory %s: %s",
			toolboxConfigDir,
			err)
		return errors.New("failed to create configuration directory")
	}
	
	// On macOS, we'll skip the complex lock file mechanism and system migration
	// that's needed for Linux, since Podman on macOS typically manages its own
	// VM-based storage more reliably.
	
	// Check if we have an existing stamp file
	stampBytes, err := os.ReadFile(stampPath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			logrus.Debugf("Migrating to newer Podman: failed to read migration stamp file %s: %s",
				stampPath,
				err)
			return errors.New("failed to read migration stamp file")
		}
	} else {
		stampString := string(stampBytes)
		podmanVersionOld := strings.TrimSpace(stampString)
		if podmanVersionOld != "" {
			logrus.Debugf("Old Podman version is %s", podmanVersionOld)
			
			// If versions are the same, no migration needed
			if podmanVersionOld == podmanVersion {
				logrus.Debug("Migration not needed: Podman version unchanged")
				return nil
			}
		}
	}
	
	// On macOS, try system migrate but don't fail if it doesn't work
	// since Podman Desktop and other macOS Podman installations may handle
	// this differently
	logrus.Debug("Attempting Podman system migrate (may skip on macOS)")
	if err = podman.SystemMigrate(""); err != nil {
		logrus.Debugf("Podman system migrate failed (expected on some macOS setups): %s", err)
		// Don't return error - just log it and continue
		logrus.Debug("Continuing without system migration (common on macOS)")
	} else {
		logrus.Debug("Podman system migrate succeeded")
	}
	
	// Update stamp file regardless of migration result
	logrus.Debugf("Migration to Podman version %s completed", podmanVersion)
	logrus.Debugf("Updating Podman version in %s", stampPath)
	podmanVersionBytes := []byte(podmanVersion + "\n")
	err = os.WriteFile(stampPath, podmanVersionBytes, 0664)
	if err != nil {
		logrus.Debugf("Migrating to newer Podman: failed to update Podman version in migration stamp file %s: %s",
			stampPath,
			err)
		return errors.New("failed to update Podman version in migration stamp file")
	}
	
	return nil
}