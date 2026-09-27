//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

const (
	movefileReplaceExisting = 0x1
	movefileWriteThrough    = 0x8
)

var (
	kernel32       = syscall.NewLazyDLL("kernel32.dll")
	moveFileExProc = kernel32.NewProc("MoveFileExW")
)

// replaceFile performs Windows' atomic replacement operation. os.Rename does not
// replace an existing destination on Windows, while MoveFileEx does.
func replaceFile(source, destination string) error {
	src, err := syscall.UTF16PtrFromString(source)
	if err != nil {
		return err
	}
	dst, err := syscall.UTF16PtrFromString(destination)
	if err != nil {
		return err
	}
	r1, _, callErr := moveFileExProc.Call(
		uintptr(unsafe.Pointer(src)),
		uintptr(unsafe.Pointer(dst)),
		movefileReplaceExisting|movefileWriteThrough,
	)
	if r1 == 0 {
		return callErr
	}
	return nil
}
