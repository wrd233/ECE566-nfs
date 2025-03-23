// Package nfs provides the NFS protocol implementation
package nfs

import (
	"errors"
	"fmt"
	"log"
	"os"
	"syscall"

	"github.com/example/nfsserver/pkg/api"
	"github.com/example/nfsserver/pkg/fs"
)

// MapErrorToStatus converts a Go error to an NFS status code
func MapErrorToStatus(err error) api.Status {
	if err == nil {
		return api.Status_OK
	}

	// Map filesystem errors to NFS status codes
	if errors.Is(err, fs.ErrNotExist) {
		return api.Status_ERR_NOENT
	} else if errors.Is(err, fs.ErrPermission) {
		return api.Status_ERR_ACCES
	} else if errors.Is(err, fs.ErrExist) {
		return api.Status_ERR_EXIST
	} else if errors.Is(err, fs.ErrIO) {
		return api.Status_ERR_IO
	} else if errors.Is(err, fs.ErrIsDir) {
		return api.Status_ERR_ISDIR
	} else if errors.Is(err, fs.ErrNotDir) {
		return api.Status_ERR_NOTDIR
	} else if errors.Is(err, fs.ErrInvalidName) {
		return api.Status_ERR_INVAL
	} else if errors.Is(err, fs.ErrInvalidHandle) {
		return api.Status_ERR_BADHANDLE
	} else if errors.Is(err, fs.ErrNoSpace) {
		return api.Status_ERR_NOSPC
	} else if errors.Is(err, fs.ErrReadOnly) {
		return api.Status_ERR_ROFS
	} else if errors.Is(err, fs.ErrBadCookie) {
		return api.Status_ERR_BAD_COOKIE
	} else if errors.Is(err, fs.ErrStale) {
		return api.Status_ERR_STALE
	} else if errors.Is(err, fs.ErrNotSupported) {
		return api.Status_ERR_NOTSUPP
	} else if errors.Is(err, fs.ErrNotEmpty) {
		return api.Status_ERR_NOTEMPTY
	}

	// Map standard Go errors
	if errors.Is(err, os.ErrPermission) {
		return api.Status_ERR_PERM
	} else if errors.Is(err, os.ErrNotExist) {
		return api.Status_ERR_NOENT
	} else if errors.Is(err, os.ErrExist) {
		return api.Status_ERR_EXIST
	}

	// Map syscall errors
	var errno syscall.Errno
	if errors.As(err, &errno) {
		switch errno {
		case syscall.EPERM:
			return api.Status_ERR_PERM
		case syscall.ENOENT:
			return api.Status_ERR_NOENT
		case syscall.EIO:
			return api.Status_ERR_IO
		case syscall.ENXIO:
			return api.Status_ERR_NXIO
		case syscall.EACCES:
			return api.Status_ERR_ACCES
		case syscall.EEXIST:
			return api.Status_ERR_EXIST
		case syscall.ENODEV:
			return api.Status_ERR_NODEV
		case syscall.ENOTDIR:
			return api.Status_ERR_NOTDIR
		case syscall.EISDIR:
			return api.Status_ERR_ISDIR
		case syscall.EINVAL:
			return api.Status_ERR_INVAL
		case syscall.EFBIG:
			return api.Status_ERR_FBIG
		case syscall.ENOSPC:
			return api.Status_ERR_NOSPC
		case syscall.EROFS:
			return api.Status_ERR_ROFS
		case syscall.ENAMETOOLONG:
			return api.Status_ERR_NAMETOOLONG
		case syscall.ENOTEMPTY:
			return api.Status_ERR_NOTEMPTY
		}
	}

	// Default for unrecognized errors
	LogUnknownError(err)
	return api.Status_ERR_IO
}

// LogUnknownError logs detailed information about unrecognized errors
func LogUnknownError(err error) {
	log.Printf("Unknown error type: %T, message: %v", err, err)
}

// LogRequest logs information about a received NFS request
func LogRequest(op string, reqID string, clientAddr string) {
	log.Printf("NFS request: %s, ID: %s, Client: %s", op, reqID, clientAddr)
}

// LogResponse logs information about an NFS response
func LogResponse(op string, reqID string, status api.Status, duration string) {
	log.Printf("NFS response: %s, ID: %s, Status: %s, Duration: %s", 
		op, reqID, status.String(), duration)
}

// LogError logs an error with its context
func LogError(op string, reqID string, err error) {
	log.Printf("NFS error: %s, ID: %s, Error: %v", op, reqID, err)
}

// NFSError represents an error with NFS status code
type NFSError struct {
	Status  api.Status // NFS status code
	Message string     // Error description
	Cause   error      // Underlying error
}

// Error implements the error interface
func (e *NFSError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (underlying: %v)", e.Status.String(), e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Status.String(), e.Message)
}

// Unwrap returns the underlying error
func (e *NFSError) Unwrap() error {
	return e.Cause
}

// NewNFSError creates a new NFSError
func NewNFSError(status api.Status, message string, cause error) *NFSError {
	return &NFSError{
		Status:  status,
		Message: message,
		Cause:   cause,
	}
}