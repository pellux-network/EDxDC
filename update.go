package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/ncruces/zenity"
	log "github.com/sirupsen/logrus"
)

const githubRepo = "pellux-network/EDxDC"

// githubRelease represents the structure of a GitHub release as returned by the API.
type githubRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
	Assets  []struct {
		BrowserDownloadURL string `json:"browser_download_url"`
		Name               string `json:"name"`
	} `json:"assets"`
}

// CheckForUpdate checks GitHub for a new release and prompts the user if one is available.
func CheckForUpdate(currentVersion string) {
	client := &http.Client{Timeout: 5 * time.Second}
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", githubRepo)
	resp, err := client.Get(url)
	if err != nil {
		log.Warnf("Update check failed: %v", err)
		return
	}
	defer resp.Body.Close()

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		log.Warnf("Failed to parse update info: %v", err)
		return
	}

	latest := strings.TrimPrefix(release.TagName, "v")
	current := strings.TrimPrefix(currentVersion, "v")
	if latest != "" && latest != current {
		log.Infof("A new version is available: %s (current: %s). Download at: %s", release.TagName, currentVersion, release.HTMLURL)
		showUpdatePopup(release.TagName, release.HTMLURL, currentVersion, release)
	} else {
		log.Infof("You are running the latest version: %s", currentVersion)
	}
}

// showUpdatePopup displays a Zenity dialog prompting the user to update.
// Clicking "Release Page" opens the browser but does not dismiss the dialog.
func showUpdatePopup(version, url, currentVersion string, release githubRelease) {
	msg := fmt.Sprintf(
		"A new version (%s) is available! You are on version %s\n\nRelease page:\n%s",
		version, currentVersion, url,
	)
	title := "EDxDC Update Available"

	for {
		err := zenity.Question(
			msg,
			zenity.Title(title),
			zenity.OKLabel("Update"),
			zenity.ExtraButton("Release Page"),
			zenity.CancelLabel("Dismiss"),
		)
		switch err {
		case zenity.ErrExtraButton:
			openBrowser(url)
			// Loop again, do not dismiss
		case nil:
			// Run update synchronously and exit the app after starting the updater
			if err := doAutoUpdate(release); err != nil {
				zenity.Error(fmt.Sprintf("Update failed: %v", err), zenity.Title("Update Error"))
			}
			// doAutoUpdate will call os.Exit(0) after starting the updater
			return
		case zenity.ErrCanceled:
			// Dismiss (user clicked X or Cancel)
			return
		}
	}
}

// doAutoUpdate downloads, extracts, and launches the updater for the new version.
func doAutoUpdate(release githubRelease) error {
	// 1. Find the ZIP asset in the release.
	var zipURL, zipName string
	for _, asset := range release.Assets {
		if strings.HasSuffix(asset.Name, ".zip") {
			zipURL = asset.BrowserDownloadURL
			zipName = asset.Name
			break
		}
	}
	if zipURL == "" {
		return fmt.Errorf("no zip asset found in release")
	}

	// 2. Download the ZIP to a temp directory.
	tmpDir := os.TempDir()
	zipPath := filepath.Join(tmpDir, zipName)
	out, err := os.Create(zipPath)
	if err != nil {
		return fmt.Errorf("failed to create temp zip: %w", err)
	}
	defer out.Close()

	resp, err := http.Get(zipURL)
	if err != nil {
		return fmt.Errorf("failed to download zip: %w", err)
	}
	defer resp.Body.Close()
	if _, err := io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("failed to save zip: %w", err)
	}
	out.Close()
	defer os.Remove(zipPath) // Clean up ZIP after update

	// 3. Unzip to parent directory (not a subdirectory).
	exePath, _ := os.Executable()
	currentDir := filepath.Dir(exePath)
	parentDir := filepath.Dir(currentDir)

	// List directories before unzip to detect the new one after extraction.
	beforeDirs, _ := listDirs(parentDir)

	if err := unzip(zipPath, parentDir); err != nil {
		return fmt.Errorf("failed to unzip: %w", err)
	}

	// List directories after unzip and find the new one.
	afterDirs, _ := listDirs(parentDir)
	newDir := findNewDir(beforeDirs, afterDirs)
	if newDir == "" {
		return fmt.Errorf("could not find new directory after unzip")
	}

	// 4. Copy main.conf and logs to the new directory.
	copyFile(filepath.Join(currentDir, "main.conf"), filepath.Join(newDir, "main.conf"))
	copyDir(filepath.Join(currentDir, "logs"), filepath.Join(newDir, "logs"))

	// 5. Find new exe path (search for .exe in newDir).
	newExe, err := findExeInDir(newDir)
	if err != nil {
		return fmt.Errorf("failed to find new exe in %s: %w", newDir, err)
	}

	// 6. Start updater and exit.
	err = startUpdater(currentDir, newExe, newDir)
	if err != nil {
		return fmt.Errorf("failed to start updater: %w", err)
	}
	os.Exit(0)
	return nil
}

