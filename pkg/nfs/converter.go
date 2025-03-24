package nfs

import (
	"time"
	"log"

	"github.com/example/nfsserver/pkg/api"
	"github.com/example/nfsserver/pkg/fs"
)

// FSInfoToProtoAttributes converts filesystem FileInfo to NFS FileAttributes
func FSInfoToProtoAttributes(info fs.FileInfo) *api.FileAttributes {
	// Create file time structure for access time
	atime := &api.FileTime{
		Seconds: info.AccessTime.Unix(),
		Nano:    int32(info.AccessTime.Nanosecond()),
	}

	// Create file time structure for modification time
	mtime := &api.FileTime{
		Seconds: info.ModifyTime.Unix(),
		Nano:    int32(info.ModifyTime.Nanosecond()),
	}

	// Create file time structure for change time
	ctime := &api.FileTime{
		Seconds: info.ChangeTime.Unix(),
		Nano:    int32(info.ChangeTime.Nanosecond()),
	}

	// Convert file type
	var fileType api.FileType
	switch info.Type {
	case fs.FileTypeRegular:
		fileType = api.FileType_REGULAR
	case fs.FileTypeDirectory:
		fileType = api.FileType_DIRECTORY
	case fs.FileTypeSymlink:
		fileType = api.FileType_SYMLINK
	case fs.FileTypeBlock:
		fileType = api.FileType_BLOCK
	case fs.FileTypeChar:
		fileType = api.FileType_CHAR
	case fs.FileTypeFIFO:
		fileType = api.FileType_FIFO
	case fs.FileTypeSocket:
		fileType = api.FileType_SOCKET
	default:
		fileType = api.FileType_REGULAR
	}

	// Create and return attributes
	return &api.FileAttributes{
		Type:      fileType,
		Mode:      uint32(info.Mode),
		Nlink:     info.Nlink,
		Uid:       info.Uid,
		Gid:       info.Gid,
		Size:      uint64(info.Size),
		Used:      info.Blocks * 512, // Block size is typically 512 bytes
		RdevMajor: uint32(info.Rdev >> 32),
		RdevMinor: uint32(info.Rdev & 0xFFFFFFFF),
		Fileid:    uint64(info.Size),  // This should actually come from inode
		Atime:     atime,
		Mtime:     mtime,
		Ctime:     ctime,
		Blksize:   info.BlockSize,
		Blocks:    uint32(info.Blocks),
	}
}

// ProtoAttributesToFSAttr converts NFS FileAttributes to filesystem FileAttr
func ProtoAttributesToFSAttr(attr *api.FileAttributes) fs.FileAttr {
	result := fs.FileAttr{}

	// Only set the fields that are present in the request
	if attr.Mode != 0 {
		mode := fs.FileMode(attr.Mode)
		result.Mode = &mode
	}

	if attr.Size != 0 {
		size := int64(attr.Size)
		result.Size = &size
	}

	if attr.Uid != 0 {
		uid := attr.Uid
		result.Uid = &uid
	}

	if attr.Gid != 0 {
		gid := attr.Gid
		result.Gid = &gid
	}

	// Handle timestamps
	if attr.Atime != nil && (attr.Atime.Seconds != 0 || attr.Atime.Nano != 0) {
		atime := time.Unix(attr.Atime.Seconds, int64(attr.Atime.Nano))
		result.AccessTime = &atime
	}

	if attr.Mtime != nil && (attr.Mtime.Seconds != 0 || attr.Mtime.Nano != 0) {
		mtime := time.Unix(attr.Mtime.Seconds, int64(attr.Mtime.Nano))
		result.ModifyTime = &mtime
	}

	return result
}

// ProtoCredsToFSCreds converts NFS Credentials to filesystem Credentials
func ProtoCredsToFSCreds(creds *api.Credentials) fs.Credentials {
	if creds == nil {
		// Default to root if no credentials provided
		return fs.Credentials{
			UID:    0,
			GID:    0,
			Groups: []uint32{0},
		}
	}

	return fs.Credentials{
		UID:    creds.Uid,
		GID:    creds.Gid,
		Groups: creds.Groups,
	}
}