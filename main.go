package main

import (
	"asmago/internal/app"
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
)

// version is a variable that will be injected during compilation.
// Important: This variable must be in the 'main' package.
var version string

// dryRun holds the state of the --dry-run flag.
var dryRun bool

// rootCmd is the base command when the application is called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "asmago",
	Short: "asmago is a CLI tool to simplify SSM and RDS connections.",
	Long: `asmago (AWS SSM Manager Golang) provides an interactive workflow
to connect to EC2 instances via SSM or to port-forward to an RDS database.
It also supports shortcuts to speed up repetitive tasks.`,
	// Cobra will automatically add a --version flag if this field is set.
	Version: "dev", // Default value if no version is injected
	Run: func(cmd *cobra.Command, args []string) {
		runInteractive()
	},
}

func main() {
	SetVersion(version)
	Execute()
}

// SetVersion is an exported function to set the version from the main package.
func SetVersion(v string) {
	if v != "" {
		rootCmd.Version = v
	}
}

// Execute is called by main.go to start the application.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// init runs before main and is used to register subcommands and flags.
func init() {
	rootCmd.PersistentFlags().BoolVarP(&dryRun, "dry-run", "d", false, "Display the final command without executing it")

	rootCmd.AddCommand(interactiveCmd)
	rootCmd.AddCommand(listShortcutsCmd)
	rootCmd.AddCommand(cleanCmd)
}

// interactiveCmd defines the 'interactive' subcommand.
var interactiveCmd = &cobra.Command{
	Use:     "interactive",
	Aliases: []string{"i"},
	Short:   "Start an interactive session for SSM/RDS connections.",
	Run: func(cmd *cobra.Command, args []string) {
		runInteractive()
	},
}

// listShortcutsCmd defines the 'shortcuts' subcommand.
var listShortcutsCmd = &cobra.Command{
	Use:     "shortcuts",
	Aliases: []string{"sc"},
	Short:   "Display a list of the top 5 saved shortcuts.",
	Run: func(cmd *cobra.Command, args []string) {
		application, err := app.NewApp(false) // dryRun is not relevant for listing
		if err != nil {
			log.Fatalf("❌ Failed to initialize application: %v", err)
		}
		if err := application.ListShortcuts(); err != nil {
			log.Fatalf("❌ Error displaying shortcuts: %v", err)
		}
	},
}

// cleanCmd defines the 'clean' subcommand.
var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Delete all shortcut data and usage history.",
	Long: `This command removes the entire application data directory
(~/.local/share/asmago), including all saved shortcuts and usage history.
This action cannot be undone.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := app.CleanAllData(); err != nil {
			log.Fatalf("❌ Error cleaning up data: %v", err)
		}
	},
}

// runInteractive is a helper to run the main interactive flow.
func runInteractive() {
	application, err := app.NewApp(dryRun)
	if err != nil {
		log.Fatalf("❌ Failed to initialize application: %v", err)
	}
	if err := application.Run(); err != nil {
		log.Fatalf("❌ Error: %v", err)
	}
	if !dryRun {
		fmt.Println(app.Green("\nProcess finished."))
	}
}
