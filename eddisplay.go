package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"

	_ "embed"

	"github.com/getlantern/systray"
	"github.com/ncruces/zenity"
	"github.com/pellux-network/EDxDC/conf"
	"github.com/pellux-network/EDxDC/edreader"
	"github.com/pellux-network/EDxDC/edsm"
	"github.com/pellux-network/EDxDC/mfd"
)

// TextLogFormatter gives me custom command-line formatting
type TextLogFormatter struct{}

//go:embed icon.ico
var iconData []byte

const AppVersion = "v1.0.0-beta"

func (f *TextLogFormatter) Format(entry *log.Entry) ([]byte, error) {
	timestamp := time.Now().UTC().Format("2006-01-02 15:04:05")
	level := entry.Level.String()
	message := entry.Message

	return []byte(timestamp + " - " + strings.ToUpper(level) + " - " + message + "\n"), nil
}

func cleanupOldUpdaters() {
	tmpDir := filepath.Join(os.TempDir(), "EDxDC")
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		_ = os.Remove(filepath.Join(tmpDir, entry.Name()))
	}
	// Optionally remove the directory itself if empty
	_ = os.Remove(tmpDir)
}

func main() {
	cleanupOldUpdaters()

	if len(os.Args) > 1 && os.Args[1] == "run-updater" {
		if len(os.Args) < 5 {
			fmt.Println("Usage: run-updater oldDir newExe newDir [logFile]")
			os.Exit(1)
		}
		oldDir := os.Args[2]
		newExe := os.Args[3]
		newDir := os.Args[4]
		logFile := ""
		if len(os.Args) > 5 {
			logFile = os.Args[5]
		}
		if err := RunUpdaterWithLog(oldDir, newExe, newDir, logFile); err != nil {
			fmt.Println("Updater error:", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	systray.Run(onReady, onExit)
}

func onReady() {
	// Set up systray icon and menu
	systray.SetIcon(getIcon()) // You can provide your own icon as []byte
	systray.SetTitle("EDxDC")
	systray.SetTooltip("EDxDC is running")

	// Show notification on successful start
	_ = zenity.Notify("EDxDC has started successfully.", zenity.Title("EDxDC"))

	mAbout := systray.AddMenuItem("About", "About EDxDC")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Quit EDxDC")

	// Check for updates at startup (non-blocking)
	CheckForUpdate(AppVersion)

	// Handle About menu click
	go func() {
		for range mAbout.ClickedCh {
			showAboutDialog()
		}
	}()

	// Start your main logic in a goroutine
	go func() {
		defer func() {
			// Attempt to catch any crash messages before the cmd window closes
			if r := recover(); r != nil {
				log.Warnln("Crashed with message")
				log.Warnln(r)
			}
		}()
		var logLevelArg string
		flag.StringVar(&logLevelArg, "log", "trace", "Desired log level. One of [panic, fatal, error, warning, info, debug, trace]. Default: trace.")

		flag.Parse()
		logLevel, err := log.ParseLevel(logLevelArg)
		if err != nil {
			log.Panic(err)
		}

		log.SetLevel(logLevel)
		log.SetFormatter(&TextLogFormatter{})

		// Ensure logs directory exists
		logDir := "logs"
		_ = os.MkdirAll(logDir, 0755)
		logFileName := time.Now().Format("2006-01-02_15.04.05") + ".log"
		logPath := filepath.Join(logDir, logFileName)

		// Set up log rotation
		log.SetOutput(&lumberjack.Logger{
			Filename:   logPath,
			MaxSize:    10, // megabytes
			MaxBackups: 5,
			MaxAge:     30,   //days
			Compress:   true, // compress old logs
		})

		log.Infof("Logging to %s", logPath)

		conf := conf.LoadConf()

		// Calculate number of enabled pages
		pageCount := 0
		for _, enabled := range conf.Pages {
			if enabled {
				pageCount++
			}
		}

		err = mfd.InitDevice(uint32(pageCount), edsm.ClearCache)
		if err != nil {
			log.Panic(err)
		}
		defer mfd.DeInitDevice()

		edreader.Start(conf)
		defer edreader.Stop()

		// Wait for quit
		<-mQuit.ClickedCh
		systray.Quit()
	}()
}

func onExit() {
	// Cleanup tasks if needed
}

// getIcon returns an icon as []byte. Replace with your own icon if desired.
func getIcon() []byte {
	return iconData
}

func showAboutDialog() {
	aboutText := fmt.Sprintf("EDxDC %s\n\nSeamlessly reads Elite Dangerous journal data and presents real-time system, planet, cargo, and other information on your Saitek/Logitech X52 Pro Multi-Function Display.\n\n© Pellux Network", AppVersion)
	_ = zenity.Info(aboutText, zenity.Title("About EDxDC"))
}