// listDirs returns a map of directory paths in the given parent directory.
func listDirs(parent string) (map[string]struct{}, error) {
	entries, err := os.ReadDir(parent)
	if err != nil {
		return nil, err
	}
	dirs := make(map[string]struct{})
	for _, entry := range entries {
		if entry.IsDir() {
			dirs[filepath.Join(parent, entry.Name())] = struct{}{}
		}
	}
	return dirs, nil
}

// findNewDir returns the directory present in 'after' but not in 'before'.
func findNewDir(before, after map[string]struct{}) string {
	for dir := range after {
		if _, ok := before[dir]; !ok {
			return dir
		}
	}
	return ""
}

// unzip extracts a zip archive to the destination directory.
func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()
	os.MkdirAll(dest, 0755)
	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)
		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, f.Mode())
			continue
		}
		if err := os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
			return err
		}
		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}
		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

// copyDir recursively copies a directory from src to dst.
func copyDir(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	os.MkdirAll(dst, 0755)
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}
	return nil
}

// openBrowser opens the given URL in the default browser.
func openBrowser(url string) {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	default: // linux, freebsd, etc.
		cmd = "xdg-open"
		args = []string{url}
	}
	_ = exec.Command(cmd, args...).Start()
}

// startUpdater copies the updater to a temp location, launches it, and passes all required arguments.
func startUpdater(oldDir, newExe, newDir string) error {
	exePath, err := os.Executable()
	if err != nil {
		fmt.Printf("startUpdater: os.Executable error: %v\n", err)
		return err
	}
	parentDir := filepath.Dir(oldDir)

	// Use a dedicated temp subdirectory for updater files
	tmpDir := filepath.Join(os.TempDir(), "EDxDC")
	_ = os.MkdirAll(tmpDir, 0700)
	timestamp := time.Now().UnixNano()
	tmpUpdater := filepath.Join(tmpDir, fmt.Sprintf("edx52_updater_%d.exe", timestamp))
	tmpLog := filepath.Join(tmpDir, fmt.Sprintf("edx52_updater_%d.log", timestamp))

	src, err := os.Open(exePath)
	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := os.Create(tmpUpdater)
	if err != nil {
		return err
	}
	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		return err
	}
	dst.Close()

	cmd := exec.Command(tmpUpdater, "run-updater", oldDir, newExe, newDir, tmpLog)
	cmd.Dir = parentDir

	f, err := os.OpenFile(tmpLog, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err == nil {
		cmd.Stdout = f
		cmd.Stderr = f
	} else {
		fmt.Printf("Failed to open updater log file: %v\n", err)
	}

	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	err = cmd.Start()
	if err != nil {
		fmt.Printf("startUpdater: cmd.Start error: %v\n", err)
	}
	return err
}
