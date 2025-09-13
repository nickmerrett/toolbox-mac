#!/bin/bash

# Build script for macOS target

echo "=== Building Toolbox for macOS ==="

# Method 1: Using cross-compilation file (requires macOS toolchain)
if command -v clang >/dev/null 2>&1; then
    echo "Method 1: Cross-compilation with clang"
    echo "For Apple Silicon (M1/M2):"
    echo "  meson setup build-darwin --cross-file=cross-darwin.txt"
    echo "  ninja -C build-darwin"
    echo ""
    echo "For Intel Macs:"
    echo "  meson setup build-darwin-x64 --cross-file=cross-darwin-x64.txt"
    echo "  ninja -C build-darwin-x64"
    echo ""
fi

# Method 2: Force macOS mode on Linux for testing
echo "Method 2: Force macOS build mode for testing (on Linux):"
echo "  meson setup build-macos-test -Dforce_macos_build=true"
echo "  ninja -C build-macos-test"
echo ""

# Method 3: Using Go build tags directly
echo "Method 3: Build Go components directly with macOS tags:"
echo "  cd src"
echo "  GOOS=darwin GOARCH=arm64 go build -tags darwin -o ../toolbox-darwin-arm64 ."
echo "  GOOS=darwin GOARCH=amd64 go build -tags darwin -o ../toolbox-darwin-amd64 ."
echo ""

# Detect if we're on Linux and suggest Method 2
if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    echo "=== Running on Linux - using Method 2 ==="
    
    # Clean up any existing build directory
    if [ -d "build-macos-test" ]; then
        echo "Removing existing build directory..."
        rm -rf build-macos-test
    fi
    
    echo "Setting up build with forced macOS mode..."
    if command -v meson >/dev/null 2>&1; then
        meson setup build-macos-test -Dforce_macos_build=true
        echo ""
        echo "Build configured! Run:"
        echo "  ninja -C build-macos-test"
    else
        echo "ERROR: meson not found. Please install meson build system."
        exit 1
    fi
elif [[ "$OSTYPE" == "darwin"* ]]; then
    echo "=== Running on macOS - using native build ==="
    
    # Clean up any existing build directory
    if [ -d "build" ]; then
        echo "Removing existing build directory..."
        rm -rf build
    fi
    
    echo "Setting up native macOS build..."
    if command -v meson >/dev/null 2>&1; then
        meson setup build
        echo ""
        echo "Build configured! Run:"
        echo "  ninja -C build"
    else
        echo "ERROR: meson not found. Please install meson build system."
        echo "  brew install meson"
        exit 1
    fi
else
    echo "Unknown OS: $OSTYPE"
    echo "Please use one of the methods above manually."
fi