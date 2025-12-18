// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package tnrpt

import (
	"github.com/maloquacious/semver"
)

var (
	version = semver.Version{
		Major: 1,
		Minor: 0,
		Patch: 0,
		Build: semver.Commit(),
	}
)

func Version() semver.Version {
	return version
}
