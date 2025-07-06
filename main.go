package main

import (
	// Replace with your actual module path, e.g., "github.com/your-user/asmago"
	"asmago/cmd"
)

// version is a variable that will be injected during compilation.
// Important: This variable must be in the 'main' package.
var version string

func main() {
	// Set the version in the cmd package before executing the command.
	cmd.SetVersion(version)
	cmd.Execute()
}
