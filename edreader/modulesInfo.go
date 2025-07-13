package edreader

import (
	"encoding/json"
	"os"

	log "github.com/sirupsen/logrus"
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
var currentCargoCapacity int // NEW: updated from Loadout event

func handleModulesInfoFile(file string) {
	data, err := os.ReadFile(file)
	if err != nil {
		log.Errorln(err)
		return
	}

	json.Unmarshal(data, &currentModules)
}

func ModulesInfoCargoCapacity() int {
	if currentCargoCapacity > 0 {
		return currentCargoCapacity
	}
	cargoCapacity := 0

	for _, line := range currentModules.Modules {
		switch line.Item {
		case "int_cargorack_size1_class1":
			cargoCapacity += 2
		case "int_cargorack_size2_class1":
			cargoCapacity += 4
		case "int_cargorack_size3_class1":
			cargoCapacity += 8
		case "int_cargorack_size4_class1":
			cargoCapacity += 16
		case "int_cargorack_size5_class1":
			cargoCapacity += 32
		case "int_cargorack_size6_class1":
			cargoCapacity += 64
		case "int_cargorack_size7_class1":
			cargoCapacity += 128
		case "int_cargorack_size8_class1":
			cargoCapacity += 256
		}
	}

	return cargoCapacity

}
