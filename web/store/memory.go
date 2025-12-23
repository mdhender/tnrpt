// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package store

import (
	"sync"

	"github.com/mdhender/tnrpt/model"
)

// MemoryStore is a simple in-memory store for the Sprint 13 spike.
// It holds parsed turn report data and provides read-only access for handlers.
type MemoryStore struct {
	mu      sync.RWMutex
	reports []*model.ReportX
	units   []*model.UnitX
}

// New creates a new empty MemoryStore.
func New() *MemoryStore {
	return &MemoryStore{}
}

// AddReport adds a parsed report to the store.
func (s *MemoryStore) AddReport(rx *model.ReportX) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rx.ID = int64(len(s.reports) + 1)
	s.reports = append(s.reports, rx)

	for _, ux := range rx.Units {
		ux.ID = int64(len(s.units) + 1)
		ux.ReportXID = rx.ID
		s.units = append(s.units, ux)
	}
}

// Reports returns all reports in the store.
func (s *MemoryStore) Reports() []*model.ReportX {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*model.ReportX, len(s.reports))
	copy(result, s.reports)
	return result
}

// Units returns all units in the store.
func (s *MemoryStore) Units() []*model.UnitX {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*model.UnitX, len(s.units))
	copy(result, s.units)
	return result
}

// UnitsByTurn returns units filtered by turn number.
func (s *MemoryStore) UnitsByTurn(turnNo int) []*model.UnitX {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*model.UnitX
	for _, u := range s.units {
		if u.TurnNo == turnNo {
			result = append(result, u)
		}
	}
	return result
}

// UnitsByClan returns units filtered by clan ID.
// Clan ID is 4 digits (e.g., "0500"), where the last 3 digits identify the clan.
// A unit belongs to a clan if its base ID (first 4 digits) ends with those 3 digits.
// For example, clan "0500" owns units 0500, 1500, 3500e2, but not 0508 or 0506e1.
func (s *MemoryStore) UnitsByClan(clanID string) []*model.UnitX {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(clanID) < 3 {
		return nil
	}
	clanSuffix := clanID[len(clanID)-3:]

	var result []*model.UnitX
	for _, u := range s.units {
		if len(u.UnitID) >= 4 && u.UnitID[1:4] == clanSuffix {
			result = append(result, u)
		}
	}
	return result
}

// Stats returns basic statistics about the store.
func (s *MemoryStore) Stats() Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var actCount, stepCount int
	for _, u := range s.units {
		actCount += len(u.Acts)
		for _, a := range u.Acts {
			stepCount += len(a.Steps)
		}
	}

	return Stats{
		Reports: len(s.reports),
		Units:   len(s.units),
		Acts:    actCount,
		Steps:   stepCount,
	}
}

// Stats holds store statistics.
type Stats struct {
	Reports int
	Units   int
	Acts    int
	Steps   int
}
