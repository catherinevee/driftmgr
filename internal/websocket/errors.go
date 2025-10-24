package websocket

import "errors"

var (
	// ErrClientSendBufferFull is returned when the client's send buffer is full
	ErrClientSendBufferFull = errors.New("client send buffer is full")

	// ErrInvalidMessageType is returned when an invalid message type is provided
	ErrInvalidMessageType = errors.New("invalid message type")

	// ErrUnauthorized is returned when a client is not authorized for an operation
	ErrUnauthorized = errors.New("unauthorized")

	// ErrConnectionClosed is returned when trying to send to a closed connection
	ErrConnectionClosed = errors.New("connection is closed")

	// ErrInvalidUserID is returned when an invalid user ID is provided
	ErrInvalidUserID = errors.New("invalid user ID")
)
