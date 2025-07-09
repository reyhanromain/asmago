#!/bin/bash

# build.sh - Cross-platform build script for asmago.
#
# USAGE:
#   ./build.sh            - Compiles for all platforms (linux, windows, darwin).
#   ./build.sh linux      - Compiles for Linux (amd64) only.
#   ./build.sh windows    - Compiles for Windows (amd64) only.
#   ./build.sh darwin     - Compiles for macOS (amd64 & arm64) only.
#   ./build.sh -i, --install - Compiles and installs the version for the current OS (macOS/Linux only).
#   ./build.sh -h, --help   - Displays this help message.

# Exit script if any command fails
set -e

# --- Configuration ---
APP_NAME="asmago"
CONFIG_SOURCE_PATH="./config"
DIST_DIR="dist"
VERSION="1.1.0"

# --- Script Logic ---

# Function to display a message with color
info() {
    echo -e "\033[34m[INFO]\033[0m $1"
}

success() {
    echo -e "\033[32m[SUCCESS]\033[0m $1"
}

# Function to display the script's usage instructions
usage() {
    echo "USAGE: $0 [COMMAND]"
    echo
    echo "Available Commands:"
    echo "  <empty>           Compiles for all platforms (linux, windows, darwin)."
    echo "  linux             Compiles for Linux (amd64) only."
    echo "  windows           Compiles for Windows (amd64) only."
    echo "  darwin            Compiles for macOS (amd64 & arm64) only."
    echo "  -i, --install     Compiles and installs the version for the current OS (macOS/Linux only)."
    echo "  -h, --help        Displays this help message."
    echo
}

# Function to build for a specific platform
build_for_platform() {
    local os=$1
    local arch=$2
    local output_dir="$DIST_DIR/${os}-${arch}"

    info "Starting build for $os/$arch..."

    # Determine file extensions for Windows
    local app_ext=""
    local helper_ext=""
    if [ "$os" == "windows" ]; then
        app_ext=".exe"
        helper_ext=".exe"
    fi

    # Create the output directory if it doesn't exist
    mkdir -p "$output_dir"

    # Perform cross-compilation with version injection
    local ldflags="-X 'main.version=$VERSION'"
    GOOS=$os GOARCH=$arch go build -ldflags="$ldflags" -o "$output_dir/$APP_NAME$app_ext" .

    # Copy the configuration directory
    #if [ -d "$CONFIG_SOURCE_PATH" ]; then
    #    cp -r "$CONFIG_SOURCE_PATH" "$output_dir/"
    #fi

    # Create launcher scripts if the OS is Windows
    if [ "$os" == "windows" ]; then
        info "Creating launcher scripts for Windows..."

        # Create start-asmago.bat
        echo "@echo off" > "$output_dir/start-asmago.bat"
        echo "REM This script opens a new terminal and runs the asmago application." >> "$output_dir/start-asmago.bat"
        echo "SET SCRIPT_DIR=%~dp0" >> "$output_dir/start-asmago.bat"
        echo "start \"asmago\" cmd /k \"%SCRIPT_DIR%${APP_NAME}.exe\"" >> "$output_dir/start-asmago.bat"
    fi

    success "Build for $os/$arch version $VERSION finished -> $output_dir"
}

# --- Main Process ---

# If the first argument is --help or -h, show usage and exit
if [[ "$1" == "--help" || "$1" == "-h" ]]; then
    usage
    exit 0
fi

info "Cleaning up old '$DIST_DIR' directory..."
rm -rf $DIST_DIR
mkdir $DIST_DIR

if [[ -z "$1" ]]; then
    info "No specific platform provided, starting build for all platforms..."
    build_for_platform "linux" "amd64"
    build_for_platform "windows" "amd64"
    build_for_platform "darwin" "amd64"
    build_for_platform "darwin" "arm64"
    echo
    success "All build processes are complete."

elif [[ "$1" == "--install" || "$1" == "-i" ]]; then
    INSTALL_DIR="/usr/local/bin"
    current_os=$(uname -s | tr '[:upper:]' '[:lower:]')

    if [[ "$current_os" == "linux" || "$current_os" == "darwin" ]]; then
        info "Starting build and installation process for $current_os..."

        current_arch=$(uname -m)
        if [[ "$current_arch" == "x86_64" ]]; then current_arch="amd64"; fi
        if [[ "$current_arch" == "aarch64" ]]; then current_arch="arm64"; fi

        build_for_platform "$current_os" "$current_arch"

        source_dir="$DIST_DIR/${current_os}-${current_arch}"
        info "Copying files from '$source_dir' to '$INSTALL_DIR'..."
        sudo mv "$source_dir/$APP_NAME" "$INSTALL_DIR/$APP_NAME"

        success "Installation complete! You can now run '$APP_NAME' from anywhere."
    else
        info "The --install option is only supported on macOS and Linux."
    fi

elif [[ "$1" == "linux" || "$1" == "windows" || "$1" == "darwin" ]]; then
    platform=$1
    info "Starting build for platform '$platform' only..."
    if [[ "$platform" == "darwin" ]]; then
        build_for_platform "darwin" "amd64"
        build_for_platform "darwin" "arm64"
    else
        build_for_platform "$platform" "amd64"
    fi
    echo
    success "Build process for '$platform' is complete."

else
    echo "Error: Command '$1' not recognized."
    echo
    usage
    exit 1
fi
