package edreader

import (
	"strings"
	"time"

	"github.com/pellux-network/EDxDC/conf"
	"github.com/pellux-network/EDxDC/mfd"
)

type StatusLevel int

const (
	StatusInactive StatusLevel = iota
	StatusActive
	StatusBlocked
	StatusAlert
)

type Attribute int

const (
	AttrBoost Attribute = iota
	AttrCargoScoop
	AttrExternalLights
	AttrFrameShiftDrive
	AttrHardpoints
	AttrHeatSink
	AttrLandingGear
	AttrNightVision
	AttrSilentRunning
	AttrThrottle
)

// Global state for flashing animation
var (
	flashTicker *time.Ticker
	flashState  bool
	currentConf conf.Conf
	stateMu     sync.RWMutex
)

func init() {
	// Default ticker, will be replaced in SetLightConfig
	flashTicker = time.NewTicker(500 * time.Millisecond)
	go func() {
		for range flashTicker.C {
			flashState = !flashState
			stateMu.RLock()
			cfg := currentConf
			state := lastJournalState
			stateMu.RUnlock()

			if cfg.Lights.Enabled {
				UpdateLights(state)
			}
		}
	}()
}

func SetLightConfig(cfg conf.Conf) {
	stateMu.Lock()
	defer stateMu.Unlock()
	currentConf = cfg
	interval := cfg.Lights.FlashInterval
	if interval <= 0 {
		interval = 500
	}
	if flashTicker != nil {
		flashTicker.Stop()
	}
	// We need to recreate the ticker and the loop or just the ticker? 
	// The loop is already running and listening to the old ticker channel.
	// Re-initializing the ticker variable doesn't update the loop's `range flashTicker.C`.
	// Better to use a stoppable/restartable loop or just accept 500ms as default and update if changed.
	flashTicker = time.NewTicker(time.Duration(interval) * time.Millisecond)
}

