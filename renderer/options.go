// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package renderer

type Option func(r *Renderer) error

// WithExcludeUnits adds the units to the exclude units set
func WithExcludeUnits(units ...string) Option {
	return func(r *Renderer) error {
		for _, unit := range units {
			r.excludeUnits[unit] = true
			delete(r.includeUnits, unit)
		}
		return nil
	}
}

// WithIncludeUnits adds the units to the include units set.
// If this set is not empty, only units in the set will be parsed.
func WithIncludeUnits(units ...string) Option {
	return func(r *Renderer) error {
		for _, unit := range units {
			r.includeUnits[unit] = true
			delete(r.excludeUnits, unit)
		}
		return nil
	}
}
