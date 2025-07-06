package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
)

// Color Variables
var (
	Yellow = color.New(color.FgYellow).SprintFunc()
	Green  = color.New(color.FgGreen).SprintFunc()
	Cyan   = color.New(color.FgCyan).SprintFunc()
)

// Constants
const (
	manualFlowChoice = "[ --- Run Manual Flow --- ]"
)

// --- EXPORTED PATH FUNCTIONS ---

// GetConfigDir returns the path to the application's configuration directory.
// Example: ~/.config/asmago/
func GetConfigDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("could not find user config directory: %w", err)
	}
	appConfigDir := filepath.Join(configDir, "asmago")
	return appConfigDir, nil
}

// GetDataDir returns the path to the application's data directory.
// Example: ~/.local/share/asmago/
func GetDataDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not find user home directory: %w", err)
	}
	appDataDir := filepath.Join(homeDir, ".local", "share", "asmago")
	return appDataDir, nil
}

// Data Structure Definitions
type Shortcut struct {
	Profile       string
	InstanceID    string
	InstanceName  string
	Action        string
	RDS_ID        string
	DisplayString string
	UsageCount    int
}

type EC2Instance struct {
	ID         string
	Name       *string `json:"Name"`
	UsageCount int     `json:"-"`
}

type RDSConfig struct {
	Key        string `json:"key"`
	Env        string `json:"env"`
	Type       string `json:"type"`
	Endpoint   string `json:"endpoint"`
	Port       int    `json:"port"`
	LocalPort  int    `json:"local_port"`
	UsageCount int    `json:"-"`
}

// Data Loading Functions
func loadShortcuts() (map[string]Shortcut, error) {
	dataDir, err := GetDataDir()
	if err != nil {
		return nil, err
	}
	filePath := filepath.Join(dataDir, "shortcuts.json")

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]Shortcut), nil
		}
		return nil, err
	}
	var shortcuts map[string]Shortcut
	if err := json.Unmarshal(data, &shortcuts); err != nil {
		return nil, fmt.Errorf("file %s is corrupted: %w", filePath, err)
	}
	return shortcuts, nil
}

func loadLastShortcutKey() (string, error) {
	dataDir, err := GetDataDir()
	if err != nil {
		return "", err
	}
	filePath := filepath.Join(dataDir, "last_shortcut.txt")

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}

func loadInstanceUsageData() (map[string]int, error) {
	dataDir, err := GetDataDir()
	if err != nil {
		return nil, err
	}
	filePath := filepath.Join(dataDir, "instance_usage.json")

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]int), nil
		}
		return nil, err
	}
	var usageMap map[string]int
	if err := json.Unmarshal(data, &usageMap); err != nil {
		return nil, fmt.Errorf("file %s is corrupted: %w", filePath, err)
	}
	return usageMap, nil
}

func saveInstanceUsageData(data map[string]int) error {
	dataDir, err := GetDataDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return err
	}

	filePath := filepath.Join(dataDir, "instance_usage.json")
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, bytes, 0644)
}

func loadRDSConfig() ([]RDSConfig, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return nil, err
	}
	filePath := filepath.Join(configDir, "rds.json")

	configFile, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read rds.json at %s; please ensure it exists or place a template in the app's config/ directory", filePath)
	}

	var rdsConfigs []RDSConfig
	err = json.Unmarshal(configFile, &rdsConfigs)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", filePath, err)
	}
	return rdsConfigs, nil
}

func loadRdsUsageData() (map[string]int, error) {
	dataDir, err := GetDataDir()
	if err != nil {
		return nil, err
	}
	filePath := filepath.Join(dataDir, "rds_usage.json")

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]int), nil
		}
		return nil, err
	}
	var usageMap map[string]int
	if err := json.Unmarshal(data, &usageMap); err != nil {
		return nil, fmt.Errorf("file %s is corrupted: %w", filePath, err)
	}
	return usageMap, nil
}

func saveRdsUsageData(data map[string]int) error {
	dataDir, err := GetDataDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return err
	}

	filePath := filepath.Join(dataDir, "rds_usage.json")
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, bytes, 0644)
}
