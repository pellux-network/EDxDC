package edreader

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	lcdformat "github.com/pbxx/goLCDFormat"
	"github.com/pellux-network/EDxDC/edsm"
	"github.com/pellux-network/EDxDC/mfd"
)

// Gets system body information from EDSM
func GetEDSMBodies(systemaddress int64) (*edsm.System, error) {
	sysinfo := <-edsm.GetSystemBodies(systemaddress)
	if sysinfo.Error != nil {
		return nil, fmt.Errorf("unable to fetch system information: %w", sysinfo.Error)
	}
	sys := sysinfo.S
	if sys.ID64 == 0 {
		return nil, fmt.Errorf("no EDSM data for system address %d", systemaddress)
	}
	return &sys, nil
}

// Gets system monetary values from EDSM
func GetEDSMSystemValue(systemaddress int64) (*edsm.System, error) {
	valueinfopromise := edsm.GetSystemValue(systemaddress)
	valinfo := <-valueinfopromise
	if valinfo.Error != nil {
		return nil, fmt.Errorf("unable to fetch system value: %w", valinfo.Error)
	}
	return &valinfo.S, nil
}

// Helper to render a station page
func RenderStationPage(page *mfd.Page, header string, st edsm.Station) {
	// Map allegiance to abbreviation
	abbr := map[string]string{
		"Federation":  "FED",
		"Empire":      "EMP",
		"Alliance":    "ALLI",
		"Independent": "IND",
	}
	titleCaser := cases.Title(language.English)
	alg := abbr[titleCaser.String(strings.ToLower(st.Allegiance))]
	if alg == "" {
		alg = st.Allegiance // fallback to raw if not mapped
	}
	page.Add(lcdformat.SpaceBetween(16, header, alg))
	page.Add(st.Name)
	page.Add(st.Type)
}

// Helper to render a Fleet Carrier page
func RenderFleetCarrierPage(page *mfd.Page, header, fcID, fcName string, systemAddress int64) {
	// Try to get EDSM station info for type (always "Fleet Carrier" but future-proof)
	stType := "Fleet Carrier"
	stations, err := edsm.GetSystemStations(systemAddress)
	if err == nil {
		for _, st := range stations {
			if strings.EqualFold(st.Name, fcID) {
				stType = st.Type
				break
			}
		}
	}
	page.Add(lcdformat.SpaceBetween(16, header, fcID))
	page.Add(fcName)
	page.Add(stType)
}

// Page rendering functions for MFD
func RenderLocationPage(page *mfd.Page, state Journalstate) {
	// --- Fleet Carrier: CURR FC page ---
	if state.Type == LocationDocked && state.Location.Body != "" && state.BodyType == "Station" {
		// Try to detect if docked at FC
		// state.Location.Body = StationName (FC ID), state.Location.SystemAddress
		stations, err := edsm.GetSystemStations(state.Location.SystemAddress)
		isFC := false
		if err == nil {
			for _, st := range stations {
				if strings.EqualFold(st.Name, state.Location.Body) && st.Type == "Fleet Carrier" {
					isFC = true
					break
				}
			}
		}
		if isFC || strings.HasPrefix(state.Location.Body, "FC") || len(state.Location.Body) == 7 {
			// Try to get last seen FC name from session
			fcID := state.Location.Body
			fcName := GetLastFleetCarrierName(fcID)
			if fcName == "" {
				fcName = "Unknown Fleet Carrier"
			}
			RenderFleetCarrierPage(page, "CURR FC", fcID, fcName, state.Location.SystemAddress)
			return
		}
		// ...existing code for normal stations...
		stations, err = edsm.GetSystemStations(state.Location.SystemAddress)
		if err == nil {
			for _, st := range stations {
				if strings.EqualFold(st.Name, state.Location.Body) {
					RenderStationPage(page, "CURR PORT", st)
					return
				}
			}
		}
		// fallback: show as body if not found as station
	}
	if state.Type == LocationPlanet || state.Type == LocationLanded {
		ApplyBodyPage(page, "CURR BODY", state.Location.SystemAddress, state.Location.BodyID, state.Location.Body)
	} else {
		ApplySystemPage(page, "CURR SYSTEM", state.Location.StarSystem, state.Location.SystemAddress, &state)
	}
}

