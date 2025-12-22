// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package parsers

type Config struct {
	autoEOL bool
	stripCR bool
}

type Option func(c *Config) error

func WithAutoEOL(flag bool) Option {
	return func(c *Config) error {
		c.autoEOL = flag
		return nil
	}
}

func WithStripCR(flag bool) Option {
	return func(c *Config) error {
		c.stripCR = flag
		return nil
	}
}
