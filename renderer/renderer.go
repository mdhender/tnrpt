// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package renderer

import (
	"log"
	"time"

	"github.com/mdhender/tnrpt/parsers/azul"
)

type Renderer struct {
	excludeUnits map[string]bool
	includeUnits map[string]bool
}

func New(options ...Option) (*Renderer, error) {
	r := &Renderer{
		excludeUnits: make(map[string]bool),
		includeUnits: make(map[string]bool),
	}
	for _, option := range options {
		err := option(r)
		if err != nil {
			return nil, err
		}
	}
	return r, nil
}

func (r *Renderer) Render(turn *azul.Turn_t, quiet, verbose, debug bool) error {
	started := time.Now()

	log.Printf("render: %04d-%02d: units %4d: in %v\n", turn.Year, turn.Month, len(turn.UnitMoves), time.Since(started))

	return nil
}
