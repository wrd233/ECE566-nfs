// pkg/fs/errors.go
package fs

import (
    "errors"
    "fmt"
)

// Common filesystem errors that map to NFS error codes
var (
    ErrNotExist = errors.New("file does not exist")
    ErrExist = errors.New("file already exists")
	ErrPermission = errors.New("permission denied")
    ErrIO = errors.New("input/output error")
    ErrIsDir = errors.New("is a directory")
    ErrNotDir = errors.New("not a directory")
    ErrNotEmpty = errors.New("directory not empty")
    ErrInvalidName = errors.New("invalid name")
    ErrInvalidHandle = errors.New("invalid file handle")
    ErrNoSpace = errors.New("no space left on device")
    ErrReadOnly = errors.New("read-only filesystem")
    ErrBadCookie = errors.New("invalid directory cookie")
    ErrStale = errors.New("stale file handle")
    ErrNotSupported = errors.New("operation not supported")
)

// FSError represents a filesystem error with additional context.
type FSError struct {
    Op string
    Path string
    Err error
}

// Error implements the error interface.
func (e *FSError) Error() string {
    if e.Path == "" {
        return fmt.Sprintf("%s: %v", e.Op, e.Err)
    }
    return fmt.Sprintf("%s %s: %v", e.Op, e.Path, e.Err)
}

// Unwrap returns the underlying error.
func (e *FSError) Unwrap() error {
    return e.Err
}

// NewError creates a new FSError.
func NewError(op, path string, err error) error {
    return &FSError{
        Op:   op,
        Path: path,
        Err:  err,
    }
}