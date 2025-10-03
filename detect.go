package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// FileType 文件类型枚举
type FileType int

const (
	TextFile FileType = iota
	BinaryFile
	Unknown
)

// DetectFileType 综合检测文件类型
func DetectFileType(filePath string) (FileType, error) {
	// 检查扩展名
	if hasBinaryExtension(filePath) {
		return BinaryFile, nil
	}

  // 检查扩展名
	if hasTextExtension(filePath) {
	  return TextFile, nil
	}

	// 内容检测
	return detectByContent(filePath)
}

// detectByContent 通过文件内容检测类型
func detectByContent(filePath string) (FileType, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return Unknown, err
	}
	defer file.Close()

	buffer := make([]byte, 4096) // 4KB
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return Unknown, err
	}

	if n == 0 {
		return TextFile, nil // 空文件视为文本
	}

	// 检查 null 字节
	for i := 0; i < n; i++ {
		if buffer[i] == 0 {
			return BinaryFile, nil
		}
	}

	// 检查 UTF-8 有效性
	if (n < 4096 || utf8.Valid(buffer[:n])) {
		// 进一步检查可打印字符比例
		if calculatePrintableRatio(buffer[:n]) > 0.85 {
			return TextFile, nil
		} else {
			return BinaryFile, nil
		}
	} else {
		return BinaryFile, nil
	}

	// 默认认为是文本文件（保守策略）
	return TextFile, nil
}

// calculatePrintableRatio 计算可打印字符比例
func calculatePrintableRatio(data []byte) float64 {
	if len(data) == 0 {
		return 1.0
	}

	printableCount := 0
	for i := 0; i < len(data); i++ {
		b := data[i]
		// 可打印 ASCII + 制表符 + 换行符
		if ((b >= 32 && b <= 126) || b == 9 || b == 10 || b == 13) {
			printableCount++
		}
	}

	return float64(printableCount) / float64(len(data))
}

// hasBinaryExtension 检查常见二进制文件扩展名
func hasBinaryExtension(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	
	binaryExtensions := map[string]bool{
		// 可执行文件和库
		".exe": true, ".dll": true, ".so": true, ".dylib": true,
		// 压缩归档文件
		".zip": true, ".rar": true, ".tar": true, 
		".gz": true, ".7z": true, ".bz2": true,
		// 图像文件
		".jpg": true, ".jpeg": true, ".png": true, 
		".gif": true, ".bmp": true, ".tiff": true, ".ico": true,
		// 音频视频文件
		".mp3": true, ".mp4": true, ".avi": true, 
		".mkv": true, ".mov": true, ".wav": true,
		// 编译中间文件
		".o": true, ".obj": true, ".lib": true, ".a": true,
		// 数据库文件
		".db": true, ".sqlite": true, ".mdb": true,
		// 其他二进制格式
		".pdf": true, ".doc": true, ".docx": true, 
		".xls": true, ".xlsx": true, ".ppt": true, ".pptx": true,
		// 虚拟机和容器文件
		".iso": true, ".img": true, ".dmg": true,
		// 字体文件
		".ttf": true, ".otf": true, ".woff": true, ".woff2": true,
		// 包文件
		".jar": true, ".war": true, ".ear": true,
		// 配置文件（有时是二进制）
		".bin": true, ".dat": true,
	}
	
	return binaryExtensions[ext]
}

// hasTextExtension 检查常见文本文件扩展名
func hasTextExtension(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	
	textExtensions := map[string]bool{
		".yaml": true, ".toml": true, ".json": true,
		".sh": true, ".mk": true,
		".c": true, ".cpp": true, ".h": true, ".vala": true, 
		".py": true, ".go": true, ".rs": true, ".ts": true,
	}
	
	return textExtensions[ext]
}

// isBinaryFile 决定是否跳过二进制文件
func isBinaryFile(filePath string) (bool, error) {
	fileType, err := DetectFileType(filePath)
	if err != nil {
		return false, err
	}

	return fileType == BinaryFile, nil
}

