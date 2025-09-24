package errors

import (
	"errors"
	"fmt"
)

// Base error types
var (
	// ErrCLINotFound is returned when the Claude CLI is not found
	ErrCLINotFound = errors.New("claude CLI not found")
	
	// ErrCLIConnection is returned when there's a connection error with the CLI
	ErrCLIConnection = errors.New("CLI connection error")
	
	// ErrProcess is returned when there's a subprocess error
	ErrProcess = errors.New("process error")
	
	// ErrJSONDecode is returned when JSON decoding fails
	ErrJSONDecode = errors.New("JSON decode error")
	
	// ErrMessageParse is returned when message parsing fails
	ErrMessageParse = errors.New("message parse error")
)

// CLINotFoundError indicates the Claude CLI binary was not found
type CLINotFoundError struct {
	Message string
}

func (e *CLINotFoundError) Error() string {
	return e.Message
}

func (e *CLINotFoundError) Is(target error) bool {
	return target == ErrCLINotFound
}

// CLIConnectionError indicates a connection problem with the CLI
type CLIConnectionError struct {
	Message string
	Cause   error
}

func (e *CLIConnectionError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *CLIConnectionError) Is(target error) bool {
	return target == ErrCLIConnection
}

func (e *CLIConnectionError) Unwrap() error {
	return e.Cause
}

// ProcessError indicates a subprocess error
type ProcessError struct {
	Message  string
	ExitCode int
	Stderr   string
}

func (e *ProcessError) Error() string {
	if e.Stderr != "" {
		return fmt.Sprintf("%s (exit code: %d): %s", e.Message, e.ExitCode, e.Stderr)
	}
	return fmt.Sprintf("%s (exit code: %d)", e.Message, e.ExitCode)
}

func (e *ProcessError) Is(target error) bool {
	return target == ErrProcess
}

// JSONDecodeError indicates a JSON decoding error
type JSONDecodeError struct {
	Message string
	Line    string
	Cause   error
}

func (e *JSONDecodeError) Error() string {
	if e.Line != "" {
		return fmt.Sprintf("%s: %v (line: %s)", e.Message, e.Cause, e.Line)
	}
	return fmt.Sprintf("%s: %v", e.Message, e.Cause)
}

func (e *JSONDecodeError) Is(target error) bool {
	return target == ErrJSONDecode
}

func (e *JSONDecodeError) Unwrap() error {
	return e.Cause
}

// MessageParseError indicates a message parsing error
type MessageParseError struct {
	Message string
	Data    interface{}
}

func (e *MessageParseError) Error() string {
	return fmt.Sprintf("%s: %+v", e.Message, e.Data)
}

func (e *MessageParseError) Is(target error) bool {
	return target == ErrMessageParse
}

// Helper functions
func NewCLINotFoundError(message string) error {
	return &CLINotFoundError{Message: message}
}

func NewCLIConnectionError(message string, cause error) error {
	return &CLIConnectionError{Message: message, Cause: cause}
}

func NewProcessError(message string, exitCode int, stderr string) error {
	return &ProcessError{Message: message, ExitCode: exitCode, Stderr: stderr}
}

func NewJSONDecodeError(message string, line string, cause error) error {
	return &JSONDecodeError{Message: message, Line: line, Cause: cause}
}

func NewMessageParseError(message string, data interface{}) error {
	return &MessageParseError{Message: message, Data: data}
}