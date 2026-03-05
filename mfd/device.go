package mfd

import (
	"fmt"

	"github.com/rs/zerolog/log"
)

const (
	LedFire       uint32 = 0
	LedFireARed   uint32 = 1
	LedFireAGreen uint32 = 2
	LedFireBRed   uint32 = 3
	LedFireBGreen uint32 = 4
	LedFireDRed   uint32 = 5
	LedFireDGreen uint32 = 6
	LedFireERed   uint32 = 7
	LedFireEGreen uint32 = 8
	LedT1T2Red    uint32 = 9
	LedT1T2Green  uint32 = 10
	LedT3T4Red    uint32 = 11
	LedT3T4Green  uint32 = 12
	LedT5T6Red    uint32 = 13
	LedT5T6Green  uint32 = 14
	LedPoV2Red    uint32 = 15
	LedPoV2Green  uint32 = 16
	LedClutchRed  uint32 = 17
	LedClutchGreen uint32 = 18
	LedThrottle   uint32 = 19
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

// UpdateLeds sets the state of an LED on the device
func UpdateLeds(ledIdx uint32, active bool) {
	if loaded && device > 0 {
		val := uint32(0)
		if active {
			val = 1
		}
		// Update LEDs for all pages to ensure they stay consistent when switching pages
		for p := uint32(0); p < devicePages; p++ {
			setLed(p, ledIdx, val)
		}
	}
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
