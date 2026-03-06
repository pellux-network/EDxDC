package edreader

import (
	"encoding/xml"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"golang.org/x/sys/windows/registry"
)

type Light uint32

const (
	LightFire     Light = 1
	LightFireA    Light = 2
	LightFireB    Light = 3
	LightFireD    Light = 4
	LightFireE    Light = 5
	LightPoV2     Light = 6
	LightT1T2     Light = 7
	LightT3T4     Light = 8
	LightT5T6     Light = 9
	LightThrottle Light = 10
	LightClutch   Light = 11
)

// BindingInput represents a single binding attribute in XML
type BindingInput struct {
	Device string `xml:"Device,attr"`
	Key    string `xml:"Key,attr"`
}

// ControlBinding represents a primary, secondary, and standard binding
type ControlBinding struct {
	Primary   BindingInput `xml:"Primary"`
	Secondary BindingInput `xml:"Secondary"`
	Binding   BindingInput `xml:"Binding"`
}

// ControlBindings models the Custom.4.2.binds XML root
type ControlBindings struct {
	XMLName                    xml.Name       `xml:"Root"`
	ExternalLights             ControlBinding `xml:"ShipSpotLightToggle"`
	CargoScoop                 ControlBinding `xml:"ToggleCargoScoop"`
	LandingGear                ControlBinding `xml:"LandingGearToggle"`
	Boost                      ControlBinding `xml:"UseBoostJuice"`
	HyperSuperCombination      ControlBinding `xml:"HyperSuperCombination"`
	Supercruise                ControlBinding `xml:"Supercruise"`
	Hyperspace                 ControlBinding `xml:"Hyperspace"`
	SilentRunning              ControlBinding `xml:"ToggleButtonUpInput"`
	HeatSink                   ControlBinding `xml:"DeployHeatSink"`
	Hardpoints                 ControlBinding `xml:"DeployHardpointToggle"`
	Throttle                   ControlBinding `xml:"ThrottleAxis"`
	NightVision                ControlBinding `xml:"NightVisionToggle"`
}

var currentBindings ControlBindings
var bindsLoaded bool

// LoadBindings loads the binds from the user's Local AppData
func LoadBindings(bindsPath string) {
	exp, _ := registry.ExpandString(bindsPath)

	if exp == "" {
		// Use default
		localAppData := os.Getenv("LOCALAPPDATA")
		exp = filepath.Join(localAppData, `Frontier Developments\Elite Dangerous\Options\Bindings\Custom.4.2.binds`)
	}

	data, err := os.ReadFile(exp)
	if err != nil {
		log.Warn().Err(err).Str("path", exp).Msg("Could not read bindings file, LED buttons may not map correctly to custom profiles")
		bindsLoaded = false
		return
	}

	err = xml.Unmarshal(data, &currentBindings)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to parse Elite Dangerous Bindings XML")
		bindsLoaded = false
		return
	}

	log.Info().Str("path", exp).Msg("Loaded game control bindings")
	bindsLoaded = true
}

// GetLightInputsForAttribute returns a list of Light LEDs mapped to a game attribute.
func GetLightInputsForAttribute(attr Attribute) []Light {
	if !bindsLoaded {
		// Fallback to defaults
		return GetDefaultLightInputsForAttribute(attr)
	}

	var binding ControlBinding

	switch attr {
	case AttrBoost:
		binding = currentBindings.Boost
	case AttrCargoScoop:
		binding = currentBindings.CargoScoop
	case AttrExternalLights:
		binding = currentBindings.ExternalLights
	case AttrFrameShiftDrive:
		// FSD incorporates 3 separate binds in Elite
		return append(
			append(getLightFromBinding(currentBindings.Hyperspace), getLightFromBinding(currentBindings.Supercruise)...),
			getLightFromBinding(currentBindings.HyperSuperCombination)...,
		)
	case AttrHardpoints:
		binding = currentBindings.Hardpoints
	case AttrHeatSink:
		binding = currentBindings.HeatSink
	case AttrLandingGear:
		binding = currentBindings.LandingGear
	case AttrNightVision:
		binding = currentBindings.NightVision
	case AttrSilentRunning:
		binding = currentBindings.SilentRunning
	case AttrThrottle:
		binding = currentBindings.Throttle
	default:
		return nil
	}

	return getLightFromBinding(binding)
}

func getLightFromBinding(binding ControlBinding) []Light {
	var lights []Light

	inputs := []BindingInput{binding.Primary, binding.Secondary, binding.Binding}
	for _, in := range inputs {
		if l := mapX52InputToLight(in.Device, in.Key); l != 0 {
			lights = append(lights, l)
		}
	}

	return lights
}

func mapX52InputToLight(device string, key string) Light {
	if device != "SaitekX52Pro" && device != "06A30762" {
		return 0
	}

	switch key {
	case "Joy_31":
		return LightClutch
	case "Joy_2":
		return LightFire
	case "Joy_3":
		return LightFireA
	case "Joy_4":
		return LightFireB
	case "Joy_7":
		return LightFireD
	case "Joy_8":
		return LightFireE
	case "Joy_22", "Joy_23", "Joy_21", "Joy_20":
		return LightPoV2
	case "Joy_9", "Joy_10":
		return LightT1T2
	case "Joy_11", "Joy_12":
		return LightT3T4
	case "Joy_13", "Joy_14":
		return LightT5T6
	case "Joy_ZAxis":
		return LightThrottle
	}
	return 0
}

func GetDefaultLightInputsForAttribute(attr Attribute) []Light {
	switch attr {
	case AttrBoost:
		return []Light{LightFireD}
	case AttrCargoScoop:
		return []Light{LightT1T2}
	case AttrExternalLights:
		return []Light{LightT3T4}
	case AttrFrameShiftDrive:
		return []Light{LightT1T2, LightT3T4, LightT5T6}
	case AttrHardpoints:
		return []Light{LightFireB}
	case AttrHeatSink:
		return []Light{LightT5T6}
	case AttrLandingGear:
		return []Light{LightT1T2, LightT3T4}
	case AttrNightVision:
		return []Light{LightPoV2}
	case AttrSilentRunning:
		return []Light{LightFireA}
	case AttrThrottle:
		return []Light{LightFireE, LightThrottle}
	}
	return nil
}
