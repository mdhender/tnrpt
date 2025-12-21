// Copyright (c) 2025 Michael D Henderson. All rights reserved.

// Package docx implements a parser for Word DOCX files.
// It's more of an adapter than a parser.
package docx

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

type Docx struct {
	Source string
	Text   []byte
}

func ParsePath(path string, trimLeading, trimTrailing bool, quiet, verbose, debug bool) (*Docx, error) {
	path = filepath.Clean(path)
	text, err := parsePath(path)
	if err != nil || text == nil {
		return nil, err
	}
	return &Docx{Source: path, Text: trimOptions(text, trimLeading, trimTrailing)}, nil
}

func ParseReader(r *bytes.Reader, trimLeading, trimTrailing bool, quiet, verbose, debug bool) (*Docx, error) {
	text, err := parseReader(r)
	if err != nil || text == nil {
		return nil, err
	}
	return &Docx{Source: "<>", Text: trimOptions(text, trimLeading, trimTrailing)}, nil
}

// parse a reader that's loaded a .docx file. Returns the body text with
// whitespace preserved from <w:t xml:space="preserve"> plus tabs and
// line breaks. Injects a line-feed at the end of every paragraph.
func parseReader(r *bytes.Reader) ([]byte, error) {
	zr, err := zip.NewReader(r, r.Size())
	if err != nil {
		return nil, errors.Join(ErrNotAWordDocument, ErrorUncompressFailed, err)
	}
	for _, file := range zr.File {
		if file.Name == "word/document.xml" {
			rc, err := file.Open()
			if err != nil {
				return nil, errors.Join(ErrBadInput, err)
			}
			defer rc.Close()
			text, err := translateWordXML(rc)
			if err != nil {
				return nil, errors.Join(ErrBadInput, err)
			}
			return text, nil
		}
	}
	return nil, errors.Join(ErrNotAWordDocument, ErrWordXmlDocumentNotFound)
}

// parsePath is a helper function. It opens and parses a .docx file. Returns
// the body text with whitespace preserved from <w:t xml:space="preserve">
// plus tabs and line breaks.  Injects a line-feed at the end of every paragraph.
func parsePath(path string) ([]byte, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("open docx: %w", err)
	}
	defer zr.Close()
	for _, file := range zr.File {
		if file.Name == "word/document.xml" {
			rc, err := file.Open()
			if err != nil {
				return nil, fmt.Errorf("open document.xml: %w", err)
			}
			defer rc.Close()
			text, err := translateWordXML(rc)
			if err != nil {
				return nil, err
			}
			return text, nil
		}
	}
	return nil, errors.New("word/document.xml not found")
}

// translateWordXML actually walks the XML and builds the text.
func translateWordXML(r io.Reader) ([]byte, error) {
	dec := xml.NewDecoder(r)

	var (
		buf      = &bytes.Buffer{}
		preserve = false // xml:space="preserve" on current <w:t>
		inT      = false // we're inside a <w:t>

		// if we ever need to do something in a paragraph
		// seenTextInP = false // to know if paragraph has any text
		// inP         = false // we're inside a paragraph
	)

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("xml decode: %w", err)
		}

		switch se := tok.(type) {
		case xml.StartElement:
			local := se.Name.Local

			switch local {
			case "p":
				//inP, seenTextInP = true, false

			case "t":
				inT = true
				preserve = false
				// check attributes for xml:space="preserve"
				for _, a := range se.Attr {
					if (a.Name.Local == "space" || strings.HasSuffix(a.Name.Local, "space")) &&
						a.Value == "preserve" {
						preserve = true
						break
					}
				}

			case "tab":
				buf.WriteByte('\t')
				//seenTextInP = true

			case "br":
				buf.WriteByte('\n')
				//seenTextInP = true
			}

		case xml.EndElement:
			local := se.Name.Local
			switch local {
			case "t":
				inT = false
				preserve = false
			case "p":
				// inject a line-feed at the end of every paragraph
				buf.WriteByte('\n')
				//inP, seenTextInP = false, false
			}

		case xml.CharData:
			if inT {
				text := string(se)
				if preserve {
					buf.WriteString(text)
				} else {
					// most Word text here is already “normal”, but, just
					// in case, don’t trim aggressively — just write it.
					buf.WriteString(text)
				}
				//seenTextInP = true
			}
		}
	}

	return buf.Bytes(), nil
}

func trimOptions(text []byte, trimLeading, trimTrailing bool) []byte {
	if trimLeading == false && trimTrailing == false {
		return text
	}
	const asciiSpace = " \t\n\v\f\r"
	lines := bytes.Split(text, []byte{'\n'})
	if trimLeading && trimTrailing {
		for i, line := range lines {
			lines[i] = bytes.TrimSpace(line)
		}
	} else if trimLeading {
		for i, line := range lines {
			lines[i] = bytes.TrimLeft(line, asciiSpace)
		}
	} else {
		for i, line := range lines {
			lines[i] = bytes.TrimRight(line, asciiSpace)
		}
	}
	return bytes.Join(lines, []byte{'\n'})
}

type Error string

func (e Error) Error() string {
	return string(e)
}

const (
	ErrBadInput                = Error("bad input")
	ErrNotAWordDocument        = Error("not a word document")
	ErrorUncompressFailed      = Error("uncompress failed")
	ErrWordXmlDocumentNotFound = Error("word/document.xml not found")
)
