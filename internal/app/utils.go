package app

import (
	"fmt"
	"os/exec"
)

// checkDependencies verifies that all required external dependencies are available.
func checkDependencies() error {
	// 1. Check if the 'aws' CLI exists in the system's PATH.
	if _, err := exec.LookPath("aws"); err != nil {
		return fmt.Errorf("dependency 'aws' CLI not found. Please ensure the AWS CLI is installed and in your PATH")
	}

	return nil
}
