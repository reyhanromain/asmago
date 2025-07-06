package app

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ensureUserConfigExists checks if rds.json exists in the user's
// configuration directory. If not, it attempts to copy it from the
// directory where the application is running.
func ensureUserConfigExists() error {
	// 1. Determine the destination path (e.g., ~/.config/asmago/rds.json)
	targetConfigDir, err := GetConfigDir()
	if err != nil {
		return err
	}
	targetPath := filepath.Join(targetConfigDir, "rds.json")

	// 2. Check if the destination file already exists. If so, do nothing.
	if _, err := os.Stat(targetPath); err == nil {
		return nil // File already exists, job done.
	}

	// 3. Destination file does not exist. Find the source file near the executable.
	// Get the path where this binary/exe is located.
	executablePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get application path: %w", err)
	}
	executableDir := filepath.Dir(executablePath)

	// Determine the source path (e.g., /path/to/asmago/config/rds.json)
	sourcePath := filepath.Join(executableDir, "config", "rds.json")

	// 4. Check if the source file actually exists.
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		// If the source file doesn't exist, that's okay.
		// The app will fail later with a clear message when it tries to load it.
		return nil
	}
	defer sourceFile.Close()

	// 5. Perform the copy.
	fmt.Println(Cyan("'rds.json' configuration file not found. Copying template..."))

	// Ensure the destination directory exists.
	if err := os.MkdirAll(targetConfigDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Create the new destination file.
	destinationFile, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create destination config file: %w", err)
	}
	defer destinationFile.Close()

	// Copy the contents.
	if _, err := io.Copy(destinationFile, sourceFile); err != nil {
		return fmt.Errorf("failed to copy config file: %w", err)
	}

	fmt.Printf(Green("✅ Configuration successfully copied to: %s\n"), targetPath)
	return nil
}
