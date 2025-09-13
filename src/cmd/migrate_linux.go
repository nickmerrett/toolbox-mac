//go:build linux

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
	"io/ioutil"
	"os"
	"strings"
	"syscall"

	"github.com/containers/toolbox/pkg/podman"
	"github.com/containers/toolbox/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func migrate(cmd *cobra.Command, args []string) error {
	logrus.Debug("Migrating to newer Podman")

	if utils.IsInsideContainer() {
		logrus.Debug("Migration not needed: running inside a container")
		return nil
	}

	if cmdName, completionCmdName := cmd.Name(), completionCmd.Name(); cmdName == completionCmdName {
		logrus.Debugf("Migration not needed: command %s doesn't need it", cmdName)
		return nil
	}

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

	toolboxRuntimeDirectory, err := utils.GetRuntimeDirectory(currentUser)
	if err != nil {
		return err
	}

	migrateLock := toolboxRuntimeDirectory + "/migrate.lock"

	migrateLockFile, err := utils.Flock(migrateLock, syscall.LOCK_EX)
	if err != nil {
		logrus.Debugf("Migrating to newer Podman: %s", err)

		var errFlock *utils.FlockError

		if errors.As(err, &errFlock) {
			if errors.Is(err, utils.ErrFlockAcquire) {
				err = utils.ErrFlockAcquire
			} else if errors.Is(err, utils.ErrFlockCreate) {
				err = utils.ErrFlockCreate
			} else {
				panicMsg := fmt.Sprintf("unexpected %T: %s", err, err)
				panic(panicMsg)
			}
		}

		return err
	}

	defer migrateLockFile.Close()

	stampBytes, err := ioutil.ReadFile(stampPath)
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

			if podmanVersion == podmanVersionOld {
				logrus.Debugf("Migration not needed: Podman version %s is unchanged", podmanVersion)
				return nil
			}

			if !podman.CheckVersion(podmanVersionOld) {
				logrus.Debugf("Migration not needed: Podman version %s is old", podmanVersion)
				return nil
			}
		}
	}

	if err = podman.SystemMigrate(""); err != nil {
		logrus.Debugf("Migrating to newer Podman: failed to migrate containers: %s", err)
		return errors.New("failed to migrate containers")
	}

	logrus.Debugf("Migration to Podman version %s was ok", podmanVersion)
	logrus.Debugf("Updating Podman version in %s", stampPath)

	podmanVersionBytes := []byte(podmanVersion + "\n")
	err = ioutil.WriteFile(stampPath, podmanVersionBytes, 0664)
	if err != nil {
		logrus.Debugf("Migrating to newer Podman: failed to update Podman version in migration stamp file %s: %s",
			stampPath,
			err)
		return errors.New("failed to update Podman version in migration stamp file")
	}

	return nil
}