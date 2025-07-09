package app

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
)

func ssoRefresh(profile string) (bool, error) {
	// Replicate the Bash logic to get specific configurations
	ssoSession := getPropertyForProfile(profile, "sso_session")
	ssoStartURL := getPropertyForProfile(profile, "sso_start_url")

	// if the profile is not an SSO profile, then
	// returns false to indicates that the process can't run any further but it's not an error
	if ssoSession == "" && ssoStartURL == "" {
		return false, nil
	}

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
		fmt.Fprintln(os.Stderr, Red("\n❌ Failed to update token for profile '"+profile+"'."))
		fmt.Fprintln(os.Stderr, "Error Details:", stderr.String())
		os.Exit(1)
	}

	fmt.Println(Green("\n🚀 SSO token for profile '" + profile + "' has been successfully updated!"))

	return true, nil
}
