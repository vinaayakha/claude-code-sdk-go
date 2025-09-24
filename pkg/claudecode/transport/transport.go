package transport

import (
	"context"
	"io"
)

// Transport defines the interface for communication with Claude Code
type Transport interface {
	// Connect establishes the connection
	Connect(ctx context.Context) error
	
	// Close terminates the connection
	Close() error
	
	// Write sends data to the transport
	Write(data []byte) error
	
	// Reader returns a reader for receiving data
	Reader() io.Reader
	
	// IsConnected returns true if the transport is connected
	IsConnected() bool
	
	// SetDebug enables/disables debug logging
	SetDebug(debug bool)
}