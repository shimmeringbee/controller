package main

import (
	"fmt"
	"os"
	"time"
)

func safeWriteFile(name string, data []byte, perm os.FileMode) error {
	ut := time.Now().UnixNano() / int64(time.Millisecond)
	baseName := fmt.Sprintf("%s-%d", name, ut)
	newName := fmt.Sprintf("%s-new", baseName)
	oldName := fmt.Sprintf("%s-old", baseName)

	if err := os.WriteFile(newName, data, perm); err != nil {
		return fmt.Errorf("failed to write new file: %w", err)
	}

	_, err := os.Stat(name)
	oldExists := !os.IsNotExist(err)

	if oldExists {
		if err := os.Rename(name, oldName); err != nil {
			return fmt.Errorf("failed to move old file to temporary location: %w", err)
		}
	}

	if err := os.Rename(newName, name); err != nil {
		return fmt.Errorf("failed to move new file to file location: %w", err)
	}

	if oldExists {
		if err := os.Remove(oldName); err != nil {
			return fmt.Errorf("failed to remove old file: %w", err)
		}
	}

	return nil
}
