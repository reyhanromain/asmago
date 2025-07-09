package app

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/manifoldco/promptui"
)

// App is the central struct of the application.
type App struct {
	shortcutMgr *ShortcutManager
	DryRun      bool
}

// NewApp is the constructor for creating a new application instance.
func NewApp(dryRun bool) (*App, error) {
	// Ensure the user configuration file exists.
	if err := ensureUserConfigExists(); err != nil {
		return nil, fmt.Errorf("failed during configuration initialization: %w", err)
	}

	// Check for required dependencies.
	if err := checkDependencies(); err != nil {
		return nil, err
	}

	shortcutMgr, err := newShortcutManager()
	if err != nil {
		return nil, err
	}
	return &App{shortcutMgr: shortcutMgr, DryRun: dryRun}, nil
}

// Run is the main entry point for running the application.
func (a *App) Run() error {
	shortcutList, displayItems := a.shortcutMgr.getDisplayList()

	var selectedShortcut *Shortcut
	if len(shortcutList) > 0 {
		displayItems = append(displayItems, manualFlowChoice)
		prompt := promptui.Select{Label: "Select Shortcut or Run Manual Flow", Items: displayItems, Size: 10}
		index, result, err := prompt.Run()
		if err != nil {
			fmt.Println("Process cancelled.")
			return nil
		}
		if result != manualFlowChoice {
			selectedShortcut = &shortcutList[index]
		}
	}

	if selectedShortcut != nil {
		return a.executeShortcut(selectedShortcut)
	}
	return a.runManualFlow()
}

// ListShortcuts displays the most frequently and last used shortcuts.
func (a *App) ListShortcuts() error {
	shortcutList, _ := a.shortcutMgr.getDisplayList()
	if len(shortcutList) == 0 {
		fmt.Println(Yellow("No shortcuts found."))
		return nil
	}

	fmt.Println(Cyan("Top 5 Shortcuts:"))
	for i, sc := range shortcutList {
		remark := ""
		if i == 0 && a.shortcutMgr.lastUsedKey != "" && sc.DisplayString == a.shortcutMgr.shortcuts[a.shortcutMgr.lastUsedKey].DisplayString {
			remark = Yellow(" (last used)")
		}

		fmt.Printf(" %d. %s%s\n", i+1, sc.DisplayString, remark)
	}
	return nil
}

// executeShortcut runs the workflow based on a selected shortcut.
func (a *App) executeShortcut(sc *Shortcut) error {
	fmt.Printf(Cyan("--- Running Shortcut: %s ---\n"), sc.DisplayString)
	region, err := getRegionForProfile(sc.Profile)
	if err != nil {
		return fmt.Errorf("failed to get region for shortcut: %w", err)
	}

	a.shortcutMgr.addOrUpdate(*sc)

	if err := executeFinalAction(sc, region, a.DryRun); err != nil {
		return fmt.Errorf("failed to execute shortcut: %w", err)
	}

	if !a.DryRun {
		fmt.Println(Green("\nShortcut executed successfully."))
	}
	return nil
}

// runManualFlow runs the step-by-step manual workflow.
func (a *App) runManualFlow() error {
	fmt.Println(Cyan("--- Running Manual Flow ---"))

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	configPath := filepath.Join(homeDir, ".aws", "config")

	profiles, err := getAWSProfiles(configPath)
	if err != nil {
		return err
	}
	if len(profiles) == 0 {
		return fmt.Errorf("no AWS profiles found")
	}

	promptSelectProfile := promptui.Select{Label: "Select AWS Profile", Items: profiles, Searcher: func(input string, index int) bool { return fuzzy.Match(input, profiles[index]) }}
	_, selectedProfile, err := promptSelectProfile.Run()
	if err != nil {
		return fmt.Errorf("selection cancelled")
	}

	selectedRegion, err := getRegionForProfile(selectedProfile)
	if err != nil {
		return err
	}
	fmt.Println("-------------------------------------")
	fmt.Printf("Using Profile: %s, Region: %s\n", selectedProfile, selectedRegion)
	fmt.Println("-------------------------------------")

	selectedInstance, err := getAndSelectInstance(selectedProfile, selectedRegion, 0)
	if err != nil {
		return err
	}
	if selectedInstance == nil {
		fmt.Println(Yellow("\nProcess aborted by user."))
		return nil
	}

	instanceNameStr := *selectedInstance.Name
	if selectedInstance.Name == nil || *selectedInstance.Name == "" {
		instanceNameStr = selectedInstance.ID
	}
	fmt.Println("-------------------------------------")
	fmt.Printf("✅ Instance Selected: %s (%s)\n", instanceNameStr, selectedInstance.ID)
	fmt.Println("-------------------------------------")

	actionItems := []string{"Start Session (SSM)", "Connect RDS"}
	actionPrompt := promptui.Select{Label: "Select Action", Items: actionItems, Searcher: func(input string, index int) bool { return fuzzy.Match(input, actionItems[index]) }}
	_, selectedAction, err := actionPrompt.Run()
	if err != nil {
		return fmt.Errorf("selection cancelled")
	}

	var rdsID string
	if selectedAction == "Connect RDS" {
		rdsConfig, err := handleRDSSelection(selectedInstance)
		if err != nil {
			return err
		}
		if rdsConfig == nil {
			fmt.Println(Yellow("\nProcess aborted by user."))
			return nil
		}
		rdsID = fmt.Sprintf("%s|%s|%s", rdsConfig.Key, rdsConfig.Env, rdsConfig.Type)
	}

	shortcut := Shortcut{
		Profile:      selectedProfile,
		InstanceID:   selectedInstance.ID,
		InstanceName: instanceNameStr,
		Action:       selectedAction,
		RDS_ID:       rdsID,
	}

	a.shortcutMgr.addOrUpdate(shortcut)
	fmt.Println(Green("✅ Scenario successfully saved/updated as a shortcut."))

	return executeFinalAction(&shortcut, selectedRegion, a.DryRun)
}
