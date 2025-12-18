// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package parser

import "os"

type Parser struct {
	input []byte
}

func New(path string) (*Parser, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return &Parser{
		input: data,
	}, nil
}

type Node struct{}

func (p *Parser) Parse() (*Node, error) {
	return &Node{}, nil
}
