package edreader

import (
	"strings"
	"sync"
	"sync/atomic"
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
	flashTicker     *time.Ticker
	flashStop       chan struct{}
	flashIntervalMs int64 = 500
	currentConf     conf.Conf
	stateMu         sync.RWMutex
)

// isFlashOn returns the current flash phase based on wall clock time.
// Matches edxlc's animated_state: (milliseconds / interval) & 1 == 0
func isFlashOn() bool {
	interval := atomic.LoadInt64(&flashIntervalMs)
	if interval <= 0 {
		interval = 500
	}
	return (time.Now().UnixMilli() / interval) % 2 == 0
}

func startFlashLoop(interval int) {
	if interval <= 0 {
		interval = 500
	}
	atomic.StoreInt64(&flashIntervalMs, int64(interval))
	// Capture ticker and stop channel as local variables so the goroutine
	// closure references its OWN ticker/channel, not the package-level ones.
	// This prevents a race where the old goroutine stops the new ticker.
	ticker := time.NewTicker(time.Duration(interval) * time.Millisecond)
	stop := make(chan struct{})
	flashTicker = ticker
	flashStop = stop

	go func() {
		for {
			select {
			case <-ticker.C:
				stateMu.RLock()
				cfg := currentConf
				state := lastJournalState
				stateMu.RUnlock()

				if cfg.Lights.Enabled {
					UpdateLights(state)
				}
			case <-stop:
				ticker.Stop()
				return
			}
		}
	}()
}

func init() {
	startFlashLoop(500)
}

// SetLightConfig applies a new light configuration
func SetLightConfig(cfg conf.Conf) {
	stateMu.Lock()
	currentConf = cfg
	stateMu.Unlock()

	LoadBindings(cfg.BindsFile)

	interval := cfg.Lights.FlashInterval
	// Stop the old flash loop and start a new one with the updated interval
	if flashStop != nil {
		close(flashStop)
	}
	startFlashLoop(interval)
}

func UpdateLights(state Journalstate) {
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
	if state.ShipFlags&(FlagFsdCharging|FlagOverheating) == (FlagFsdCharging | FlagOverheating) {
		levels[AttrFrameShiftDrive] = StatusAlert
	} else if state.ShipFlags&FlagSupercruise != 0 && state.ShipFlags&FlagHardpointsDeployed != 0 {
		levels[AttrFrameShiftDrive] = StatusInactive
	} else if state.ShipFlags&(FlagCargoScoopDeployed|FlagMassLocked|FlagFsdCooldown|FlagHardpointsDeployed|FlagLandingGearDeployed) != 0 {
		levels[AttrFrameShiftDrive] = StatusBlocked
	} else if state.ShipFlags&FlagFsdCharging != 0 {
		levels[AttrFrameShiftDrive] = StatusActive
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

	// Resolve highest level per Light according to Priority: Alert > Blocked > Active > Inactive
	// Map attributes to their physical Light(s) via key bindings
	// (Matching edxlc game/controls.rs defaults)
	// Boost -> FireD, CargoScoop -> T1T2, ExternalLights -> T3T4
	// FSD -> T1T2/T3T4/T5T6, Hardpoints -> FireB, HeatSink -> T5T6
	// LandingGear -> T1T2/T3T4, NightVision -> PoV2
	lightMaxLevels := map[Light]StatusLevel{
		LightFire:     StatusInactive,
		LightClutch:   StatusInactive,
		LightFireA:    StatusInactive,
		LightFireB:    StatusInactive,
		LightFireD:    StatusInactive,
		LightFireE:    StatusInactive,
		LightPoV2:     StatusInactive,
		LightT1T2:     StatusInactive,
		LightT3T4:     StatusInactive,
		LightT5T6:     StatusInactive,
		LightThrottle: StatusInactive,
	}

	updateMax := func(l Light, level StatusLevel) {
		if level > lightMaxLevels[l] {
			lightMaxLevels[l] = level
		}
	}

	for attr, level := range levels {
		mappedLights := GetLightInputsForAttribute(attr)
		for _, l := range mappedLights {
			updateMax(l, level)
		}
	}

	// Determine theme, with fallback to default if section is missing
	theme := currentConf.Lights.Default
	if state.ShipFlags&FlagNightVision != 0 {
		if len(currentConf.Lights.NightVision.Inactive) > 0 {
			theme = currentConf.Lights.NightVision
		}
	} else if state.ShipFlags&FlagHardpointsDeployed != 0 && (state.ShipFlags&FlagSupercruise == 0) {
		if len(currentConf.Lights.HardpointsDeployed.Inactive) > 0 {
			theme = currentConf.Lights.HardpointsDeployed
		}
	}

	// Apply highest resolved levels to LEDs
	// Binary LEDs
	applyBinaryLed(mfd.LedFire, lightMaxLevels[LightFire], theme)
	applyBinaryLed(mfd.LedThrottle, lightMaxLevels[LightThrottle], theme)

	// RAG LEDs
	applyRagLed(mfd.LedFireARed, mfd.LedFireAGreen, lightMaxLevels[LightFireA], theme)
	applyRagLed(mfd.LedFireBRed, mfd.LedFireBGreen, lightMaxLevels[LightFireB], theme)
	applyRagLed(mfd.LedFireDRed, mfd.LedFireDGreen, lightMaxLevels[LightFireD], theme)
	applyRagLed(mfd.LedFireERed, mfd.LedFireEGreen, lightMaxLevels[LightFireE], theme)
	applyRagLed(mfd.LedT1T2Red, mfd.LedT1T2Green, lightMaxLevels[LightT1T2], theme)
	applyRagLed(mfd.LedT3T4Red, mfd.LedT3T4Green, lightMaxLevels[LightT3T4], theme)
	applyRagLed(mfd.LedT5T6Red, mfd.LedT5T6Green, lightMaxLevels[LightT5T6], theme)
	applyRagLed(mfd.LedPoV2Red, mfd.LedPoV2Green, lightMaxLevels[LightPoV2], theme)
	
	applyRagLed(mfd.LedClutchRed, mfd.LedClutchGreen, lightMaxLevels[LightClutch], theme)
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
		active = isFlashOn()
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
	// Handle dual-color flash modes: "color1-color2-flash" alternates between two colors
	// e.g. "red-amber-flash" flashes red ↔ amber, "green-red-flash" flashes green ↔ red
	// Also handle simple flash: "red-flash" flashes red ↔ off
	// And static modes: "red", "green", "amber", "off"

	isFlash := strings.HasSuffix(mode, "-flash")
	base := strings.TrimSuffix(mode, "-flash")

	if !isFlash {
		return staticColor(base)
	}

	// Check for dual-color flash patterns (color1-color2)
	dualColors := map[string][2]string{
		"red-amber":   {"red", "amber"},
		"red-green":   {"red", "green"},
		"amber-red":   {"amber", "red"},
		"amber-green": {"amber", "green"},
		"green-amber": {"green", "amber"},
		"green-red":   {"green", "red"},
	}

	if colors, ok := dualColors[base]; ok {
		if isFlashOn() {
			return staticColor(colors[0])
		}
		return staticColor(colors[1])
	}

	// Simple flash: color ↔ off
	if isFlashOn() {
		return staticColor(base)
	}
	return false, false
}

func staticColor(color string) (red, green bool) {
	switch color {
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
	if flashStop != nil {
		close(flashStop)
		flashStop = nil
	}
	allOff()
}

func allOff() {
	for i := uint32(0); i <= mfd.LedThrottle; i++ {
		mfd.UpdateLeds(i, false)
	}
}