func RenderDestinationPage(page *mfd.Page, state Journalstate) {
	if state.ShowSplashScreen {
		page.Add("################")
		page.Add("  EDxDC v1.0.0-beta  ")
		page.Add("################")
		return
	}
	if state.ArrivedAtFSDTarget {
		page.Add("################")
		page.Add("  You have arrived  ")
		page.Add("################")
		return
	}

	// Local destination in current system
	if state.Destination.SystemAddress != 0 &&
		state.Destination.SystemAddress == state.Location.SystemAddress &&
		state.Destination.Name != "" {

		// --- Fleet Carrier: TGT FC page ---
		// Try to parse FC name/id from destination name
		fcName, fcID := ExtractFleetCarrierNameID(state.Destination.Name)
		if fcID != "" {
			if fcName == "" {
				fcName = "Unknown Fleet Carrier"
			}
			RenderFleetCarrierPage(page, "TGT FC", fcID, fcName, state.Destination.SystemAddress)
			return
		}

		// Try to match station by name
		stations, err := edsm.GetSystemStations(state.Location.SystemAddress)
		if err == nil {
			for _, st := range stations {
				if strings.EqualFold(st.Name, state.Destination.Name) {
					RenderStationPage(page, "TGT PORT", st)
					return
				}
			}
		}

		// Fallback to body logic if BodyID is set
		if state.Destination.BodyID != 0 {
			sys, err := GetEDSMBodies(state.Location.SystemAddress)
			if err == nil {
				body := sys.BodyByID(state.Destination.BodyID)
				switch {
				case body.IsLandable:
					ApplyBodyPage(page, "TGT BODY", state.Location.SystemAddress, state.Destination.BodyID, state.Destination.Name)
					return
				default:
					page.Add(lcdformat.SpaceBetween(16, "TGT BODY", state.Destination.Name))
					if body.SubType != "" {
						page.Add(body.SubType)
					}
					return
				}
			}
		}
		// Fallback if EDSM fails or no BodyID
		page.Add(lcdformat.SpaceBetween(16, "TGT BODY", state.Destination.Name))
		return
	}

	// FSD target (next jump)
	if state.EDSMTarget.SystemAddress != 0 {
		ApplySystemPage(page, "NEXT JUMP", state.EDSMTarget.Name, state.EDSMTarget.SystemAddress, &state)
		return
	}

	page.Add(" No Destination ")
}

func RenderCargoPage(page *mfd.Page, _ Journalstate) {
	lines := []string{}
	// Cargo header
	lines = append(lines, fmt.Sprintf("CARGO: %04d/%04d", currentCargo.Count, ModulesInfoCargoCapacity()))
	// If currentCargo is nil (never loaded), show "No cargo data"
	if currentCargo.Inventory == nil {
		lines = append(lines, lcdformat.FillAround(16, "*", " NO CRGO DATA "))
		for _, line := range lines {
			page.Add(line)
		}
		return
	}

	if len(currentCargo.Inventory) == 0 {
		// If cargo inventory is empty, show "Cargo Hold Empty"
		lines = append(lines, lcdformat.FillAround(16, "*", " NO CARGO "))
		for _, line := range lines {
			page.Add(line)
		}
		return
	}
	sort.Slice(currentCargo.Inventory, func(i, j int) bool {
		a := currentCargo.Inventory[i]
		b := currentCargo.Inventory[j]
		return a.displayname() < b.displayname()
	})

	for _, line := range currentCargo.Inventory {
		lines = append(lines, lcdformat.SpaceBetween(16, line.displayname(), printer.Sprintf("%d", line.Count)))
	}
	// Add all pages in slice to the MFD
	for _, line := range lines {
		page.Add(line)
	}
}

