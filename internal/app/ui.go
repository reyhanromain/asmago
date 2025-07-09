package app

import (
	"fmt"
	"sort"
	"strings"

	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/manifoldco/promptui"
)

// selectInstanceFromList prompts the user to select an EC2 instance from a list.
func selectInstanceFromList(instances []EC2Instance) (*EC2Instance, error) {
	formattedItems := make([]string, len(instances))
	for i, inst := range instances {
		if inst.Name != nil && *inst.Name != "" {
			formattedItems[i] = fmt.Sprintf("%s (%s)", *inst.Name, inst.ID)
		} else {
			formattedItems[i] = fmt.Sprintf("%s (No 'Name' tag)", inst.ID)
		}
	}

	prompt := promptui.Select{
		Label:    "Select Instance",
		Items:    formattedItems,
		Size:     10,
		Searcher: func(input string, index int) bool { return fuzzy.Match(input, formattedItems[index]) },
	}
	index, _, err := prompt.Run()
	if err != nil {
		return nil, fmt.Errorf("instance selection cancelled")
	}

	selectedInstance := instances[index]

	// Update usage data
	usageData, err := loadInstanceUsageData()
	if err != nil {
		fmt.Println(Yellow("Warning: Failed to load instance usage data: %v", err))
	} else {
		usageData[selectedInstance.ID]++
		if err := saveInstanceUsageData(usageData); err != nil {
			fmt.Println(Yellow("Warning: Failed to save instance usage data: %v", err))
		}
	}

	return &selectedInstance, nil
}

// handleRDSSelection guides the user through selecting an RDS target.
func handleRDSSelection(selectedInstance *EC2Instance) (*RDSConfig, error) {
	allConfigs, err := loadRDSConfig()
	if err != nil {
		return nil, err
	}
	usageData, err := loadRdsUsageData()
	if err != nil {
		return nil, err
	}
	for i, config := range allConfigs {
		configKey := fmt.Sprintf("%s|%s|%s", config.Key, config.Env, config.Type)
		allConfigs[i].UsageCount = usageData[configKey]
	}

	// Filter by environment based on instance name
	var envFilter string
	if selectedInstance.Name != nil {
		instanceName := *selectedInstance.Name
		if strings.HasPrefix(instanceName, "dev") {
			envFilter = "dev"
			fmt.Println(Cyan("ℹ️  'dev' instance detected, showing RDS for 'dev' env..."))
		} else if strings.HasPrefix(instanceName, "qa") {
			envFilter = "qa"
			fmt.Println(Cyan("ℹ️  'qa' instance detected, showing RDS for 'qa' env..."))
		}
	}
	var availableConfigs []RDSConfig
	if envFilter == "" {
		availableConfigs = allConfigs
	} else {
		for _, config := range allConfigs {
			if config.Env == envFilter {
				availableConfigs = append(availableConfigs, config)
			}
		}
	}
	if len(availableConfigs) == 0 {
		return nil, fmt.Errorf("no matching RDS configurations found")
	}

	// Select connection type
	typeItems := []string{"read", "write"}
	typePrompt := promptui.Select{Label: "Select RDS Connection Type", Items: typeItems, Searcher: func(input string, index int) bool { return fuzzy.Match(input, typeItems[index]) }}
	_, selectedType, err := typePrompt.Run()
	if err != nil {
		return nil, nil // User cancelled
	}

	var finalConfigs []RDSConfig
	for _, config := range availableConfigs {
		if config.Type == selectedType {
			finalConfigs = append(finalConfigs, config)
		}
	}
	if len(finalConfigs) == 0 {
		return nil, fmt.Errorf("no RDS configurations found for type '%s'", selectedType)
	}

	// Select RDS target
	sort.Slice(finalConfigs, func(i, j int) bool { return finalConfigs[i].UsageCount > finalConfigs[j].UsageCount })
	var finalKeys []string
	for _, config := range finalConfigs {
		displayKey := config.Key
		if envFilter == "" {
			displayKey = fmt.Sprintf("%s (%s)", config.Key, config.Env)
		}
		finalKeys = append(finalKeys, displayKey)
	}
	keyPrompt := promptui.Select{Label: "Select RDS Target", Items: finalKeys, Searcher: func(input string, index int) bool { return fuzzy.Match(input, finalKeys[index]) }}
	selectedIndex, _, err := keyPrompt.Run()
	if err != nil {
		return nil, nil // User cancelled
	}
	selectedRDS := finalConfigs[selectedIndex]

	// Save usage data
	configKey := fmt.Sprintf("%s|%s|%s", selectedRDS.Key, selectedRDS.Env, selectedRDS.Type)
	usageData[configKey]++
	if err := saveRdsUsageData(usageData); err != nil {
		fmt.Println(Yellow("Warning: Failed to save RDS usage data: %v", err))
	}

	return &selectedRDS, nil
}
