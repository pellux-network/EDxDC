package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/ncruces/zenity"
)

// RunUpdaterWithLog performs the update process from a temporary updater executable.
// It deletes only known files and directories from the old version directory (for safety),
// shows a Zenity dialog on success/failure, and then starts the new version.
// All logs are written to the provided logFile.
func RunUpdaterWithLog(oldDir, newExe, newDir, logFile string) error {
	// Open the log file for writing update progress and errors.
	if logFile == "" {
		parentDir := filepath.Dir(oldDir)
		logFile = filepath.Join(parentDir, "updater.log")
	}
	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		zenity.Error(fmt.Sprintf("Failed to open updater log file: %v", err), zenity.Title("EDx52 Updater"))
		return err
	}
	defer f.Close()
	logf := func(format string, args ...interface{}) {
		fmt.Fprintf(f, format+"\n", args...)
		f.Sync()
	}

	logf("Updater process started")
	logf("oldDir=%s, newExe=%s, newDir=%s, parentDir=%s", oldDir, newExe, newDir, filepath.Dir(oldDir))
	cwd, _ := os.Getwd()
	logf("Current working directory before chdir: %s", cwd)
	// Change working directory to the parent of the old directory to avoid locking it.
	if err := os.Chdir(filepath.Dir(oldDir)); err != nil {
		logf("Failed to change working directory: %v", err)
	}
	cwd, _ = os.Getwd()
	logf("Current working directory after chdir: %s", cwd)

	// Wait briefly to ensure the old process has exited and file locks are released.
	time.Sleep(2 * time.Second)
	for i := 0; i < 30; i++ {
		if !isDirLocked(oldDir) {
			break
		}
		logf("Directory still locked, waiting...")
		time.Sleep(1 * time.Second)
	}

	// Attempt to kill any remaining process running the old exe by name.
	oldExe, exeErr := findExeInDir(oldDir)
	if exeErr == nil && oldExe != "" {
		oldExeName := filepath.Base(oldExe)
		logf("Attempting to kill old exe by name: %s", oldExeName)
		killErr := killProcessByExeName(oldExeName)
		if killErr != nil {
			logf("Failed to kill old exe: %v", killErr)
		} else {
			logf("Successfully killed old exe: %s", oldExeName)
		}
	} else {
		logf("No old exe found to kill in %s: %v", oldDir, exeErr)
	}

	// Wait a moment to ensure the old process is gone.
	time.Sleep(1 * time.Second)

	// Close the log file before starting the new process or deleting the directory.
	f.Close()

	// List of known files and directories to delete (update as needed for your app)
	knownFiles := []string{
		"EDxDC*.exe", // all versioned exes
		"conf.yaml",
		"LICENSE",
		"README.md",
	}
	knownDirs := []string{
		"names",
		"bin",
	}

	// Try to delete the old directory contents, retrying if necessary.
	var updateErr error
	for i := 0; i < 10; i++ {
		updateErr = safeRemoveOldDir(oldDir, knownFiles, knownDirs)
		if updateErr == nil {
			break
		}
		time.Sleep(1 * time.Second)
	}
	if updateErr != nil {
		// Show a Zenity error dialog if deletion fails.
		_ = zenity.Error(fmt.Sprintf("Update failed: %v", updateErr), zenity.Title("EDxDC Updater"))
		return fmt.Errorf("failed to remove old dir %s: %w", oldDir, updateErr)
	}

	// Schedule self-deletion of the updater exe if running from %TEMP%.
	selfPath, _ := os.Executable()
	if strings.Contains(strings.ToLower(selfPath), os.TempDir()) {
		// Use a delayed cmd to delete the updater exe after exit.
		exec.Command("cmd", "/C", "ping", "127.0.0.1", "-n", "2", ">", "NUL", "&&", "del", selfPath).Start()
	}

	// Show a Zenity info dialog to inform the user the update is complete.
	_ = zenity.Info("Update complete! Starting new version.", zenity.Title("EDxDC Updater"))

	// Start the new version after the user dismisses the info dialog.
	cmd := exec.Command(newExe)
	cmd.Dir = newDir
	cmd.SysProcAttr = getSysProcAttr()
	if err := cmd.Start(); err != nil {
		_ = zenity.Error(fmt.Sprintf("Failed to start new exe: %v", err), zenity.Title("EDxDC Updater"))
		return fmt.Errorf("failed to start new exe: %w", err)
	}

	return nil
}

// findExeInDir returns the first .exe file found in the given directory.
func findExeInDir(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), ".exe") {
			return filepath.Join(dir, entry.Name()), nil
		}
	}
	return "", fmt.Errorf("no .exe found in %s", dir)
}

// killProcessByExeName attempts to kill all processes by exe filename (Windows only).
func killProcessByExeName(exeName string) error {
	if runtime.GOOS != "windows" {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "taskkill", "/F", "/T", "/IM", exeName, "/FI", "STATUS eq RUNNING")
	output, err := cmd.CombinedOutput()
	fmt.Printf("taskkill output: %s\n", string(output))
	return err
}

// isDirLocked tries to create a file in the directory to check if it's locked (Windows).
func isDirLocked(dir string) bool {
	test := filepath.Join(dir, ".locktest")
	f, err := os.Create(test)
	if err != nil {
		return true
	}
	f.Close()
	os.Remove(test)
	return false
}

// getSysProcAttr returns the correct SysProcAttr for detached process on Windows.
func getSysProcAttr() *syscall.SysProcAttr {
	if runtime.GOOS == "windows" {
		return &syscall.SysProcAttr{HideWindow: true}
	}
	return nil
}

// safeRemoveOldDir deletes only known files (using glob patterns) and directories in the old directory.
// If the directory is empty after deletion, it will be removed as well.
func safeRemoveOldDir(oldDir string, knownFiles []string, knownDirs []string) error {
	// Remove files matching patterns
	for _, pattern := range knownFiles {
		matches, _ := filepath.Glob(filepath.Join(oldDir, pattern))
		for _, match := range matches {
			_ = os.Remove(match)
		}
	}
	for _, dir := range knownDirs {
		_ = os.RemoveAll(filepath.Join(oldDir, dir))
	}
	// Try to remove the directory itself if empty (ignore error if not empty)
	_ = os.Remove(oldDir)
	return nil
}
