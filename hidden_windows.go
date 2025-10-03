//go:build windows

package main

// 为Windows系统添加必要的导入
import (
	"os"
	"syscall"
)

// isHiddenWindows checks hidden attribute on Windows
func isHiddenDir(path string, info os.FileInfo) (bool, error) {
	// On Windows, we need to check the FILE_ATTRIBUTE_HIDDEN flag
	// This requires using syscall and the Windows API
	pointer, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return false, err
	}
	
	attributes, err := syscall.GetFileAttributes(pointer)
	if err != nil {
		return false, err
	}
	
	return attributes&syscall.FILE_ATTRIBUTE_HIDDEN != 0, nil
}