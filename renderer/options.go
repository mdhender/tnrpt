// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package renderer

type Option func(p *Renderer) error

func WithAutoEOL(flag bool) Option {
	return func(p *Renderer) error {
		p.autoEOL = flag
		return nil
	}
}

// WithExcludeUnits adds the units to the exclude units set
func WithExcludeUnits(units ...string) Option {
	return func(p *Renderer) error {
		for _, unit := range units {
			p.excludeUnits[unit] = true
			delete(p.includeUnits, unit)
		}
		return nil
	}
}

// WithIncludeUnits adds the units to the include units set.
// If this set is not empty, only units in the set will be parsed.
func WithIncludeUnits(units ...string) Option {
	return func(p *Renderer) error {
		for _, unit := range units {
			p.includeUnits[unit] = true
			delete(p.excludeUnits, unit)
		}
		return nil
	}
}

func WithStripCR(flag bool) Option {
	return func(p *Renderer) error {
		p.stripCR = flag
		return nil
	}
}
