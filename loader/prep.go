package loader

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
)

// prepare runs any special hacks for prepping the data before passing it onto
// the loader
func prepare(url, dir string) error {
	switch url {
	case "http://www.nyc.gov/html/dot/downloads/misc/siferry-gtfs.zip":
		return siFerry(dir)

	case "http://web.mta.info/developers/data/mnr/google_transit.zip":
		return mnr(dir)

	case "http://web.mta.info/developers/data/lirr/google_transit.zip":
		return lirr(dir)

	case "http://data.trilliumtransit.com/gtfs/path-nj-us/path-nj-us.zip":
		return njpath(dir)

	case "https://www.njtransit.com/mt/mt_servlet.srv?hdnPageAction=MTDevResourceDownloadTo&Category=rail":
		return njtrail(dir)

	case "http://web.mta.info/developers/data/nyct/subway/google_transit.zip":
		return mtasubway(dir)
	}

	return nil
}

type amods map[string]amod

type amod struct {
	// skip this row completely
	skip   bool
	newVal string
}

func modifyAgencies(dir string, mods amods) error {
	// appendedHeader is true if we need to append "agency_id", false
	// otherwise
	var appendedHeader bool
	var currentAgency string

	// Open up routes file as csv reader
	routeFile := path.Join(dir, "routes.txt")
	inFH, err := os.Open(routeFile)
	if err != nil {
		return err
	}
	r := csv.NewReader(inFH)
	r.LazyQuotes = true

	// Create an outgoing csv file for transformed data
	w, outFH := writecsvtmp(dir)
	defer outFH.Close()
	defer os.Remove(outFH.Name())

	// Read the existing header
	header, err := r.Read()
	if err != nil {
		return err
	}

	// Try to find agency id idx. If there is no agency_id header,
	// then add one.
	agencyIdx := maybeFind(header, "agency_id")
	if agencyIdx == -1 {
		appendedHeader = true
		header = append(header, "agency_id")
	}

	// Write header to the output file
	err = w.Write(header)
	if err != nil {
		return err
	}

	for {
		// Read until EOF or error
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if appendedHeader {
			// If there was an appended error, current value is blank
			currentAgency = ""
		} else {
			// Otherwise get the actual value
			currentAgency = rec[agencyIdx]
		}

		mod, ok := mods[currentAgency]
		// skip if requested
		if ok && mod.skip {
			continue
		}

		if ok && appendedHeader {
			// We found an empty "" => "newVal" with appended header, we must
			// append it. Basically if we're appending the header, the old
			// value must by definition be blank
			rec = append(rec, mod.newVal)

		} else if ok && !appendedHeader {
			// We found a "oldVal" => "newVal" with no need to append.
			rec[agencyIdx] = mod.newVal

		} else if !ok && appendedHeader {
			// Return an error if we didn't find a value but we needed
			// to append agency
			err = errors.New("no new value found and no existing agency")
			return err
		} else if !ok {
			// Skip if it wasn't found
			continue
		}

		// Write modified record to output
		err = w.Write(rec)
		if err != nil {
			return err
		}

	}

	// Flush and close output
	w.Flush()
	err = outFH.Close()
	if err != nil {
		return err
	}

	// Rename to official name in same dir
	err = os.Rename(outFH.Name(), routeFile)
	if err != nil {
		return err
	}

	// Success!
	return nil
}

// njpath modifies agency_id to PATH
func njpath(dir string) error {
	return modifyAgencies(dir,
		amods{
			"151": amod{false, "PATH"},
		},
	)
}

func lirr(dir string) error {
	return modifyAgencies(dir,
		amods{
			"": amod{false, "LI"},
		},
	)
}

// mnr modifies the agency id to standardize on simply "MTA MNR" rather than
// different numeric ids. Skip anything that isn't agency_id == 1.
func mnr(dir string) error {
	return modifyAgencies(dir,
		amods{
			"1": amod{false, "MTA MNR"},
		},
	)
}

