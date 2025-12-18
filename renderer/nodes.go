// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package renderer

type Node interface {
	Kind() string
	Position() (line, column int)
}
type BaseNode struct {
	Type string `json:"kind,omitempty"`
}

func (b *BaseNode) Kind() string {
	return b.Type
}
func (b *BaseNode) Position() (line, column int) {
	return 0, 0
}

type RootNode struct {
	BaseNode
	Source  string `json:"source,omitempty"`
	Version string `json:"version,omitempty"`
}
