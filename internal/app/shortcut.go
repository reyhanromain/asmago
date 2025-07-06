package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ShortcutManager manages shortcut data.
type ShortcutManager struct {
	shortcuts   map[string]Shortcut
	lastUsedKey string
}

// newShortcutManager creates a new instance of ShortcutManager.
func newShortcutManager() (*ShortcutManager, error) {
	shortcuts, err := loadShortcuts()
	if err != nil {
		return nil, err
	}
	lastUsedKey, err := loadLastShortcutKey()
	if err != nil {
		return nil, err
	}
	return &ShortcutManager{shortcuts: shortcuts, lastUsedKey: lastUsedKey}, nil
}

// addOrUpdate adds a new shortcut or updates the usage count of an existing one.
func (sm *ShortcutManager) addOrUpdate(shortcut Shortcut) {
	key := fmt.Sprintf("%s;%s;%s;%s", shortcut.Profile, shortcut.InstanceID, shortcut.Action, shortcut.RDS_ID)

	var rdsPart string
	if shortcut.Action == "Connect RDS" {
		rdsKey := strings.Split(shortcut.RDS_ID, "|")[0]
		rdsPart = fmt.Sprintf(" -> Connect RDS (%s)", rdsKey)
	} else {
		rdsPart = " -> Start Session (SSM)"
	}
	shortcut.DisplayString = fmt.Sprintf("%s -> %s%s", shortcut.Profile, shortcut.InstanceName, rdsPart)

	if sc, ok := sm.shortcuts[key]; ok {
		sc.UsageCount++
		sc.DisplayString = shortcut.DisplayString
		sm.shortcuts[key] = sc
	} else {
		shortcut.UsageCount = 1
		sm.shortcuts[key] = shortcut
	}

	sm.lastUsedKey = key
	if err := sm.save(); err != nil {
		fmt.Println(Yellow("Warning: Failed to save shortcut data: %v", err))
	}
}

// save persists the shortcut data to disk.
func (sm *ShortcutManager) save() error {
	dataDir, err := GetDataDir()
	if err != nil {
		return err
	}

	// Ensure the data directory exists.
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return err
	}

	// Save shortcuts.json
	shortcutsPath := filepath.Join(dataDir, "shortcuts.json")
	bytes, err := json.MarshalIndent(sm.shortcuts, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(shortcutsPath, bytes, 0644); err != nil {
		return err
	}

	// Save last_shortcut.txt
	lastShortcutPath := filepath.Join(dataDir, "last_shortcut.txt")
	return os.WriteFile(lastShortcutPath, []byte(sm.lastUsedKey), 0644)
}

// getDisplayList prepares the sorted list of shortcuts for display.
func (sm *ShortcutManager) getDisplayList() ([]Shortcut, []string) {
	if len(sm.shortcuts) == 0 {
		return nil, nil
	}

	var mostFrequent []Shortcut
	var lastUsedShortcut *Shortcut
	var finalShortcutList []Shortcut
	var displayItems []string

	if sm.lastUsedKey != "" {
		if sc, ok := sm.shortcuts[sm.lastUsedKey]; ok {
			lastUsedShortcut = &sc
		}
	}

	for key, sc := range sm.shortcuts {
		if lastUsedShortcut != nil && key == sm.lastUsedKey {
			continue
		}
		mostFrequent = append(mostFrequent, sc)
	}
	sort.Slice(mostFrequent, func(i, j int) bool {
		return mostFrequent[i].UsageCount > mostFrequent[j].UsageCount
	})

	if lastUsedShortcut != nil {
		finalShortcutList = append(finalShortcutList, *lastUsedShortcut)
		displayItems = append(displayItems, lastUsedShortcut.DisplayString+" (last used)")
	}

	remainingSlots := 5 - len(finalShortcutList)
	if remainingSlots > 0 && len(mostFrequent) > 0 {
		limit := remainingSlots
		if len(mostFrequent) < limit {
			limit = len(mostFrequent)
		}
		finalShortcutList = append(finalShortcutList, mostFrequent[:limit]...)
		for i := 0; i < limit; i++ {
			displayItems = append(displayItems, mostFrequent[i].DisplayString)
		}
	}

	return finalShortcutList, displayItems
}
