package analyzer

import (
	"path/filepath"

	"github.com/Joker-of-Gotham/gitdex/internal/git"
)

// largeFileExts contains file extensions typically requiring Git LFS.
var largeFileExts = map[string]bool{
	".psd": true, ".zip": true, ".tar": true, ".mp4": true, ".bin": true,
	".gz": true, ".bz2": true, ".7z": true, ".rar": true,
	".mov": true, ".avi": true, ".mkv": true, ".wav": true,
	".ai": true, ".sketch": true,
}

// DetectLFSNeed returns true if any of the given files has an extension that typically
// requires Git LFS (large binary files).
func DetectLFSNeed(files []git.FileStatus) bool {
	for _, f := range files {
		ext := filepath.Ext(f.Path)
		if largeFileExts[ext] {
			return true
		}
	}
	return false
}
