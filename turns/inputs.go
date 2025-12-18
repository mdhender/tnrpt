// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package turns

import (
	"fmt"
	"log"
	"path/filepath"
	"regexp"
	"strconv"
)

var (
	// turn report files have names that match the pattern YEAR-MONTH.CLAN_ID.report.txt.
	rxTurnReportFile = regexp.MustCompile(`^(\d{3,4})-(\d{2})\.(0\d{3})\.report\.txt$`)
)

// TurnReportFile_t represents a turn report file.
type TurnReportFile_t struct {
	Id   string // the id of the report file, taken from the file name.
	Path string // the path to the report file
	Turn struct {
		Id     string // the id of the turn, taken from the file name.
		Year   int    // the year of the turn
		Month  int    // the month of the turn
		ClanId string // the clan id of the turn
	}
}

// CollectInputs returns a []*TurnReportFile_t if it thinks the file is a turn report.
func CollectInputs(path, fileName string, quiet, verbose, debug bool) ([]*TurnReportFile_t, error) {
	matches := rxTurnReportFile.FindStringSubmatch(fileName)
	// length of matches is 4 because it includes the whole string in the slice
	if len(matches) != 4 {
		if debug {
			log.Printf("turns: input %q: does not match YYYY-MM.CLAN.report.txt", fileName)
		}
		return nil, nil
	}

	year, _ := strconv.Atoi(matches[1])
	month, _ := strconv.Atoi(matches[2])
	if year < 899 || year > 9999 || month < 1 || month > 12 {
		if debug {
			log.Printf("turns: input %q: invalid turn year or month\n", fileName)
		}
		return nil, nil
	}

	clanId := matches[3]

	rf := &TurnReportFile_t{
		Id:   fmt.Sprintf("%04d-%02d.%s", year, month, clanId),
		Path: filepath.Join(path, fileName),
	}
	rf.Turn.Id = fmt.Sprintf("%04d-%02d", year, month)
	rf.Turn.Year, rf.Turn.Month = year, month
	rf.Turn.ClanId = clanId

	return []*TurnReportFile_t{rf}, nil
}
