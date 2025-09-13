//go:build darwin

/*
 * Copyright © 2020 – 2025 Red Hat Inc.
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
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/containers/toolbox/pkg/utils"
)

func askForConfirmation(prompt string) bool {
	fmt.Print(prompt)
	
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		answer := strings.ToLower(strings.TrimSpace(scanner.Text()))
		if answer == "y" || answer == "yes" {
			return true
		} else if answer == "n" || answer == "no" {
			return false
		} else {
			fmt.Print("Please enter y/yes or n/no: ")
		}
	}
	
	return false
}

func askForConfirmationAsync(ctx context.Context, prompt string) (<-chan bool, <-chan error) {
	confirmationChan := make(chan bool)
	errChan := make(chan error)
	
	go func() {
		defer close(confirmationChan)
		defer close(errChan)
		
		// Simple synchronous implementation for macOS
		result := askForConfirmation(prompt)
		confirmationChan <- result
	}()
	
	return confirmationChan, errChan
}

func discardInputAsync(ctx context.Context) (<-chan int, <-chan error) {
	intChan := make(chan int)
	errChan := make(chan error)
	
	go func() {
		defer close(intChan)
		defer close(errChan)
		// Simple implementation for macOS - just return 0
		intChan <- 0
	}()
	
	return intChan, errChan
}

// Error creation functions
func createErrorContainerNotFound(container string) error {
	return fmt.Errorf("container %s not found", container)
}

func createErrorDistroWithoutRelease(distro string) error {
	return fmt.Errorf("distro %s requires a release", distro)
}

func createErrorInvalidContainer(containerArg string) error {
	return fmt.Errorf("invalid container: %s", containerArg)
}

func createErrorInvalidDistro(distro string) error {
	return fmt.Errorf("invalid distro: %s", distro)
}

func createErrorInvalidImageForContainerName(container string) error {
	return fmt.Errorf("invalid image for container %s", container)
}

func createErrorInvalidImageWithoutBasename() error {
	return errors.New("invalid image without basename")
}

func createErrorInvalidRelease(hint string) error {
	return fmt.Errorf("invalid release: %s", hint)
}

func createErrorProfileDNotFound() error {
	return errors.New("profile.d not found")
}

func createErrorSudoersDNotFound() error {
	return errors.New("sudoers.d not found")
}

func getCDIFileForNvidia(targetUser *user.User) (string, error) {
	// NVIDIA CDI files are typically not used on macOS
	return "", errors.New("NVIDIA CDI not supported on macOS")
}

func getCurrentUserHomeDir() string {
	if homeDir := os.Getenv("HOME"); homeDir != "" {
		return homeDir
	}
	
	currentUser, err := user.Current()
	if err != nil {
		return ""
	}
	
	return currentUser.HomeDir
}

func getUsageForCommonCommands() string {
	return `Common commands are:
    create      Create a new Toolbx container
    enter       Enter a Toolbx container for interactive use
    list        List existing Toolbx containers and images
    run         Run a command in a Toolbx container

Go to https://containertoolbx.org for documentation.`
}

// Simplified polling function for macOS (without Linux-specific eventfd)
type pollFunc func(int32) (bool, error)

func poll(pollFn pollFunc, eventFD int32, fds ...int32) error {
	// Simple polling implementation without eventfd
	for {
		for _, fd := range fds {
			if done, err := pollFn(fd); err != nil {
				return err
			} else if done {
				return nil
			}
		}
		time.Sleep(100 * time.Millisecond) // Simple polling interval
	}
}

func resolveContainerAndImageNames(container, containerArg, distroCLI, imageCLI, releaseCLI string) (
	resolvedContainerName, resolvedImageName, resolvedRelease string,
	err error) {
	
	return utils.ResolveContainerAndImageNames(container, distroCLI, imageCLI, releaseCLI)
}

func showManual(manual string) error {
	// On macOS, try to use 'man' command
	manPath := filepath.Join("/usr/share/man/man1", manual+".1")
	if _, err := os.Stat(manPath); err == nil {
		return fmt.Errorf("manual page for %s not implemented on macOS", manual)
	}
	return fmt.Errorf("manual page %s not found", manual)
}

func watchContextForEventFD(ctx context.Context, eventFD int) {
	// macOS doesn't have eventfd, so this is a no-op
	<-ctx.Done()
}