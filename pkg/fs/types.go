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
    // ModeMask is the mask for the file permission bits
    ModeMask FileMode = 0777
    // ModeSetUID is the set-user-ID bit
    ModeSetUID FileMode = 04000
    // ModeSetGID is the set-group-ID bit
    ModeSetGID FileMode = 02000
    // ModeSticky is the sticky bit
    ModeSticky FileMode = 01000
)

// FileInfo contains information about a file.
type FileInfo struct {
    // Type is the file type
    Type FileType
    
    // Mode contains the permission bits
    Mode FileMode
    
    // Size is the file size in bytes
    Size int64
    
    // Uid is the user ID of the file's owner
    Uid uint32
    
    // Gid is the group ID of the file's group
    Gid uint32
    
    // Nlink is the number of hard links to the file
    Nlink uint32
    
    // Rdev is the device ID (if special file)
    Rdev uint64
    
    // BlockSize is the filesystem block size
    BlockSize uint32
    
    // Blocks is the number of blocks allocated
    Blocks uint64
    
    // AccessTime is the time of last access
    AccessTime time.Time
    
    // ModifyTime is the time of last modification
    ModifyTime time.Time
    
    // ChangeTime is the time of last status change
    ChangeTime time.Time
    
    // CreateTime is the time of creation (may not be available on all filesystems)
    CreateTime time.Time
}

// FileAttr contains attributes to set on a file.
// Only non-nil fields will be modified.
type FileAttr struct {
    // Mode is the permission bits to set
    Mode *FileMode
    
    // Size is the file size to set (truncate or extend)
    Size *int64
    
    // Uid is the user ID to set
    Uid *uint32
    
    // Gid is the group ID to set
    Gid *uint32
    
    // AccessTime is the access time to set
    AccessTime *time.Time
    
    // ModifyTime is the modification time to set
    ModifyTime *time.Time
}

// DirEntry represents an entry in a directory.
type DirEntry struct {
    // Name is the name of the entry
    Name string
    
    // FileId is a unique identifier for the file
    FileId uint64
    
    // Cookie is a position for resuming readdir operations
    Cookie int64
    
    // Attributes contains file attributes (if available)
    Attributes *FileInfo
}

// FSStat contains information about a filesystem.
type FSStat struct {
    // TotalBytes is the total size of the filesystem in bytes
    TotalBytes uint64
    
    // FreeBytes is the number of free bytes available
    FreeBytes uint64
    
    // AvailBytes is the number of bytes available to non-privileged users
    AvailBytes uint64
    
    // TotalFiles is the total number of file slots
    TotalFiles uint64
    
    // FreeFiles is the number of free file slots
    FreeFiles uint64
    
    // NameMaxLength is the maximum length of a file name
    NameMaxLength uint32
}