// siFerry preps the Staten Island Ferry download by adding a direction id
// to files and fixing the header name of trip_headsign
func siFerry(dir string) error {
	var directionID string
	var err error

	// Get the incoming file
	tripFile := path.Join(dir, "trips.txt")
	inFH, err := os.Open(tripFile)
	if err != nil {
		return err
	}
	r := csv.NewReader(inFH)
	r.LazyQuotes = true

	// Create an outgoing csv file for transformed data
	w, outFH := writecsvtmp(dir)
	defer outFH.Close()
	defer os.Remove(outFH.Name())

	header, err := r.Read()
	if err != nil {
		return err
	}

	// Fix incorrectly named header
	for k, v := range header {
		if v == "headsign" {
			header[k] = "trip_headsign"
		}
	}

	headsignIdx := find(header, "trip_headsign")

	// Add a direction id header
	header = append(header, "direction_id")
	err = w.Write(header)
	if err != nil {
		return err
	}

	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}

		headsign := rec[headsignIdx]

		// SI Ferry has two destinations, add a 0/1 direction ID
		if headsign == "To Whitehall" {
			directionID = "0"
		} else {
			directionID = "1"
		}

		rec = append(rec, directionID)

		err = w.Write(rec)
		if err != nil {
			return err
		}

	}

	w.Flush()
	err = outFH.Close()
	if err != nil {
		return err
	}

	err = os.Rename(outFH.Name(), tripFile)
	if err != nil {
		return err
	}

	return nil
}

// njtrail adds route_color and route_text_color to NJT files
func njtrail(dir string) error {
	// Open up routes file as csv reader
	routeFile := path.Join(dir, "routes.txt")
	inFH, err := os.Open(routeFile)
	if err != nil {
		return err
	}
	r := csv.NewReader(inFH)
	r.LazyQuotes = true

	// Create an outgoing csv file for transformed data
	w, outFH := writecsvtmp(dir)
	defer outFH.Close()
	defer os.Remove(outFH.Name())

	// Read the existing header
	header, err := r.Read()
	if err != nil {
		return err
	}

	nameIdx := find(header, "route_long_name")
	rcIdx := find(header, "route_color")

	header = append(header, "route_text_color")
	rtIdx := len(header) - 1

	// Write header to the output file
	err = w.Write(header)
	if err != nil {
		return err
	}

	for {
		// Read until EOF or error
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Add one space to rec to allow for text color
		rec = append(rec, "")

		name := rec[nameIdx]
		switch name {
		case "Atlantic City Rail Line":
			rec[rcIdx] = "005DAB"
			rec[rtIdx] = "FFFFFF"

		case "Montclair-Boonton Line":
			rec[rcIdx] = "FAA634"
			rec[rtIdx] = "FFFFFF"

		case "Hudson-Bergen Light Rail", "Newark Light Rail", "Riverline Light Rail":
			rec[rcIdx] = "63F519"
			rec[rtIdx] = "000000"

		case "Main/Bergen County Line":
			rec[rcIdx] = "FFD006"
			rec[rtIdx] = "000000"

		case "Port Jervis Line":
			rec[rcIdx] = "BBCBE2"
			rec[rtIdx] = "000000"

		case "Morris & Essex Line", "Gladstone Branch":
			rec[rcIdx] = "00A850"
			rec[rtIdx] = "FFFFFF"

		case "Northeast Corridor", "Princeton Shuttle":
			rec[rcIdx] = "EE3A43"
			rec[rtIdx] = "FFFFFF"

		case "North Jersey Coast Line":
			rec[rcIdx] = "00A3E4"
			rec[rtIdx] = "000000"

		case "Pasack Valley Line":
			rec[rcIdx] = "A0218C"
			rec[rtIdx] = "FFFFFF"

		case "Raritan Valley Line":
			rec[rcIdx] = "FAA634"
			rec[rtIdx] = "000000"

		default:
			return fmt.Errorf("unrecognized NJT line: %s", name)

		}

		err = w.Write(rec)
		if err != nil {
			return err
		}
	}

	// Flush and close output
	w.Flush()
	err = outFH.Close()
	if err != nil {
		return err
	}

	// Rename to official name in same dir
	err = os.Rename(outFH.Name(), routeFile)
	if err != nil {
		return err
	}

	// Success!
	return nil
}

// mtasubway changes the color of 6X to same color as 6
func mtasubway(dir string) error {
	rw, err := newRewrite(dir, "routes.txt")
	if err != nil {
		return err
	}
	defer rw.clean()

	// Write header to the output file, unmodified
	err = rw.w.Write(rw.header)
	if err != nil {
		return err
	}

	routeIdx := find(rw.header, "route_id")
	routeColorIdx := find(rw.header, "route_color")

	for {
		// Read until EOF or error
		rec, err := rw.r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		routeID := rec[routeIdx]

		// 6X should be same color as 6
		if routeID == "6X" {
			rec[routeColorIdx] = "00933C"
		}

		err = rw.w.Write(rec)
		if err != nil {
			return err
		}
	}

	return rw.finish()
}