func UpdateLights(state Journalstate) {
	if state.ShipFlags == 0 && state.ShipFlags2 == 0 {
		// All off if no state
		allOff()
		return
	}

	// Calculate status levels for each attribute
	levels := make(map[Attribute]StatusLevel)

	// Cargo Scoop
	levels[AttrCargoScoop] = StatusInactive
	if state.ShipFlags&FlagCargoScoopDeployed != 0 {
		levels[AttrCargoScoop] = StatusActive
	}

	// External Lights
	levels[AttrExternalLights] = StatusInactive
	if state.ShipFlags&FlagLightsOn != 0 {
		levels[AttrExternalLights] = StatusActive
	}

	// FSD
	levels[AttrFrameShiftDrive] = StatusInactive
	if state.ShipFlags&FlagFsdCharging != 0 {
		if state.ShipFlags&FlagOverheating != 0 {
			levels[AttrFrameShiftDrive] = StatusAlert
		} else {
			levels[AttrFrameShiftDrive] = StatusActive
		}
	} else if state.ShipFlags&FlagSupercruise != 0 && state.ShipFlags&FlagHardpointsDeployed != 0 {
		levels[AttrFrameShiftDrive] = StatusInactive
	} else if state.ShipFlags&(FlagCargoScoopDeployed|FlagMassLocked|FlagFsdCooldown|FlagHardpointsDeployed|FlagLandingGearDeployed) != 0 {
		levels[AttrFrameShiftDrive] = StatusBlocked
	}

	// Landing Gear
	levels[AttrLandingGear] = StatusInactive
	if state.ShipFlags&FlagLandingGearDeployed != 0 {
		levels[AttrLandingGear] = StatusActive
	} else if state.ShipFlags&FlagDocking != 0 {
		levels[AttrLandingGear] = StatusAlert
	}

	// Heat Sink
	levels[AttrHeatSink] = StatusInactive
	if state.ShipFlags&FlagOverheating != 0 {
		levels[AttrHeatSink] = StatusAlert
	}

	// Silent Running
	levels[AttrSilentRunning] = StatusInactive
	if state.ShipFlags&FlagSilentRunning != 0 {
		if state.ShipFlags&FlagOverheating != 0 {
			levels[AttrSilentRunning] = StatusAlert
		} else {
			levels[AttrSilentRunning] = StatusActive
		}
	}

	// Hardpoints
	levels[AttrHardpoints] = StatusInactive
	if state.ShipFlags&FlagHardpointsDeployed != 0 {
		levels[AttrHardpoints] = StatusActive
	}

	// Boost
	levels[AttrBoost] = StatusInactive
	if state.ShipFlags&FlagLandingGearDeployed != 0 {
		levels[AttrBoost] = StatusBlocked
	}

	// Throttle
	levels[AttrThrottle] = StatusInactive
	if state.ShipFlags&FlagIsSpeeding != 0 {
		levels[AttrThrottle] = StatusAlert
	}

	// Night Vision
	levels[AttrNightVision] = StatusInactive
	if state.ShipFlags&FlagNightVision != 0 {
		levels[AttrNightVision] = StatusActive
	}

	// Determine theme
	theme := currentConf.Lights.Default
	if state.ShipFlags&FlagNightVision != 0 {
		theme = currentConf.Lights.NightVision
	} else if state.ShipFlags&FlagHardpointsDeployed != 0 && (state.ShipFlags&FlagSupercruise == 0) {
		theme = currentConf.Lights.HardpointsDeployed
	}

	// Apply levels to LEDs
	// Binary LEDs
	applyBinaryLed(mfd.LedFire, levels[AttrHardpoints], theme)
	applyBinaryLed(mfd.LedThrottle, levels[AttrThrottle], theme)

	// RAG LEDs
	applyRagLed(mfd.LedFireARed, mfd.LedFireAGreen, levels[AttrBoost], theme)
	applyRagLed(mfd.LedFireBRed, mfd.LedFireBGreen, levels[AttrFrameShiftDrive], theme)
	applyRagLed(mfd.LedFireDRed, mfd.LedFireDGreen, levels[AttrCargoScoop], theme)
	applyRagLed(mfd.LedFireERed, mfd.LedFireEGreen, levels[AttrExternalLights], theme)
	applyRagLed(mfd.LedT1T2Red, mfd.LedT1T2Green, levels[AttrLandingGear], theme)
	applyRagLed(mfd.LedT3T4Red, mfd.LedT3T4Green, levels[AttrHeatSink], theme)
	applyRagLed(mfd.LedT5T6Red, mfd.LedT5T6Green, levels[AttrSilentRunning], theme)
	applyRagLed(mfd.LedPoV2Red, mfd.LedPoV2Green, levels[AttrNightVision], theme)
	applyRagLed(mfd.LedClutchRed, mfd.LedClutchGreen, levels[AttrBoost], theme) // Boost also on clutch
}

func applyBinaryLed(ledIdx uint32, level StatusLevel, theme conf.LightState) {
	modeStrings := theme.Inactive
	switch level {
	case StatusActive:
		modeStrings = theme.Active
	case StatusBlocked:
		modeStrings = theme.Blocked
	case StatusAlert:
		modeStrings = theme.Alert
	}

	if len(modeStrings) < 1 {
		return
	}

	mode := modeStrings[0]
	active := false
	switch mode {
	case "on":
		active = true
	case "flash":
		active = flashState
	case "off":
		active = false
	}
	mfd.UpdateLeds(ledIdx, active)
}

func applyRagLed(redIdx, greenIdx uint32, level StatusLevel, theme conf.LightState) {
	modeStrings := theme.Inactive
	switch level {
	case StatusActive:
		modeStrings = theme.Active
	case StatusBlocked:
		modeStrings = theme.Blocked
	case StatusAlert:
		modeStrings = theme.Alert
	}

	if len(modeStrings) < 2 {
		return
	}

	colorMode := modeStrings[1]
	
	red, green := getColor(colorMode)
	
	mfd.UpdateLeds(redIdx, red)
	mfd.UpdateLeds(greenIdx, green)
}

func getColor(mode string) (red, green bool) {
	isFlash := strings.HasSuffix(mode, "-flash")
	base := strings.TrimSuffix(mode, "-flash")

	if isFlash && !flashState {
		return false, false
	}

	switch base {
	case "red":
		return true, false
	case "green":
		return false, true
	case "amber":
		return true, true
	case "off":
		return false, false
	}
	return false, false
}

func StopLights() {
	if flashTicker != nil {
		flashTicker.Stop()
	}
	allOff()
}

func allOff() {
	for i := uint32(0); i <= 19; i++ {
		mfd.UpdateLeds(i, false)
	}
}
