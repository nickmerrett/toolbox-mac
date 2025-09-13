# Building Toolbox for macOS

This document describes how to build Toolbox for macOS target.

## Prerequisites

### macOS Host
- macOS 10.15+ (Catalina or later)
- Xcode Command Line Tools: `xcode-select --install`
- Homebrew: `brew install meson ninja go`
- Podman Desktop (for runtime support)

### Linux Host (Cross-compilation)
- Recent Linux distribution
- `meson`, `ninja-build`, `golang-go`
- `clang` (for cross-compilation attempts)

## Build Methods

### Method 1: Native Build on macOS

```bash
# On macOS machine
git clone <repository>
cd toolbox-mac

# Install dependencies
brew install meson ninja go go-md2man

# Build
meson setup build
ninja -C build

# Install (optional)
sudo ninja -C build install
```

### Method 2: Cross-compilation (Linux → macOS)

**Note:** Full cross-compilation requires macOS SDK and toolchain. This method configures the build system for macOS target but may fail at link time without proper macOS libraries.

```bash
# For Apple Silicon Macs (M1/M2)
meson setup build-darwin --cross-file=cross-darwin.txt
ninja -C build-darwin

# For Intel Macs
meson setup build-darwin-x64 --cross-file=cross-darwin-x64.txt  
ninja -C build-darwin-x64
```

### Method 3: Force macOS Mode (Testing on Linux)

This method forces the build system to use macOS-specific code paths while building on Linux. Useful for testing the platform abstraction layer.

```bash
meson setup build-macos-test -Dforce_macos_build=true
ninja -C build-macos-test
```

### Method 4: Go-only Build

Build just the Go components with macOS target:

```bash
cd src
GOOS=darwin GOARCH=arm64 go build -tags darwin -o ../toolbox-darwin-arm64 .
GOOS=darwin GOARCH=amd64 go build -tags darwin -o ../toolbox-darwin-amd64 .
```

## Automated Build

Use the provided script:

```bash
./build-macos.sh
```

The script will:
- Detect your platform
- Choose the appropriate build method
- Set up the build directory
- Provide next steps

## Platform Differences

The macOS port includes these changes from the Linux version:

### Dependencies
- **No libsubid**: Subordinate user ID functionality is simulated
- **No systemd**: Uses macOS-appropriate directories and services
- **No cgroups**: Resource management handled by container runtime

### Runtime Directories
- Linux: `/run/user/$UID`
- macOS: `~/Library/Caches/toolbox`

### Container Features
- Simplified volume mounts (no `/run/host` mapping)
- Uses `slirp4netns` networking
- Mounts macOS-specific paths (`/Users`, `/Applications`)

### Build Tags
Code uses Go build tags for platform separation:
- `//go:build linux` - Linux-specific code
- `//go:build darwin` - macOS-specific code  
- `//go:build unix` - Cross-platform Unix code

## Testing

1. Ensure Podman Desktop is installed and running
2. Test basic commands:
   ```bash
   ./toolbox --help
   ./toolbox list
   ./toolbox create test-container
   ./toolbox enter test-container
   ```

## Troubleshooting

### Build Issues
- **meson not found**: Install via homebrew or package manager
- **go not found**: Install Go from https://golang.org or package manager
- **libsubid errors**: Ensure you're using macOS build mode (`-Dforce_macos_build=true`)

### Runtime Issues  
- **Container creation fails**: Check Podman Desktop is running
- **Volume mount errors**: Verify paths exist and are accessible
- **Permission errors**: Check user namespace handling in Podman settings

## Contributing

When adding macOS-specific code:
1. Use appropriate build tags (`//go:build darwin`)
2. Add cross-platform interfaces when possible
3. Update both Linux and macOS code paths
4. Test on both platforms before submitting