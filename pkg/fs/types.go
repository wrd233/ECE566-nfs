package fs

import (
    "time"
)

// FileType represents the type of a file.
type FileType uint32

const (
    // FileTypeRegular is a regular file
    FileTypeRegular FileType = iota
    // FileTypeDirectory is a directory
    FileTypeDirectory
    // FileTypeSymlink is a symbolic link
    FileTypeSymlink
    // FileTypeBlock is a block special device
    FileTypeBlock
    // FileTypeChar is a character special device
    FileTypeChar
    // FileTypeFIFO is a named pipe
    FileTypeFIFO
    // FileTypeSocket is a socket
    FileTypeSocket
)

// String returns a string representation of the file type
func (ft FileType) String() string {
    switch ft {
    case FileTypeRegular:
        return "regular"
    case FileTypeDirectory:
        return "directory"
    case FileTypeSymlink:
        return "symlink"
    case FileTypeBlock:
        return "block"
    case FileTypeChar:
        return "char"
    case FileTypeFIFO:
        return "fifo"
    case FileTypeSocket:
        return "socket"
    default:
        return "unknown"
    }
}

// FileMode represents the permission bits of a file.
type FileMode uint32

const (
    ModeMask FileMode = 0777
    ModeSetUID FileMode = 04000
    ModeSetGID FileMode = 02000
    ModeSticky FileMode = 01000
)

// FileInfo contains information about a file.
type FileInfo struct {
    Type FileType
    Mode FileMode
    Size int64
    Uid uint32
    Gid uint32
    Nlink uint32
    Rdev uint64
    BlockSize uint32
    Blocks uint64
    AccessTime time.Time
    ModifyTime time.Time
    ChangeTime time.Time
    CreateTime time.Time
}

// FileAttr contains attributes to set on a file.
// Only non-nil fields will be modified.
type FileAttr struct {
    Mode *FileMode
    Size *int64
    Uid *uint32
    Gid *uint32
    AccessTime *time.Time
    ModifyTime *time.Time
}

// DirEntry represents an entry in a directory.
type DirEntry struct {
    Name string
    FileId uint64
    Cookie int64
    Attributes *FileInfo
}

// FSStat contains information about a filesystem.
type FSStat struct {
    TotalBytes uint64
    FreeBytes uint64
    AvailBytes uint64
    TotalFiles uint64
    FreeFiles uint64
    NameMaxLength uint32
}