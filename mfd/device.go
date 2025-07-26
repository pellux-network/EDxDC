package mfd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/rs/cors"
	log "github.com/sirupsen/logrus"
	"go.bug.st/serial"
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
	log.SetLevel(log.AllLevels[0])
	log.Infoln("Initializing device driver...")
	if pages < 1 {
		return fmt.Errorf("pages parameter must be a positive integer")
	}
	devicePages = pages
	currentLines = make([]uint32, pages)

	currentDisplay = Display{Pages: make([]Page, pages)}

	buttonCallback = softButtonCallback

	log.Debugln("Initializing driver connection")
	initialize()
	log.Debugln("Registering device callbacks")
	registerDeviceCallback()
	log.Debugln("Searching for device")
	enumerate()
	go StartControlWebhook()
	go StartSerialDevice()
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
		log.Debugln("Device found.")
		log.Debugln("Setting up page button callback")
		registerPageCallback(device)
		log.Debugln("Setting up scroll button callback")
		registerSoftButtonCallback(device)
		log.Debugln("Adding pages...")
		for p := uint32(0); p < devicePages; p++ {
			addPage(p, p == 0)
		}
		pageActive = true
		refreshDisplay()
		loaded = true
		log.Debugln("Device init complete")
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
		log.Debugln("Refreshing display")
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
		// send a rest API update with the current lines
		RESTPageUpdate(page.Lines)
		SerialPageUpdate(page.Lines)
	}

}

func min(a, b uint32) uint32 {
	if a < b {
		return a
	}
	return b
}

func StartControlWebhook() {
	mux := http.NewServeMux()
	mux.HandleFunc("/update", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.Error(w, "Only PUT allowed", http.StatusMethodNotAllowed)
			return
		}
		// get the requested page number from the header
		pageStr := r.Header.Get("Page")
		pageUint, err := strconv.ParseUint(pageStr, 10, 32)
		fmt.Printf("Page header: %s, parsed: %d\n", pageStr, pageUint)
		fmt.Printf("Current page: %d, Active Page: %t\n", currentPage, pageActive)
		if err != nil {
			http.Error(w, "Invalid Page header", http.StatusBadRequest)
			return
		}
		// set the current page
		currentPage = uint32(pageUint)
		pageActive = false
		fmt.Printf("Current page: %d, Active Page: %t\n", currentPage, pageActive)
		// refresh the display to show the new page
		refreshDisplay()
		// respond with OK
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/incpage", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.Error(w, "Only PUT allowed", http.StatusMethodNotAllowed)
			return
		}
		// trigger the page change callback
		currentPage = (currentPage + 1)
		onPageChange(device, currentPage, true, uintptr(0))
		// currentPage = (currentPage + 1) % devicePages
		Write(currentDisplay)
		fmt.Printf("Current page: %d, Active Page: %t\n", currentPage, pageActive)
		// refresh the display to show the new page
		refreshDisplay()

		// respond with OK
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/decpage", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.Error(w, "Only PUT allowed", http.StatusMethodNotAllowed)
			return
		}
		// trigger the page change callback
		currentPage = (currentPage - 1)
		onPageChange(device, currentPage, true, uintptr(0))
		// currentPage = (currentPage + 1) % devicePages
		Write(currentDisplay)
		fmt.Printf("Current page: %d, Active Page: %t\n", currentPage, pageActive)
		// refresh the display to show the new page
		refreshDisplay()

		// respond with OK
		w.WriteHeader(http.StatusOK)
	})
	fmt.Println("Dev control webhook listening at http://localhost:8551")

	handler := cors.AllowAll().Handler(mux)
	if err := http.ListenAndServe(":8551", handler); err != nil {
		log.Fatal(err)
	}
}

var serialPort serial.Port

func StartSerialDevice() {
	fmt.Println("Starting serial device...")
	mode := &serial.Mode{
		BaudRate: 115200,
		Parity:   serial.NoParity,
		DataBits: 8,
		StopBits: serial.OneStopBit,
	}
	port, err := serial.Open("COM3", mode)
	if err != nil {
		log.Fatal(err)
	}
	serialPort = port

	fmt.Println("Device opened successfully")
}

// RESTPageUpdate sends the current page lines to the REST API
// Currently for development purposes, it posts to a local server
func RESTPageUpdate(lines []string) {
	// Send a POST request to the testscreen API with the lines
	jsonData, err := json.Marshal(lines)
	if err != nil {
		log.Println("Error marshaling lines to JSON:", err)
		return
	}

	resp, err := http.Post("http://localhost:8080/lines", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Println("Error posting to example.com:", err)
		return
	}
	defer resp.Body.Close()
}

type SerialUpdate struct {
	Lines []string `json:"lines"`
}

func SerialPageUpdate(lines []string) {
	// Send a serial update with the current lines
	serialFrame := SerialUpdate{Lines: lines}
	jsonData, err := json.Marshal(serialFrame)
	log.Debugln("Serial update data:", string(jsonData))
	if err != nil {
		fmt.Println("Error marshaling lines to JSON:", err)
		return
	}

	_, err = serialPort.Write(jsonData)
	if err != nil {
		fmt.Println("Error writing to serial port:", err)
		return
	}
	fmt.Println("Serial port updated with lines")
}
