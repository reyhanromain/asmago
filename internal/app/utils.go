package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// checkDependencies verifies that all required external dependencies are available.
func checkDependencies() error {
	// 1. Check if the 'aws' CLI exists in the system's PATH.
	if _, err := exec.LookPath("aws"); err != nil {
		return fmt.Errorf("dependency 'aws' CLI not found. Please ensure the AWS CLI is installed and in your PATH")
	}

	// 2. Check if the 'aws-sso-refresh' script exists.
	refreshCmdPath, err := getRefreshCommandPath()
	if err != nil {
		return err
	}

	if _, err := os.Stat(refreshCmdPath); os.IsNotExist(err) {
		return fmt.Errorf("helper script '%s' not found. Please ensure it is in the same directory as the application", refreshCmdPath)
	}

	return nil
}

// getRefreshCommandName ONLY returns the filename, not the path.
func getRefreshCommandName() string {
	if runtime.GOOS == "windows" {
		return "aws-sso-refresh.exe"
	}
	return "aws-sso-refresh"
}

// getRefreshCommandPath creates an absolute path to the helper script.
func getRefreshCommandPath() (string, error) {
	// Get the path where the asmago binary/exe is located.
	executablePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get application path: %w", err)
	}
	executableDir := filepath.Dir(executablePath)

	// Join the executable's directory with the correct helper filename.
	return filepath.Join(executableDir, getRefreshCommandName()), nil
}
