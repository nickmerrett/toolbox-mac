#!/usr/bin/env python3

import platform
import sys
import os

print("=== Host Detection Debug ===")
print(f"Python platform.system(): {platform.system()}")
print(f"Python platform.machine(): {platform.machine()}")
print(f"Python platform.platform(): {platform.platform()}")
print(f"Python sys.platform: {sys.platform}")
print(f"OS uname: {os.uname()}")
print(f"Environment OSTYPE: {os.environ.get('OSTYPE', 'not set')}")

# Check if we're in some kind of virtualization
print("\n=== Virtualization Check ===")
print(f"/proc exists: {os.path.exists('/proc')}")
print(f"/sys exists: {os.path.exists('/sys')}")
print(f"/run exists: {os.path.exists('/run')}")
print(f"/lib64 exists: {os.path.exists('/lib64')}")

# Check for macOS specific paths
print(f"/System exists: {os.path.exists('/System')}")
print(f"/Applications exists: {os.path.exists('/Applications')}")
print(f"/usr/lib/dyld exists: {os.path.exists('/usr/lib/dyld')}")

print("\n=== Manual Meson Check ===")
print("Try running: python3 -c \"import platform; print(platform.system())\"")