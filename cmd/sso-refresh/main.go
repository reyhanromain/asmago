package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// Define functions for text coloring
var (
	green  = color.New(color.FgGreen).SprintFunc()
	red    = color.New(color.FgRed).SprintFunc()
	yellow = color.New(color.FgYellow).SprintFunc()
	cyan   = color.New(color.FgCyan).SprintFunc()
)

// Main function for CLI setup and execution
func main() {
	var force bool

	var rootCmd = &cobra.Command{
		Use:   "aws-sso-refresh [profile-name]",
		Short: "An advanced script to check and update AWS SSO tokens.",
		Long:  "Checks the validity of the SSO token for a given AWS profile and guides the user to log in again if necessary.",
		Args:  cobra.ExactArgs(1), // Requires exactly 1 argument (the profile name)
		Run: func(cmd *cobra.Command, args []string) {
			profile := args[0]
			runLogic(profile, force)
		},
	}

	rootCmd.Flags().BoolVarP(&force, "force", "f", false, "Force the login process even if the token is still active.")
	rootCmd.CompletionOptions.DisableDefaultCmd = true // Hide the 'completion' command

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, red("❌ ", err))
		os.Exit(1)
	}
}

// Function containing the main application logic
func runLogic(profile string, force bool) {
	fmt.Println(cyan("ℹ️  Checking SSO status for profile:", profile, "..."))

	// Replicate the Bash logic to get specific configurations
	ssoSession := getConfigValue(profile, "sso_session")
	ssoStartURL := getConfigValue(profile, "sso_start_url")

	if ssoSession == "" && ssoStartURL == "" {
		fmt.Println(green("✅ Info: Profile '" + profile + "' is not an SSO profile. No action needed."))
		return
	}

	if !force {
		fmt.Println(cyan("   SSO profile detected, checking token validity..."))
		if isTokenValid(profile) {
			fmt.Println(green("✅ SSO token for profile '" + profile + "' is still active."))
			return
		}

		fmt.Println(yellow("⚠️ SSO token for profile '" + profile + "' is invalid/expired."))
		if !confirmWithFZF(profile) {
			fmt.Println("Token update cancelled by user.")
			return
		}
	} else {
		fmt.Println(green("✅ SSO profile detected (--force mode)."))
	}

	fmt.Println(cyan("⏳ Attempting to update token for profile '" + profile + "'..."))
	fmt.Println(yellow("   Please complete the login process in your browser..."))

	// Run 'aws sso login' as the browser login flow is a feature of the AWS CLI
	var loginCmd *exec.Cmd
	if ssoSession != "" {
		loginCmd = exec.Command("aws", "sso", "login", "--sso-session", ssoSession)
	} else {
		loginCmd = exec.Command("aws", "sso", "login", "--profile", profile)
	}

	var stderr bytes.Buffer
	loginCmd.Stdin = os.Stdin
	loginCmd.Stdout = os.Stdout
	loginCmd.Stderr = &stderr

	if err := loginCmd.Run(); err != nil {
		fmt.Fprintln(os.Stderr, red("\n❌ Failed to update token for profile '"+profile+"'."))
		fmt.Fprintln(os.Stderr, "Error Details:", stderr.String())
		os.Exit(1)
	}

	fmt.Println(green("\n🚀 SSO token for profile '" + profile + "' has been successfully updated!"))

	// Display active session info after successful login
	displayCallerIdentity(profile)
}

// Helper to run 'aws configure get'
func getConfigValue(profile, key string) string {
	cmd := exec.Command("aws", "configure", "get", key, "--profile", profile)
	output, err := cmd.Output()
	if err != nil {
		return "" // Key or profile does not exist
	}
	return strings.TrimSpace(string(output))
}

// Helper to check token validity with the AWS SDK
func isTokenValid(profile string) bool {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithSharedConfigProfile(profile))
	if err != nil {
		return false
	}
	client := sts.NewFromConfig(cfg)
	_, err = client.GetCallerIdentity(context.TODO(), &sts.GetCallerIdentityInput{})
	return err == nil
}

// Helper to display a confirmation prompt with fzf
func confirmWithFZF(profile string) bool {
	if _, err := exec.LookPath("fzf"); err != nil {
		fmt.Fprintln(os.Stderr, red("❌ 'fzf' utility not found to display confirmation prompt."))
		return false
	}

	prompt := fmt.Sprintf("⚠️ SSO token for profile '%s' is invalid/expired. Update token now? > ", profile)

	c1 := exec.Command("printf", "Yes\nNo")
	c2 := exec.Command("fzf", "--prompt", prompt, "--color=prompt:yellow:bold")

	c2.Stdin, _ = c1.StdoutPipe()
	c2.Stderr = os.Stderr

	var fzfOutput bytes.Buffer
	c2.Stdout = &fzfOutput

	_ = c1.Start()
	_ = c2.Run() // We ignore fzf errors if the user cancels (Esc)

	return strings.TrimSpace(fzfOutput.String()) == "Yes"
}

// Helper to display session info after login
func displayCallerIdentity(profile string) {
	cmd := exec.Command("bash", "-c", `aws sts get-caller-identity --profile `+profile+` | jq -r '[ "  👤 Account: \(.Account)", "  👤 User ID: \(.UserId)", "  👤 ARN: \(.Arn)" ] | .[]'`)

	output, err := cmd.Output()
	if err != nil {
		fmt.Fprintln(os.Stderr, yellow("\nInfo: Could not retrieve session details (jq might not be installed)."))
		return
	}

	fmt.Println(cyan("--- Active Session Information ---"))
	fmt.Print(string(output))
	fmt.Println(cyan("--------------------------------"))
}
