package client

import (
	"errors"
	"fmt"

	"github.com/example/nfsserver/pkg/api"
)

// Common error types
var (
	ErrNoServer       = errors.New("no server connection")
	ErrNotImplemented = errors.New("operation not implemented")
	ErrInvalidHandle  = errors.New("invalid file handle")
	ErrInvalidPath    = errors.New("invalid path")
	ErrPermission     = errors.New("permission denied")
	ErrNotExist       = errors.New("file does not exist")
	ErrIsDir          = errors.New("is a directory")
	ErrNotDir         = errors.New("not a directory")
	ErrTimeout        = errors.New("operation timed out")
)

// NFSError represents an error in an NFS operation
type NFSError struct {
	// Operation that failed
	Op string
	
	// NFS status code
	Status api.Status
	
	// Error message
	Message string
	
	// Underlying error
	Err error
}

// Error implements the error interface
func (e *NFSError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s failed: %s (%s) - %v", e.Op, e.Status, e.Message, e.Err)
	}
	return fmt.Sprintf("%s failed: %s (%s)", e.Op, e.Status, e.Message)
}

// Unwrap returns the underlying error
func (e *NFSError) Unwrap() error {
	return e.Err
}

// NewNFSError creates a new NFS error
func NewNFSError(op string, status api.Status, message string, err error) *NFSError {
	return &NFSError{
		Op:      op,
		Status:  status,
		Message: message,
		Err:     err,
	}
}

// StatusToError converts an NFS status to an error
func StatusToError(op string, status api.Status) error {
	if status == api.Status_OK {
		return nil
	}
	
	var message string
	var err error
	
	switch status {
	case api.Status_ERR_PERM:
		message = "not owner"
		err = ErrPermission
	case api.Status_ERR_NOENT:
		message = "no such file or directory"
		err = ErrNotExist
	case api.Status_ERR_IO:
		message = "I/O error"
	case api.Status_ERR_NXIO:
		message = "no such device or address"
	case api.Status_ERR_ACCES:
		message = "permission denied"
		err = ErrPermission
	case api.Status_ERR_EXIST:
		message = "file exists"
	case api.Status_ERR_NODEV:
		message = "no such device"
	case api.Status_ERR_NOTDIR:
		message = "not a directory"
		err = ErrNotDir
	case api.Status_ERR_ISDIR:
		message = "is a directory"
		err = ErrIsDir
	case api.Status_ERR_INVAL:
		message = "invalid argument"
	case api.Status_ERR_FBIG:
		message = "file too large"
	case api.Status_ERR_NOSPC:
		message = "no space left on device"
	case api.Status_ERR_ROFS:
		message = "read-only file system"
	case api.Status_ERR_NAMETOOLONG:
		message = "filename too long"
	case api.Status_ERR_NOTEMPTY:
		message = "directory not empty"
	case api.Status_ERR_DQUOT:
		message = "disk quota exceeded"
	case api.Status_ERR_STALE:
		message = "stale file handle"
		err = ErrInvalidHandle
	case api.Status_ERR_BADHANDLE:
		message = "illegal NFS file handle"
		err = ErrInvalidHandle
	case api.Status_ERR_NOT_SYNC:
		message = "update synchronization mismatch"
	case api.Status_ERR_BAD_COOKIE:
		message = "READDIR cookie is stale"
	case api.Status_ERR_NOTSUPP:
		message = "operation not supported"
		err = ErrNotImplemented
	case api.Status_ERR_TOOSMALL:
		message = "buffer or request is too small"
	case api.Status_ERR_SERVERFAULT:
		message = "server fault"
	case api.Status_ERR_BADTYPE:
		message = "type not supported"
	case api.Status_ERR_JUKEBOX:
		message = "operation requires human intervention"
	default:
		message = "unknown error"
	}
	
	return NewNFSError(op, status, message, err)
}
