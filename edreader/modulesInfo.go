package edreader

import (
	"encoding/json"
	"os"

	"github.com/pellux-network/EDxDC/logging"
	"github.com/rs/zerolog/log"
)

const FileModulesInfo = "ModulesInfo.json"

// ModulesInfo struct to load the ModulesInfo file saved by ED
type ModulesInfo struct {
	Modules []ModulesLine
}

// ModulesLine struct to load individual module in the ModuleInfo
type ModulesLine struct {
	Slot string
	Item string
}

var currentModules ModulesInfo
var currentCargoCapacity int

func handleModulesInfoFile(file string) {
	data, err := os.ReadFile(file)
	if err != nil {
		log.Warn().Err(err).Str("file", logging.CleanPath(file)).Msg("Failed to read ModulesInfo file")
		return
	}
	json.Unmarshal(data, &currentModules)
}

func ModulesInfoCargoCapacity() int {
	if currentCargoCapacity > 0 {
		return currentCargoCapacity
	}
	cargoCapacity := 0

	for i, line := range currentModules.Modules {
		log.Debug().
			Int("index", i).
			Str("slot", line.Slot).
			Str("item", line.Item).
			Msg("Processing module line")

		switch line.Item {
		case "int_cargorack_size1_class1":
			cargoCapacity += 2
			log.Debug().Int("added", 2).Int("total", cargoCapacity).Msg("Matched size1 cargo rack")
		case "int_cargorack_size2_class1":
			cargoCapacity += 4
			log.Debug().Int("added", 4).Int("total", cargoCapacity).Msg("Matched size2 cargo rack")
		case "int_cargorack_size3_class1":
			cargoCapacity += 8
			log.Debug().Int("added", 8).Int("total", cargoCapacity).Msg("Matched size3 cargo rack")
		case "int_cargorack_size4_class1":
			cargoCapacity += 16
			log.Debug().Int("added", 16).Int("total", cargoCapacity).Msg("Matched size4 cargo rack")
		case "int_cargorack_size5_class1":
			cargoCapacity += 32
			log.Debug().Int("added", 32).Int("total", cargoCapacity).Msg("Matched size5 cargo rack")
		case "int_cargorack_size6_class1":
			cargoCapacity += 64
			log.Debug().Int("added", 64).Int("total", cargoCapacity).Msg("Matched size6 cargo rack")
		case "int_cargorack_size7_class1":
			cargoCapacity += 128
			log.Debug().Int("added", 128).Int("total", cargoCapacity).Msg("Matched size7 cargo rack")
		case "int_largecargorack_size7_class1":
			cargoCapacity += 192
			log.Debug().Int("added", 192).Int("total", cargoCapacity).Msg("Matched large size7 cargo rack")
		case "int_cargorack_size8_class1":
			cargoCapacity += 256
			log.Debug().Int("added", 256).Int("total", cargoCapacity).Msg("Matched size8 cargo rack")
		case "int_largecargorack_size8_class1":
			cargoCapacity += 384
			log.Debug().Int("added", 384).Int("total", cargoCapacity).Msg("Matched large size8 cargo rack")
		default:
			log.Trace().Str("item", line.Item).Msg("Module not counted for cargo capacity")
		}
	}

	log.Info().Int("final_cargo_capacity", cargoCapacity).Msg("Calculated total cargo capacity")
	return cargoCapacity
}
