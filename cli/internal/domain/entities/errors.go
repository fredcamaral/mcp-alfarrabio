// Package entities defines core data structures and business entities
// for the lerian-mcp-memory CLI application.
package entities

import "errors"

// Task validation and business rule errors
var (
	ErrInvalidStatusTransition = errors.New("invalid status transition")
	ErrInvalidEstimation       = errors.New("estimation must be non-negative")
	ErrInvalidActualTime       = errors.New("actual time must be non-negative")
	ErrEmptyContent            = errors.New("task content cannot be empty")
	ErrContentTooLong          = errors.New("task content exceeds maximum length")
	ErrInvalidRepository       = errors.New("repository cannot be empty")
	ErrInvalidUUID             = errors.New("invalid UUID format")
)
