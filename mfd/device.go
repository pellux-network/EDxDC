package mfd

import (
	"fmt"

	"github.com/rs/zerolog/log"
)

// The current device handle
var device uintptr = 0

// The number of pages the device has been initialized with
var devicePages uint32 = 0

// Whether or not the device has been loaded yet
var loaded = false

// User-defined callback function for the soft button click
var buttonCallback func()

// The current text content to display
var currentDisplay Display

// The currently displayed page
var currentPage uint32

// Whether or not the current page is active
var pageActive bool

// The line index for each page
var currentLines []uint32

// InitDevice sets up the device for use
func InitDevice(pages uint32, softButtonCallback func()) error {
	log.Info().Uint32("pages", pages).Msg("Initializing device driver")
	if pages < 1 {
		return fmt.Errorf("pages parameter must be a positive integer")
	}
	devicePages = pages
	currentLines = make([]uint32, pages)

	currentDisplay = Display{Pages: make([]Page, pages)}

	buttonCallback = softButtonCallback

	log.Debug().Msg("Initializing driver connection")
	initialize()
	log.Debug().Msg("Registering device callbacks")
	registerDeviceCallback()
	log.Debug().Msg("Searching for device")
	enumerate()
	return nil
}

// DeInitDevice unregisters the device driver interaction. Should be called before terminating the program
func DeInitDevice() {
	deinitialize()
}

// UpdateDisplay updates the displayed text with a new set of pages.
func UpdateDisplay(display Display) error {

	if len(display.Pages) != int(devicePages) {
		return fmt.Errorf("provided display has %d pages. Must have %d", len(display.Pages), devicePages)
	}
	currentDisplay = display
	refreshDisplay()
	return nil
}

func initPages() {
	if !loaded {
		log.Debug().Msg("Device found.")
		log.Debug().Msg("Setting up page button callback")
		registerPageCallback(device)
		log.Debug().Msg("Setting up scroll button callback")
		registerSoftButtonCallback(device)
		log.Debug().Msg("Adding pages...")
		for p := uint32(0); p < devicePages; p++ {
			addPage(p, p == 0)
		}
		pageActive = true
		refreshDisplay()
		loaded = true
		log.Debug().Msg("Device init complete")
	}
}

func incrementLine() {
	page := currentDisplay.Pages[currentPage]
	line := currentLines[currentPage]
	pageLines := uint32(len(page.Lines))
	currentLines[currentPage] = min(line+1, pageLines)
	refreshDisplay()
}

func decrementLine() {
	line := currentLines[currentPage]
	if line > 0 {
		currentLines[currentPage] = line - 1
	}
	refreshDisplay()
}

// refreshDisplay refreshes the display to show the current values for page, line and display variables
func refreshDisplay() {
	if loaded && device > 0 && pageActive {
		log.Trace().Uint32("page", currentPage).Msg("Refreshing display")
		page := currentDisplay.Pages[currentPage]
		line := currentLines[currentPage]

		if line >= uint32(len(page.Lines)) {
			line = uint32(len(page.Lines)) - 1
		}

		for l := uint32(0); l < 3; l++ {
			shiftedLine := int(line + l)
			text := ""
			if shiftedLine < len(page.Lines) {
				text = page.Lines[shiftedLine]
			}
			setString(currentPage, l, text)
		}
	}

}

func min(a, b uint32) uint32 {
	if a < b {
		return a
	}
	return b
}
