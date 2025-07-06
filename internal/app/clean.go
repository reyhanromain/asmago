package app

import (
	"fmt"
	"os"

	"github.com/manifoldco/promptui"
)

// CleanAllData removes all stored data files (shortcuts, usage, etc.).
// It will ask for user confirmation before proceeding.
func CleanAllData() error {
	// Get the path to the application's data directory (e.g., ~/.local/share/asmago)
	dataDir, err := GetDataDir()
	if err != nil {
		return fmt.Errorf("failed to get data directory path: %w", err)
	}

	// Check if the directory exists before trying to delete it.
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		fmt.Println(Yellow("Data directory not found, nothing to clean."))
		return nil
	}

	fmt.Println(Yellow("This will delete all saved shortcuts and usage history."))
	prompt := promptui.Prompt{
		Label:     fmt.Sprintf("Are you sure you want to delete the directory '%s'?", dataDir),
		IsConfirm: true,
	}

	// prompt.Run() will return an error if the user selects 'n' or hits Ctrl+C.
	_, err = prompt.Run()
	if err != nil {
		fmt.Println("Cleanup cancelled.")
		return nil
	}

	fmt.Printf("Deleting directory: %s\n", dataDir)
	// Delete the entire data directory and all its contents.
	if err := os.RemoveAll(dataDir); err != nil {
		return fmt.Errorf("failed to delete data directory: %w", err)
	}

	fmt.Println(Green("✅ All data has been successfully cleaned up."))
	return nil
}
