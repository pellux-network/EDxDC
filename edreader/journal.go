package edreader

import (
	"bufio"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/buger/jsonparser"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// LocationType indicates where in a system the player is
type LocationType int

const (
	// LocationSystem means the player is somewhere in the system, not close to a body
	LocationSystem LocationType = iota
	// LocationPlanet means the player is close to a planetary body
	LocationPlanet
	// LocationLanded indicates the player has touched down
	LocationLanded
	// LocationDocked indicates the player has docked at a station (or outpost)
	LocationDocked
)

// Journalstate contains the player state based on the journal
type Journalstate struct {
	Location
	EDSMTarget
	Destination
	ArrivedAtFSDTarget     bool
	ArrivedAtFSDTargetTime time.Time
	LastFSDTargetSystem    string
	LastFSDTargetAddress   int64
	ShowSplashScreen       bool      // NEW: splash flag
	SplashScreenStartTime  time.Time // NEW: splash start time
}

// Location indicates the players current location in the game
type Location struct {
	Type LocationType

	SystemAddress int64
	StarSystem    string

	Body     string
	BodyID   int64
	BodyType string

	Latitude  float64
	Longitude float64
}

// EDSMTarget indicates a system targeted by the FSD drive for a jump
type EDSMTarget struct {
	Name                  string
	SystemAddress         int64
	RemainingJumpsInRoute int // NEW: for FSD Target jumps
}

// Destination holds the current destination info from Status.json
type Destination struct {
	SystemAddress int64
	BodyID        int64
	Name          string
}

const (
	systemaddress = "SystemAddress"
	bodyid        = "BodyID"
	starsystem    = "StarSystem"
	docked        = "Docked"
	body          = "Body"
	bodytype      = "BodyType"
	bodyname      = "BodyName"
	stationname   = "StationName"
	stationtype   = "StationType"
	latitude      = "Latitude"
	longitude     = "Longitude"
	name          = "Name"
)

// --- Fleet Carrier helpers ---

var (
	lastFCReceiveTextNameMu sync.Mutex
	lastFCReceiveTextName   = map[string]string{} // map[fcID]fcName
)

type parser struct {
	line []byte
}

func (p *parser) getString(field string) (string, bool) {
	str, err := jsonparser.GetString(p.line, field)
	if err != nil {
		return "", false
	}
	return str, true
}

func (p *parser) getInt(field string) (int64, bool) {
	num, err := jsonparser.GetInt(p.line, field)
	if err != nil {
		return 0, false
	}
	return num, true
}

func (p *parser) getBool(field string) (bool, bool) {
	b, err := jsonparser.GetBoolean(p.line, field)
	if err != nil {
		return false, false
	}
	return b, true
}

func (p *parser) getFloat(field string) (float64, bool) {
	f, err := jsonparser.GetFloat(p.line, field)
	if err != nil {
		return 0, false
	}
	return f, true
}

var printer = message.NewPrinter(language.English)

var (
	lastJournalFile     string
	lastJournalOffset   int64
	lastJournalState    Journalstate
	lastStatusFileSize  int64  // NEW: for status file change detection
	firstEnabledPageKey string // NEW: track first enabled page
	lastSystemAddress   int64  // NEW: track last system address for prefetching
)

func init() {
	lastJournalState.ShowSplashScreen = true
	lastJournalState.SplashScreenStartTime = time.Now()
}

// ExtractFleetCarrierNameID splits a string like "Stormcrow VZY-8XQ" into ("Stormcrow", "VZY-8XQ").
// Returns ("", "") if not a FC.
func ExtractFleetCarrierNameID(full string) (string, string) {
	parts := strings.Fields(full)
	if len(parts) < 2 {
		return "", ""
	}
	id := parts[len(parts)-1]
	if matched, _ := regexp.MatchString(`^[A-Z0-9]{3}-[A-Z0-9]{3}$`, id); !matched {
		return "", ""
	}
	name := strings.TrimSpace(strings.TrimSuffix(full, id))
	name = strings.TrimSpace(name)
	return name, id
}

// SaveFleetCarrierReceiveText remembers the last FC name for a given ID.
func SaveFleetCarrierReceiveText(from string) {
	parts := strings.Fields(from)
	if len(parts) < 2 {
		return
	}
	id := parts[len(parts)-1]
	if matched, _ := regexp.MatchString(`^[A-Z0-9]{3}-[A-Z0-9]{3}$`, id); !matched {
		return
	}
	name := strings.TrimSpace(strings.TrimSuffix(from, id))
	name = strings.TrimSpace(name)
	lastFCReceiveTextNameMu.Lock()
	lastFCReceiveTextName[id] = name
	lastFCReceiveTextNameMu.Unlock()
}

// GetLastFleetCarrierName returns the last seen FC name for a given ID, or "".
func GetLastFleetCarrierName(id string) string {
	lastFCReceiveTextNameMu.Lock()
	defer lastFCReceiveTextNameMu.Unlock()
	return lastFCReceiveTextName[id]
}

// Call this at startup after loading config, e.g. in main or Start()
func SetFirstEnabledPageKey(cfg map[string]bool) {
	for _, key := range []string{"destination", "location", "cargo"} {
		if cfg[key] {
			firstEnabledPageKey = key
			break
		}
	}
}

// handleJournalFile reads only new lines from the journal file since the last read.
// This is the one that actually loads the journal file
func handleJournalFile(filename string) {
	if filename == "" {
		return
	}
	file, err := os.Open(filename)
	if err != nil {
		log.Warnln("Error opening journal file ", filename, err)
		return
	}
	defer file.Close()

	var offset int64 = 0
	if filename == lastJournalFile {
		offset = lastJournalOffset
	}

	info, err := file.Stat()
	if err != nil {
		log.Warnln("Error stating journal file ", filename, err)
		return
	}

	// If file shrank (rotated), start from beginning
	if offset > info.Size() {
		offset = 0
	}

	_, err = file.Seek(offset, 0)
	if err != nil {
		log.Warnln("Error seeking journal file ", filename, err)
		return
	}

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	state := lastJournalState // Start from last known state
	linesRead := 0
	for scanner.Scan() {
		ParseJournalLine(scanner.Bytes(), &state)
		linesRead++
	}
	if linesRead > 0 {
		lastJournalState = state // Only update if new lines were read
	}

	// Save offset for next time
	pos, _ := file.Seek(0, 1)
	lastJournalFile = filename
	lastJournalOffset = pos

	checkSplashScreen()
}

// handleStatusFile reads Status.json for the current destination
func handleStatusFile(filename string) {
	if filename == "" {
		return
	}
	file, err := os.Open(filename)
	if err != nil {
		return
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return
	}
	if info.Size() == lastStatusFileSize {
		return
	}
	lastStatusFileSize = info.Size()

	data, err := io.ReadAll(file)
	if err != nil {
		return
	}

	destObj, _, _, err := jsonparser.Get(data, "Destination")
	if err == nil && len(destObj) > 0 {
		sysID, _ := jsonparser.GetInt(destObj, "System")
		bodyID, _ := jsonparser.GetInt(destObj, "Body")
		name := ""
		nameRaw, _ := jsonparser.GetString(destObj, "Name")
		if nameRaw != "" && !strings.HasPrefix(nameRaw, "$") {
			name = nameRaw
		} else {
			nameLoc, _ := jsonparser.GetString(destObj, "Name_Localised")
			if nameLoc != "" {
				name = nameLoc
			} else {
				name = nameRaw
			}
		}
		// --- Fleet Carrier: parse name/id if present ---
		fcName, fcID := ExtractFleetCarrierNameID(name)
		if fcID != "" {
			// Store for session (for TGT FC page)
			lastFCReceiveTextNameMu.Lock()
			if _, ok := lastFCReceiveTextName[fcID]; !ok {
				lastFCReceiveTextName[fcID] = fcName
			}
			lastFCReceiveTextNameMu.Unlock()
		}
		lastJournalState.Destination = Destination{
			SystemAddress: sysID,
			BodyID:        bodyID,
			Name:          name,
		}
	} else {
		lastJournalState.Destination = Destination{}
	}

	// After updating Destination, check for arrival
	checkArrival()

	checkSplashScreen()
}

// ParseJournalLine parses a single line of the journal and returns the new state after parsing.
func ParseJournalLine(line []byte, state *Journalstate) {
	re := regexp.MustCompile(`"event":"(\w*)"`)

	event := re.FindStringSubmatch(string(line))
	if len(event) < 2 {
		// Not a valid event line, skip
		return
	}
	p := parser{line}
	switch event[1] {
	case "Location":
		eLocation(p, state)
	case "SupercruiseEntry":
		eSupercruiseEntry(p, state)
	case "SupercruiseExit":
		eSupercruiseExit(p, state)
	case "FSDJump":
		eFSDJump(p, state)
	case "Touchdown":
		eTouchDown(p, state)
	case "Liftoff":
		eLiftoff(p, state)
	case "FSDTarget":
		eFSDTarget(p, state)
	case "ApproachBody":
		eApproachBody(p, state)
	case "ApproachSettlement":
		eApproachSettlement(p, state)
	case "Loadout":
		eLoadout(p)
	case "NavRouteClear":
		state.EDSMTarget = EDSMTarget{}
		state.LastFSDTargetSystem = ""
		state.LastFSDTargetAddress = 0
		state.ArrivedAtFSDTarget = false
		state.ArrivedAtFSDTargetTime = time.Time{}
	case "ReceiveText":
		eReceiveText(p)
	case "Docked":
		eDocked(p, state)
	}
}

func eLocation(p parser, state *Journalstate) {
	// clear current location completely
	state.Type = LocationSystem
	state.Location.SystemAddress, _ = p.getInt(systemaddress)
	state.StarSystem, _ = p.getString(starsystem)

	// Prefetch stations if system changed
	if state.Location.SystemAddress != lastSystemAddress && state.Location.SystemAddress != 0 {
		PrefetchStations(state.Location.SystemAddress)
		lastSystemAddress = state.Location.SystemAddress
	}

	bodyType, ok := p.getString(bodytype)

	if ok && bodyType == "Planet" {
		state.Location.BodyID, _ = p.getInt(bodyid)
		state.Location.Body, _ = p.getString(body)
		state.BodyType, _ = p.getString(bodytype)
		state.Type = LocationPlanet

		lat, ok := p.getFloat(latitude)
		if ok {
			state.Latitude = lat
			state.Longitude, _ = p.getFloat(longitude)
			state.Type = LocationLanded
		}
	}

	docked, _ := p.getBool(docked)
	if docked {
		state.Type = LocationDocked
	}
}

func eSupercruiseEntry(p parser, state *Journalstate) {
	state.Type = LocationSystem // don't throw away info
}

func eSupercruiseExit(p parser, state *Journalstate) {
	eLocation(p, state)
}

func eFSDJump(p parser, state *Journalstate) {
	eLocation(p, state)
	jumpSystem, _ := p.getString(starsystem)
	jumpAddress, _ := p.getInt(systemaddress)
	// Only trigger arrival if there was a valid FSD target (not zero/empty)
	if (state.LastFSDTargetAddress != 0 && jumpAddress == state.LastFSDTargetAddress) ||
		(state.LastFSDTargetSystem != "" && jumpSystem != "" && strings.EqualFold(jumpSystem, state.LastFSDTargetSystem)) {
		// Only if the previous FSD target was actually set
		if state.LastFSDTargetAddress != 0 || state.LastFSDTargetSystem != "" {
			state.ArrivedAtFSDTarget = true
			state.ArrivedAtFSDTargetTime = time.Now()
			// Reset jumps remaining on arrival
			state.EDSMTarget.RemainingJumpsInRoute = 0
			// Also clear EDSMTarget.SystemAddress and Name to reflect no target
			state.EDSMTarget.SystemAddress = 0
			state.EDSMTarget.Name = ""
			// Clear last FSD target so further jumps don't retrigger arrival
			state.LastFSDTargetSystem = ""
			state.LastFSDTargetAddress = 0
		}
	}
}

func eTouchDown(p parser, state *Journalstate) {
	state.Latitude, _ = p.getFloat(latitude)
	state.Longitude, _ = p.getFloat(longitude)
	state.Type = LocationLanded
}

func eLiftoff(p parser, state *Journalstate) {
	state.Type = LocationPlanet
}

func eFSDTarget(p parser, state *Journalstate) {
	systemAddress, _ := p.getInt(systemaddress)
	state.EDSMTarget.SystemAddress = systemAddress
	state.EDSMTarget.Name, _ = p.getString(name)
	jumps, ok := p.getInt("RemainingJumpsInRoute")
	if ok && systemAddress != 0 {
		state.EDSMTarget.RemainingJumpsInRoute = int(jumps)
	} else {
		state.EDSMTarget.RemainingJumpsInRoute = 0
	}
	// Save the last FSD target for arrival detection
	state.LastFSDTargetSystem = state.EDSMTarget.Name
	state.LastFSDTargetAddress = state.EDSMTarget.SystemAddress
	// New target: clear arrival state
	state.ArrivedAtFSDTarget = false
	state.ArrivedAtFSDTargetTime = time.Time{}
}

func eApproachBody(p parser, state *Journalstate) {
	state.Location.Body, _ = p.getString(body)
	state.Location.BodyID, _ = p.getInt(bodyid)

	state.Type = LocationPlanet
}

func eApproachSettlement(p parser, state *Journalstate) {
	state.Location.Body, _ = p.getString(bodyname)
	state.Location.BodyID, _ = p.getInt(bodyid)

	state.Type = LocationPlanet
}

func eLoadout(p parser) {
	capacity, ok := p.getInt("CargoCapacity")
	if ok {
		currentCargoCapacity = int(capacity)
	}
}

func eDocked(p parser, state *Journalstate) {
	stationName, _ := p.getString("StationName")
	stationType, _ := p.getString("StationType")
	systemAddress, _ := p.getInt("SystemAddress")
	systemName, _ := p.getString("StarSystem")

	state.Type = LocationDocked
	state.Location.Body = stationName
	state.Location.BodyID = 0
	state.Location.SystemAddress = systemAddress
	state.Location.StarSystem = systemName
	state.BodyType = "Station"

	// --- Fleet Carrier: store last FC name for this ID if docking at FC ---
	if stationType == "FleetCarrier" {
		// Try to get last seen FC name from ReceiveText, else leave as is
		// (Display logic will handle fallback)
	}
}

// --- Fleet Carrier: parse ReceiveText for FC name ---
func eReceiveText(p parser) {
	from, _ := p.getString("From")
	message, _ := p.getString("Message")
	channel, _ := p.getString("Channel")
	if channel == "npc" && strings.HasSuffix(message, "docking_granted;") {
		// Only store if looks like FC docking granted
		SaveFleetCarrierReceiveText(from)
	}
}

func checkArrival() {
	// Only clear arrival state if a new target is set, or N seconds have passed
	const arrivalTimeout = 10 * time.Second // <-- Change this value as desired
	if lastJournalState.ArrivedAtFSDTarget {
		if lastJournalState.EDSMTarget.SystemAddress != 0 || // new FSD target
			lastJournalState.Destination.SystemAddress != 0 || // local target
			(!lastJournalState.ArrivedAtFSDTargetTime.IsZero() &&
				time.Since(lastJournalState.ArrivedAtFSDTargetTime) > arrivalTimeout) {
			lastJournalState.ArrivedAtFSDTarget = false
			lastJournalState.ArrivedAtFSDTargetTime = time.Time{}
		}
	}
}

func checkSplashScreen() {
	const splashTimeout = 10 * time.Second
	if lastJournalState.ShowSplashScreen {
		timeoutPassed := time.Since(lastJournalState.SplashScreenStartTime) > splashTimeout

		firstPageReady := false
		switch firstEnabledPageKey {
		case "destination":
			firstPageReady = lastJournalState.Destination.SystemAddress != 0 ||
				lastJournalState.EDSMTarget.SystemAddress != 0 ||
				(lastJournalState.Type == LocationDocked && lastJournalState.Location.Body != "")
		case "location":
			firstPageReady = lastJournalState.Location.SystemAddress != 0
		case "cargo":
			firstPageReady = len(currentCargo.Inventory) > 0
		default:
			firstPageReady = true // fallback: don't block forever
		}

		if timeoutPassed && firstPageReady {
			lastJournalState.ShowSplashScreen = false
		}
	}
}
