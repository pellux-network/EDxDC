package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/pellux-network/EDxDC/logging"
	"github.com/rs/zerolog/log"

	_ "embed"

	"github.com/abemedia/go-winsparkle"
	_ "github.com/abemedia/go-winsparkle/dll" // Embed DLL.
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

const AppVersion = "1.2.3-beta"

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
	exePath, err := os.Executable()
	if err != nil {
		fmt.Println("Could not determine executable path:", err)
		os.Exit(1)
	}
	exeDir := filepath.Dir(exePath)
	portablePath := filepath.Join(exeDir, ".portable")

	_, err = os.Stat(portablePath)
	isPortable := err == nil

	// Determine config path and load config before WinSparkle setup
	var baseDir string
	if isPortable {
		baseDir = exeDir
	} else {
		appDataDir, err := os.UserConfigDir()
		if err != nil {
			fmt.Println("Could not determine user config dir:", err)
			os.Exit(1)
		}
		baseDir = filepath.Join(appDataDir, "EDxDC")
		_ = os.MkdirAll(baseDir, 0755)
	}
	confPath := filepath.Join(baseDir, "main.conf")
	appConf := conf.LoadOrCreateConf(confPath)

	// Initialize logging before anything else
	logging.Init(baseDir, appConf.Loglevel)
	log.Info().Str("config", logging.CleanPath(confPath)).Msg("Loaded configuration")

	if !isPortable {
		// WinSparkle setup
		winsparkle.SetAppcastURL("https://pellux-network.github.io/EDxDC/appcast.xml")
		winsparkle.SetAppDetails("pellux-network.github.io/EDxDC", "EDxDC", AppVersion)
		winsparkle.SetAutomaticCheckForUpdates(appConf.CheckForUpdates)

		winsparkle.Init()
		winsparkle.CheckUpdateWithoutUI()
		log.Info().Msg("WinSparkle initialized for updates")
	} else {
		log.Info().Msg("Portable mode detected: using manual update function.")
	}

	cleanupOldUpdaters()

	if len(os.Args) > 1 && os.Args[1] == "run-updater" {
		if len(os.Args) < 5 {
			log.Error().Msg("Usage: run-updater oldDir newExe newDir [logFile]")
			os.Exit(1)
		}
		oldDir := os.Args[2]
		newExe := os.Args[3]
		newDir := os.Args[4]
		logFile := ""
		if len(os.Args) > 5 {
			logFile = os.Args[5]
		}
		log.Info().Str("oldDir", oldDir).Str("newExe", newExe).Str("newDir", newDir).Str("logFile", logFile).Msg("Running updater")
		if err := RunUpdaterWithLog(oldDir, newExe, newDir, logFile); err != nil {
			log.Error().Err(err).Msg("Updater error")
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Use a quit channel to coordinate shutdown
	quitCh := make(chan struct{})

	// Setup signal handling for graceful shutdown (works in most terminals)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		close(quitCh)
	}()

	// Register WinSparkle shutdown callback to trigger clean shutdown
	winsparkle.SetShutdownRequestCallback(func() {
		close(quitCh)
	})

	systray.Run(func() { onReady(quitCh, isPortable, appConf) }, onExit)
}

func onReady(quitCh chan struct{}, isPortable bool, conf conf.Conf) {
	// Set up systray icon and menu
	systray.SetIcon(getIcon()) // You can provide your own icon as []byte
	systray.SetTitle("EDxDC")
	systray.SetTooltip("EDxDC is running")

	// Show notification on successful start
	_ = zenity.Notify("EDxDC has started successfully.", zenity.Title("EDxDC"))

	mCheckUpdate := systray.AddMenuItem("Check for Updates", "Check for updates to EDxDC")
	mAbout := systray.AddMenuItem("About", "About EDxDC")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Quit EDxDC")

	// Check for updates at startup
	if isPortable {
		CheckForUpdate("v" + AppVersion) // Manual update function
	}

	log.Info().Msg("Systray ready, application started")

	// Handle Check for Updates menu click
	go func() {
		for range mCheckUpdate.ClickedCh {
			if isPortable {
				CheckForUpdate("v" + AppVersion)
			} else {
				winsparkle.CheckUpdateWithUI()
			}
		}
	}()

	// Handle About menu click
	go func() {
		for range mAbout.ClickedCh {
			showAboutDialog()
		}
	}()

	// Start your main logic in a goroutine
	go func() {
		var logLevelArg string
		flag.StringVar(&logLevelArg, "log", "trace", "Desired log level. One of [panic, fatal, error, warning, info, debug, trace]. Default: trace.")

		flag.Parse()
		log.Info().Str("level", conf.Loglevel).Msg("Setting log level")
		logging.SetLevel(conf.Loglevel)

		// Use app directory for config and logs in portable mode, otherwise use %APPDATA%\EDxDC
		var baseDir string
		if isPortable {
			exePath, err := os.Executable()
			if err != nil {
				log.Fatal().Err(err).Msg("Could not determine executable path")
			}
			baseDir = filepath.Dir(exePath)
		} else {
			appDataDir, err := os.UserConfigDir()
			if err != nil {
				log.Fatal().Err(err).Msg("Could not determine user config dir")
			}
			baseDir = filepath.Join(appDataDir, "EDxDC")
			_ = os.MkdirAll(baseDir, 0755)
		}

		// Logs directory inside baseDir
		logDir := filepath.Join(baseDir, "logs")
		_ = os.MkdirAll(logDir, 0755)
		logFileName := time.Now().Format("2006-01-02_15.04.05") + ".log"
		logPath := filepath.Join(logDir, logFileName)

		log.Info().Str("logfile", logging.CleanPath(logPath)).Msg("Logging to file")

		// Calculate number of enabled pages
		pageCount := 0
		for _, enabled := range conf.Pages {
			if enabled {
				pageCount++
			}
		}

		err := mfd.InitDevice(uint32(pageCount), edsm.ClearCache)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to initialize MFD device")
		}
		defer mfd.DeInitDevice()

		edreader.Start(conf)
		defer edreader.Stop()

		log.Info().Msg("Main event loop started")

		// Wait for either menu quit or OS signal or WinSparkle shutdown
		select {
		case <-mQuit.ClickedCh:
			// User clicked Quit
		case <-quitCh:
			// OS signal or WinSparkle shutdown received
		}
		systray.Quit()
	}()
}

func onExit() {
	log.Info().Msg("Application exiting, cleaning up")
	defer winsparkle.Cleanup()
}

// getIcon returns an icon as []byte. Replace with your own icon if desired.
func getIcon() []byte {
	return iconData
}

func showAboutDialog() {
	aboutText := fmt.Sprintf("EDxDC v%s\n\nSeamlessly reads Elite Dangerous journal data and presents real-time system, planet, cargo, and other information on your Saitek/Logitech X52 Pro Multi-Function Display.\n\n© Pellux Network", AppVersion)
	_ = zenity.Info(aboutText, zenity.Title("About EDxDC"))
}
