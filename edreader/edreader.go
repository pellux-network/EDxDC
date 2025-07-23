package edreader

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/google/go-cmp/cmp"
	"github.com/pellux-network/EDxDC/conf"
	"github.com/pellux-network/EDxDC/edsm"
	"github.com/pellux-network/EDxDC/logging"
	"github.com/pellux-network/EDxDC/mfd"
	"github.com/rs/zerolog/log"
)

// PageKey is a string identifier for each page
type PageKey string

const (
	PageDestination PageKey = "destination"
	PageLocation    PageKey = "location"
	PageCargo       PageKey = "cargo"
)

// PageDef describes a page and how to render it
type PageDef struct {
	Key         PageKey
	DisplayName string
	Render      func(*mfd.Page, Journalstate)
}

// Registry of all possible pages
var PageRegistry = []PageDef{
	{
		Key:         PageDestination,
		DisplayName: "Destination",
		Render:      RenderDestinationPage, // This function contains the dynamic logic
	},
	{
		Key:         PageLocation,
		DisplayName: "Location",
		Render:      RenderLocationPage,
	},
	{
		Key:         PageCargo,
		DisplayName: "Cargo",
		Render:      RenderCargoPage,
	},
}

// Mfd is the MFD display structure to be used by this module.
var (
	Mfd     mfd.Display
	MfdLock = sync.RWMutex{}
	PrevMfd mfd.Display
	watcher *fsnotify.Watcher
	stopCh  chan struct{}
)

// Start starts the Elite Dangerous journal reader routine using fsnotify
func Start(cfg conf.Conf) {
	log.Info().Msg("Starting journal listener")
	journalfolder := cfg.ExpandJournalFolderPath()
	log.Debug().Str("journalfolder", logging.CleanPath(journalfolder)).Msg("Looking for journal files")

	// Set the first enabled page key for splash logic
	SetFirstEnabledPageKey(cfg.Pages)

	updateMFD(journalfolder, cfg)

	var err error
	watcher, err = fsnotify.NewWatcher()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create file watcher")
	}
	stopCh = make(chan struct{})

	// Watch the folder for new/changed files
	err = watcher.Add(journalfolder)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to add watcher")
	}

	// Prefetch stations for the initial system (if known)
	if lastJournalState.Location.SystemAddress != 0 {
		log.Debug().Int64("systemAddress", lastJournalState.Location.SystemAddress).Msg("Prefetching stations for initial system")
		PrefetchStations(lastJournalState.Location.SystemAddress)
	}

	go func() {
		defer watcher.Close()
		for {
			select {
			case event := <-watcher.Events:
				log.Trace().Str("event", event.String()).Msg("File system event received")
				if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) != 0 {
					updateMFD(journalfolder, cfg)
				}
			case err := <-watcher.Errors:
				log.Warn().Err(err).Msg("Watcher error")
			case <-stopCh:
				log.Info().Msg("Journal watcher stopped")
				return
			}
		}
	}()
}

func updateMFD(journalfolder string, cfg conf.Conf) {
	journalFile := findJournalFile(journalfolder)
	log.Debug().Str("journalFile", logging.CleanPath(journalFile)).Msg("Updating MFD")
	handleJournalFile(journalFile)
	handleStatusFile(filepath.Join(journalfolder, "Status.json"))
	handleModulesInfoFile(filepath.Join(journalfolder, FileModulesInfo))

	// Update in-memory cargo before rendering pages
	handleCargoFile(filepath.Join(journalfolder, FileCargo))

	// Build enabled pages
	var enabledPages []mfd.Page
	for _, pageDef := range PageRegistry {
		if cfg.Pages[string(pageDef.Key)] {
			page := mfd.NewPage()
			pageDef.Render(&page, lastJournalState)
			enabledPages = append(enabledPages, page)
		}
	}
	MfdLock.Lock()
	Mfd = mfd.Display{Pages: enabledPages}
	MfdLock.Unlock()

	swapMfd()
}

// Stop closes the watcher again
func Stop() {
	if stopCh != nil {
		close(stopCh)
	}
	if watcher != nil {
		watcher.Close()
	}
}

func findJournalFile(folder string) string {
	// Based on https://github.com/EDCD/EDMarketConnector/blob/693463d3a0dbe58a1f72b83fc09a44a4398af3fd/monitor.py#L264
	// because I don't have Odyssey myself
	files, _ := filepath.Glob(filepath.Join(folder, "Journal.*.*.log"))

	var lastFileTime time.Time
	var mostRecentJournal = ""

	for _, filename := range files {
		info, err := os.Stat(filename)
		if err != nil {
			continue
		}
		if mostRecentJournal == "" || info.ModTime().After(lastFileTime) {
			lastFileTime = info.ModTime()
			mostRecentJournal = filename
		}
	}
	log.Debug().Str("mostRecentJournal", logging.CleanPath(mostRecentJournal)).Msg("Found most recent journal file")
	return mostRecentJournal
}

func swapMfd() {
	MfdLock.RLock()
	defer MfdLock.RUnlock()
	eq := cmp.Equal(Mfd, PrevMfd)
	if !eq {
		mfd.Write(Mfd)
		PrevMfd = Mfd.Copy()
	}
}

// Prefetches station info for a system and caches it
func PrefetchStations(systemAddress int64) {
	go func() {
		_, _ = edsm.GetSystemStations(systemAddress)
	}()
}