// Page assembly functions for MFD
func ApplySystemPage(page *mfd.Page, header, systemname string, systemaddress int64, state *Journalstate) {
	// Initialize a slice to hold lines for the page
	lines := []string{}
	// Fetch system body information
	sys, err := GetEDSMBodies(systemaddress)
	if err != nil {
		log.Println("Error fetching EDSM data: ", err)
		return
	}

	// Fetch system monetary values
	values, err := GetEDSMSystemValue(systemaddress)
	if err != nil {
		log.Println("Error fetching EDSM system value: ", err)
		return
	}

	mainBody := sys.MainStar()
	// Separate the header (classification) and header display
	newHeader := header
	// Format the header based on the header title
	if header == "NEXT JUMP" || header == "CURR SYSTEM" {
		// Add FUEL indicator if star is scoopable
		if mainBody.IsScoopable {

			newHeader = lcdformat.SpaceBetween(16, header, "FUEL")
			lines = append(lines, newHeader)
		} else {
			lines = append(lines, header)
		}

	} else {
		lines = append(lines, header)
	}
	// Add the system name line to the page
	lines = append(lines, systemname)

	// Add the star class and remaining jumps
	// page.Add("Star: %s", mainBody.SubType)
	starTypeData := ParseStarTypeString(mainBody.SubType)
	jumps := ""

	if state != nil && header == "NEXT JUMP" {
		jumps = fmt.Sprintf("J:%d", state.EDSMTarget.RemainingJumpsInRoute)
	}
	lines = append(lines, lcdformat.SpaceBetween(16, fmt.Sprintf("CLS:%s", starTypeData.Class), jumps))
	// Add the main star information
	lines = append(lines, starTypeData.Desc)
	// Add system body count and estimated values

	lines = append(lines, lcdformat.SpaceBetween(16, "Bodies:", printer.Sprintf("%d", sys.BodyCount)))
	lines = append(lines, lcdformat.SpaceBetween(16, "Scan:", printer.Sprintf("%dcr", values.EstimatedValue)))
	lines = append(lines, lcdformat.SpaceBetween(16, "Map:", printer.Sprintf("%dcr", values.EstimatedValueMapped)))

	// Print valuable bodies if available
	if len(values.ValuableBodies) > 0 {
		lines = append(lines, lcdformat.FillAround(16, "*", " VAL BODIES "))
		for _, valbody := range values.ValuableBodies {
			bodyName := valbody.ShortName(*sys)
			crValue := printer.Sprintf("%dcr", valbody.ValueMax)
			// append the body name and value to the lines
			lines = append(lines, lcdformat.SpaceBetween(16, bodyName, crValue))
		}
	}

	// Evaluate presence of landable bodies and materials
	landables := []edsm.Body{}
	matLocations := map[string][]edsm.Body{}
	// Iterate through bodies to find landable bodies and their materials
	for _, body := range sys.Bodies {
		if body.IsLandable {
			landables = append(landables, body)
			for material := range body.Materials {
				bodiesWithMat, ok := matLocations[material]
				if !ok {
					bodiesWithMat = []edsm.Body{}
					matLocations[material] = bodiesWithMat
				}
				matLocations[material] = append(bodiesWithMat, body)
			}
		}
	}

	// Add prospecting information if landable bodies are present
	// if len(landables) > 0 {
	// 	lines = append(lines, lcdformat.FillAround(16, "*", " PROSPECT "))
	// 	materialList := []string{}

	// 	for mat := range matLocations {
	// 		materialList = append(materialList, mat)
	// 		bodies := matLocations[mat]
	// 		sort.Slice(bodies, func(i, j int) bool { return bodies[i].Materials[mat] > bodies[j].Materials[mat] })
	// 	}

	// 	// Sort materials by the number of bodies and then by material percentage
	// 	sort.Slice(materialList, func(i, j int) bool {
	// 		matA := materialList[i]
	// 		matB := materialList[j]
	// 		a := matLocations[matA]
	// 		b := matLocations[matB]
	// 		if len(a) == len(b) {
	// 			return a[0].Materials[matA] > b[0].Materials[matB]
	// 		}
	// 		return len(a) > len(b)

	// 	})
	// 	// Add material information to the page
	// 	for _, material := range materialList {
	// 		bodiesWithMat := matLocations[material]
	// 		lines = append(lines, fmt.Sprintf("%s %d", material, len(bodiesWithMat)))
	// 		b := bodiesWithMat[0]
	// 		// Add the body name (number usually) and material percentage
	// 		// matLine := lcdformat.SpaceBetween(16, b.ShortName(*sys), fmt.Sprintf("%.2f%%", float64(b.Materials[material])))
	// 		matLine := lcdformat.SpaceBetween(16, b.ShortName(*sys), fmt.Sprintf("%.2f%%%%", b.Materials[material]))
	// 		lines = append(lines, matLine)
	// 	}
	// } else {
	// 	return
	// }

	// Add all pages in slice to the MFD
	for _, line := range lines {
		page.Add(line)
	}
}

func ApplyBodyPage(page *mfd.Page, header string, systemAddress int64, bodyID int64, bodyName string) {
	lines := []string{}

	sys, err := GetEDSMBodies(systemAddress)
	if err != nil {
		log.Println("Error fetching EDSM data: ", err)
		lines = append(lines, lcdformat.FillAround(16, "*", " EDSM ERROR "))
		for _, line := range lines {
			page.Add(line)
		}
		return
	}

	body := sys.BodyByID(bodyID)
	if body.BodyID == 0 {
		lines = append(lines, lcdformat.FillAround(16, "*", " NO BODY DATA "))
		for _, line := range lines {
			page.Add(line)
		}
		return
	}
	lines = append(lines, lcdformat.SpaceBetween(16, header, fmt.Sprintf("%.2fG", body.Gravity)))
	lines = append(lines, bodyName)
	lines = append(lines, cases.Title(language.English).String(body.SubType))

	// add the planet materials
	lines = append(lines, lcdformat.FillAround(16, "*", " MATERIAL "))
	for _, m := range body.MaterialsSorted() {
		lines = append(lines, lcdformat.SpaceBetween(16, fmt.Sprintf("%5.2f%%%%", m.Percentage), m.Name))
	}
	for _, line := range lines {
		page.Add(line)
	}
}

type StarTypeData struct {
	Class string
	Desc  string
}

func ParseStarTypeString(starType string) StarTypeData {
	// Parse the star type string and return a formatted version
	// Example input: K (Yellow-Orange) Star
	splitST := strings.Split(starType, " ")
	class := splitST[0]
	description := strings.ReplaceAll(splitST[1], "(", "")
	description = strings.ReplaceAll(description, ")", "")
	description = fmt.Sprintf("%s %s", description, "Star")
	return StarTypeData{
		Class: class,
		Desc:  description,
	}
}
