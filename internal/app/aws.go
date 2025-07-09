package app

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"

	"github.com/manifoldco/promptui"
)

// getAWSProfiles reads the ~/.aws/config file and returns a list of available profiles.
func getAWSProfiles(configPath string) ([]string, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file %s: %w", configPath, err)
	}
	defer file.Close()

	var profiles []string
	re := regexp.MustCompile(`^\[profile\s+(.+?)\]$`)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		matches := re.FindStringSubmatch(line)
		if len(matches) > 1 {
			profiles = append(profiles, matches[1])
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error while reading file: %w", err)
	}
	return profiles, nil
}

// getRegionForProfile gets the region for a given AWS profile.
func getRegionForProfile(profile string) (string, error) {
	region := getPropertyForProfile(profile, "region")
	if region != "" {
		return region, nil
	}

	fmt.Println("Info: Region is not configured for this profile.")

	prompt := promptui.Prompt{Label: "Enter AWS Region", Default: "ap-southeast-1"}
	result, err := prompt.Run()
	if err != nil {
		return "", fmt.Errorf("region selection cancelled: %w", err)
	}
	return result, nil
}

// Helper to run 'aws configure get'
func getPropertyForProfile(profile, key string) string {
	cmd := exec.Command("aws", "configure", "get", key, "--profile", profile)
	output, err := cmd.Output()
	if err != nil {
		return "" // Key or profile does not exist
	}
	return strings.TrimSpace(string(output))
}

func isSsoProfile(profile string) bool {
	ssoAccountId := getPropertyForProfile(profile, "sso_account_id")

	return ssoAccountId != ""
}

// getAndSelectInstance fetches a list of EC2 instances and prompts the user to select one.
// It also handles expired SSO tokens.
func getAndSelectInstance(profile, region string, retryCount int) (*EC2Instance, error) {
	fmt.Println(Cyan("ℹ️  Fetching running EC2 instances..."))
	cmd := exec.Command("aws", "ec2", "describe-instances", "--profile", profile, "--region", region, "--filters", "Name=instance-state-name,Values=running", "--query", "Reservations[].Instances[].{ID:InstanceId,Name:Tags[?Key=='Name']|[0].Value}", "--output", "json")
	output, err := cmd.CombinedOutput()

	if err == nil {
		var instances []EC2Instance
		if err := json.Unmarshal(output, &instances); err != nil {
			return nil, fmt.Errorf("failed to parse JSON output: %w", err)
		}
		if len(instances) == 0 {
			return nil, fmt.Errorf("no running EC2 instances found in region %s", region)
		}
		usageData, err := loadInstanceUsageData()
		if err != nil {
			return nil, err
		}
		for i := range instances {
			instances[i].UsageCount = usageData[instances[i].ID]
		}
		sort.Slice(instances, func(i, j int) bool {
			return instances[i].UsageCount > instances[j].UsageCount
		})
		return selectInstanceFromList(instances)
	}

	if retryCount > 0 {
		return nil, fmt.Errorf("failed to refresh SSO token after retry")
	}

	if canRetry, err := executeRefreshProfileAction(profile); err != nil {
		return nil, err
	} else if canRetry {
		fmt.Println(Cyan("🔁 Retrying..."))
		return getAndSelectInstance(profile, region, retryCount+1)
	}

	return nil, fmt.Errorf("failed to fetch running EC2 instances: %s", string(output))
}

// executeFinalAction executes the final command or displays it if in dry run mode.
func executeFinalAction(sc *Shortcut, region string, dryRun bool) error {
	var args []string
	if sc.Action == "Start Session (SSM)" {
		fmt.Println(Cyan("Preparing SSM session..."))
		args = []string{"ssm", "start-session", "--target", sc.InstanceID, "--profile", sc.Profile, "--region", region}
	} else if sc.Action == "Connect RDS" {
		fmt.Println(Cyan("Preparing port forwarding to RDS..."))
		allConfigs, _ := loadRDSConfig()
		var targetRDS RDSConfig
		for _, conf := range allConfigs {
			confID := fmt.Sprintf("%s|%s|%s", conf.Key, conf.Env, conf.Type)
			if confID == sc.RDS_ID {
				targetRDS = conf
				break
			}
		}
		if targetRDS.Endpoint == "" {
			return fmt.Errorf("RDS configuration for shortcut not found")
		}
		parameters := fmt.Sprintf("host=%s,portNumber=%d,localPortNumber=%d", targetRDS.Endpoint, targetRDS.Port, targetRDS.LocalPort)
		args = []string{"ssm", "start-session", "--target", sc.InstanceID, "--profile", sc.Profile, "--region", region, "--document-name", "AWS-StartPortForwardingSessionToRemoteHost", "--parameters", parameters}
		fmt.Printf(Cyan("Target: %s -> localhost:%d\n"), targetRDS.Endpoint, targetRDS.LocalPort)
	}

	if dryRun {
		fullCommand := "aws " + strings.Join(args, " ")
		fmt.Println(Cyan("\n-- DRY RUN MODE --"))
		fmt.Println("Command to be executed:")
		fmt.Println(Yellow(fullCommand))
		return nil
	}

	return executeInteractiveAWSCommand(sc.Profile, args, 0)
}

// executeInteractiveAWSCommand runs an AWS command that requires user interaction.
func executeInteractiveAWSCommand(profile string, args []string, retryCount int) error {
	cmd := exec.Command("aws", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout

	var stderrBuf bytes.Buffer
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)

	err := cmd.Run()
	stderrString := stderrBuf.String()

	if err == nil {
		return nil
	}

	if retryCount > 0 {
		return fmt.Errorf("failed to refresh SSO token after retry")
	}

	if canRetry, err := executeRefreshProfileAction(profile); err != nil {
		return err
	} else if canRetry {
		fmt.Println(Cyan("🔁 Retrying..."))
		return executeInteractiveAWSCommand(profile, args, retryCount+1)
	}

	return fmt.Errorf("failed to run AWS CLI command: %s", stderrString)
}

func executeRefreshProfileAction(profile string) (bool, error) {
	if !isSsoProfile(profile) {
		return false, nil
	}

	fmt.Println(Yellow("⚠️  SSO token has expired. Starting to refresh the token..."))

	ssoRefreshed, err := ssoRefresh(profile)
	if err != nil {
		return false, err
	}
	if !ssoRefreshed {
		return false, fmt.Errorf("❌ Refresh token canceled due to non SSO profile being used")
	}

	return true, nil
}
