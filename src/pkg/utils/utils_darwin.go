//go:build darwin

/*
 * Copyright © 2022 – 2024 Red Hat Inc.
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

package utils

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/sirupsen/logrus"
)

// ValidateSubIDRanges provides a macOS implementation for subordinate ID validation
// Since macOS doesn't have /etc/subuid and /etc/subgid, we simulate this functionality
func ValidateSubIDRanges(user *user.User) (bool, error) {
	if IsInsideContainer() {
		panic("cannot validate subordinate IDs inside container")
	}

	if user == nil {
		panic("cannot validate subordinate IDs when user is nil")
	}

	if user.Username == "ALL" {
		return false, errors.New("username ALL not supported")
	}

	uid, err := strconv.Atoi(user.Uid)
	if err != nil {
		return false, fmt.Errorf("invalid user ID: %w", err)
	}

	// On macOS, regular users start at UID 501
	// We'll allow containers for non-root users with reasonable UIDs
	if uid == 0 {
		return false, errors.New("root user not supported for containers on macOS")
	}

	if uid < 501 {
		return false, errors.New("system user not supported for containers")
	}

	// For macOS, we'll simulate subordinate ID ranges
	// This is a simplified approach since Podman on macOS handles
	// user namespace mapping differently
	logrus.Debugf("macOS: Simulating subordinate ID validation for user %s (UID: %d)", user.Username, uid)
	return true, nil
}

// GetCgroupsVersion returns a simulated cgroups version for macOS
// Since macOS doesn't have cgroups, we return version 1 as a safe default
func GetCgroupsVersion() (int, error) {
	// macOS doesn't have cgroups, but Podman on macOS handles resource
	// management internally. Return version 1 as it's more compatible.
	logrus.Debug("macOS: Simulating cgroups v1 (no native cgroups support)")
	return 1, nil
}

// GetRuntimeDirectory returns the macOS equivalent of Linux /run/user/<uid>
func GetRuntimeDirectory(targetUser *user.User) (string, error) {
	if runtimeDirectories == nil {
		runtimeDirectories = make(map[string]string)
	}

	if toolboxRuntimeDirectory, ok := runtimeDirectories[targetUser.Uid]; ok {
		return toolboxRuntimeDirectory, nil
	}

	gid, err := strconv.Atoi(targetUser.Gid)
	if err != nil {
		return "", fmt.Errorf("failed to convert group ID to integer: %w", err)
	}

	uid, err := strconv.Atoi(targetUser.Uid)
	if err != nil {
		return "", fmt.Errorf("failed to convert user ID to integer: %w", err)
	}

	var runtimeDirectory string

	if uid == 0 {
		// Root uses /var/run on macOS (similar to Linux /run)
		runtimeDirectory = "/var/run"
	} else {
		// For regular users, use their cache directory
		// This is more appropriate on macOS than /tmp
		homeDir := targetUser.HomeDir
		if homeDir == "" {
			return "", errors.New("user home directory not found")
		}
		runtimeDirectory = filepath.Join(homeDir, "Library", "Caches")
	}

	toolboxRuntimeDirectory := filepath.Join(runtimeDirectory, "toolbox")
	logrus.Debugf("Creating runtime directory %s", toolboxRuntimeDirectory)

	if err := os.MkdirAll(toolboxRuntimeDirectory, 0700); err != nil {
		return "", fmt.Errorf("failed to create runtime directory %s: %w", toolboxRuntimeDirectory, err)
	}

	if err := os.Chown(toolboxRuntimeDirectory, uid, gid); err != nil {
		wrappedErr := fmt.Errorf("failed to change ownership of the runtime directory %s: %w",
			toolboxRuntimeDirectory,
			err)
		return "", wrappedErr
	}

	runtimeDirectories[targetUser.Uid] = toolboxRuntimeDirectory
	return toolboxRuntimeDirectory, nil
}

// EnsureXdgRuntimeDirIsSet sets XDG_RUNTIME_DIR to a macOS-appropriate location
func EnsureXdgRuntimeDirIsSet(uid int) {
	if _, ok := os.LookupEnv("XDG_RUNTIME_DIR"); !ok {
		logrus.Debug("XDG_RUNTIME_DIR is unset")

		// On macOS, create a user-specific runtime directory in temp
		xdgRuntimeDir := filepath.Join(os.TempDir(), fmt.Sprintf("toolbox-runtime-%d", uid))
		
		// Create the directory if it doesn't exist
		if err := os.MkdirAll(xdgRuntimeDir, 0700); err != nil {
			logrus.Debugf("Failed to create XDG_RUNTIME_DIR: %v", err)
			// Fall back to temp directory
			xdgRuntimeDir = os.TempDir()
		}

		os.Setenv("XDG_RUNTIME_DIR", xdgRuntimeDir)
		logrus.Debugf("XDG_RUNTIME_DIR set to %s", xdgRuntimeDir)
	}
}

// IsInsideContainer checks if we're running inside a container on macOS
func IsInsideContainer() bool {
	// Check for standard container indicators that work on macOS
	if _, exists := os.LookupEnv("container"); exists {
		return true
	}

	// Check for Docker-specific files
	if PathExists("/.dockerenv") {
		return true
	}

	// Check for Podman/container environment files
	if PathExists("/run/.containerenv") {
		return true
	}

	// On macOS, containers might not have the same filesystem layout
	// Also check for common container filesystem indicators
	if PathExists("/proc/1/cgroup") {
		// Try to read cgroup info to detect containerization
		// This is a heuristic approach for macOS containers
		return true
	}

	return false
}

// IsInsideToolboxContainer checks for toolbox-specific container markers on macOS
func IsInsideToolboxContainer() bool {
	// Check for our custom marker file
	if PathExists("/run/.toolboxenv") {
		return true
	}

	// On macOS, also check in temp directory
	markerPath := filepath.Join(os.TempDir(), ".toolboxenv")
	if PathExists(markerPath) {
		return true
	}

	return false
}

// Flock provides file locking functionality on macOS
// The syscall.Flock function works on both Linux and macOS
func Flock(path string, how int) (*os.File, error) {
	file, err := os.Create(path)
	if err != nil {
		errs := []error{ErrFlockCreate, err}
		return nil, &FlockError{Path: path, Errs: errs}
	}

	fd := file.Fd()
	fdInt := int(fd)
	if err := syscall.Flock(fdInt, how); err != nil {
		errs := []error{ErrFlockAcquire, err}
		return nil, &FlockError{Path: path, Errs: errs, errSuffix: "on"}
	}

	return file, nil
}

// CallFlatpakSessionHelper returns an error on macOS since Flatpak is not available
func CallFlatpakSessionHelper() (string, error) {
	return "", errors.New("Flatpak is not available on macOS")
}