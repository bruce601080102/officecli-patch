package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
)

// embeddedOfficeCLIPath installs the architecture-matched official binary in a
// content-addressed cache. Replacing a file in tools/ and rebuilding therefore
// produces a wrapper with a new, independent cache entry.
func embeddedOfficeCLIPath() (string, error) {
	digest := fmt.Sprintf("%x", sha256.Sum256(embeddedOfficeCLIBinary))
	cache := os.Getenv("OFFICECLI_PATCH_CACHE")
	if cache == "" {
		var err error
		cache, err = os.UserCacheDir()
		if err != nil {
			return "", fmt.Errorf("find user cache directory: %w", err)
		}
	}
	dir := filepath.Join(cache, "officecli-patch", "officecli", digest[:16])
	path := filepath.Join(dir, embeddedOfficeCLIFilename)
	if embeddedBinaryMatches(path, digest) {
		return path, nil
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("create OfficeCLI cache directory: %w", err)
	}
	tmp, err := os.CreateTemp(dir, embeddedOfficeCLIFilename+"-*.tmp")
	if err != nil {
		return "", fmt.Errorf("create OfficeCLI cache file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0700); err != nil {
		tmp.Close()
		return "", fmt.Errorf("mark OfficeCLI executable: %w", err)
	}
	if _, err := tmp.Write(embeddedOfficeCLIBinary); err != nil {
		tmp.Close()
		return "", fmt.Errorf("write embedded OfficeCLI: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return "", fmt.Errorf("finish embedded OfficeCLI: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		if embeddedBinaryMatches(path, digest) {
			return path, nil
		}
		return "", fmt.Errorf("install embedded OfficeCLI: %w", err)
	}
	return path, nil
}

func embeddedBinaryMatches(path, expected string) bool {
	b, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return fmt.Sprintf("%x", sha256.Sum256(b)) == expected
}
