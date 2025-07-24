package edreader

import (
	"encoding/csv"
	"encoding/json"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
)

const FileCargo = "Cargo.json"

const (
	nameFileFolder        = "./names/"
	commodityNameFile     = nameFileFolder + "commodity.csv"
	rareCommodityNameFile = nameFileFolder + "rare_commodity.csv"
)

type Cargo struct {
	Count     int
	Inventory []CargoLine
}

type CargoLine struct {
	Name          string
	Count         int
	Stolen        int
	NameLocalized string `json:"Name_Localised"`
}

func (cl CargoLine) displayname() string {
	name := cl.Name
	displayName, ok := names[strings.ToLower(name)]
	if ok {
		name = displayName
	}
	return name
}

var (
	names        map[string]string
	currentCargo Cargo
)

func init() {
	log.Debug().Msg("Initializing cargo name map...")
	initNameMap()
}

func handleCargoFile(file string) {
	data, err := os.ReadFile(file)
	if err != nil {
		log.Debug().Str("file", file).Msg("No cargo file found")
		currentCargo = Cargo{}
		return
	}
	var cargo Cargo
	if err := json.Unmarshal(data, &cargo); err != nil {
		log.Error().Err(err).Str("file", file).Msg("Failed to unmarshal cargo file")
		currentCargo = Cargo{}
		return
	}
	currentCargo = cargo
}

func mapCommodities(data [][]string, symbolIdx, nameIdx int) {
	for _, com := range data[1:] {
		symbol := com[symbolIdx]
		symbol = strings.ToLower(symbol)
		name := com[nameIdx]
		names[symbol] = name
	}
}

func initNameMap() {
	commodity := readCsvFile(commodityNameFile)
	rareCommodity := readCsvFile(rareCommodityNameFile)

	names = make(map[string]string)

	// commodity.csv: symbol at 1, name at 3
	mapCommodities(commodity, 1, 3)
	// rare_commodity.csv: symbol at 1, name at 4
	mapCommodities(rareCommodity, 1, 4)
}

func readCsvFile(filename string) [][]string {
	csvfile, err := os.Open(filename)
	if err != nil {
		log.Fatal().Err(err).Str("file", filename).Msg("Failed to open CSV file")
		os.Exit(1)
	}
	defer csvfile.Close()
	csvreader := csv.NewReader(csvfile)
	records, err := csvreader.ReadAll()
	if err != nil {
		log.Fatal().Err(err).Str("file", filename).Msg("Failed to read CSV file")
		os.Exit(1)
	}
	return records
}
