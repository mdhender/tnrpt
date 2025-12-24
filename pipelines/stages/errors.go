// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package stages

import "fmt"

// ErrWriteFile is returned when file I/O operations fail.
type ErrWriteFile struct {
	Op   string // mkdir, write, read
	Path string
	Err  error
}

func (e *ErrWriteFile) Error() string {
	return fmt.Sprintf("%s %s: %v", e.Op, e.Path, e.Err)
}

func (e *ErrWriteFile) Unwrap() error {
	return e.Err
}

// ErrDatabase is returned when database operations fail.
type ErrDatabase struct {
	Op  string
	Err error
}

func (e *ErrDatabase) Error() string {
	return fmt.Sprintf("database %s: %v", e.Op, e.Err)
}

func (e *ErrDatabase) Unwrap() error {
	return e.Err
}

// ErrDocxCorrupt is returned when DOCX extraction fails due to corruption.
type ErrDocxCorrupt struct {
	Path string
	Err  error
}

func (e *ErrDocxCorrupt) Error() string {
	return fmt.Sprintf("corrupt docx %s: %v", e.Path, e.Err)
}

func (e *ErrDocxCorrupt) Unwrap() error {
	return e.Err
}

// ErrParseSyntax is returned when the bistre parser encounters syntax errors.
type ErrParseSyntax struct {
	Line int
	Msg  string
}

func (e *ErrParseSyntax) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("parse syntax error at line %d: %s", e.Line, e.Msg)
	}
	return fmt.Sprintf("parse syntax error: %s", e.Msg)
}

// Error code constants for database storage.
const (
	ErrCodeWriteFile    = "WRITE_FILE"
	ErrCodeDatabase     = "DATABASE"
	ErrCodeDocxCorrupt  = "DOCX_CORRUPT"
	ErrCodeParseSyntax  = "PARSE_SYNTAX_ERROR"
	ErrCodeUnknown      = "UNKNOWN"
)

// ErrorCode returns the error code string for a given error.
func ErrorCode(err error) string {
	switch err.(type) {
	case *ErrWriteFile:
		return ErrCodeWriteFile
	case *ErrDatabase:
		return ErrCodeDatabase
	case *ErrDocxCorrupt:
		return ErrCodeDocxCorrupt
	case *ErrParseSyntax:
		return ErrCodeParseSyntax
	default:
		return ErrCodeUnknown
	}
}
