package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/spf13/cobra"
)

type Config struct {
	SourceDir     string
	SourceString  string
	TargetString  string
	Workers       int
	Trial         bool
	Verbose       bool
}

type Result struct {
	FilesProcessed int32
	FilesFound     int32
	FilesMatches   int32
	Matches        int32
	Errors         int32
}

var rootCmd = &cobra.Command{
	Use:   "reStr",
	Short: "批量字符串替换工具",
	Long: `批量字符串替换工具，支持递归处理目录，
排除隐藏目录及子目录的文件`,
	Run: func(cmd *cobra.Command, args []string) {
		runApp()
	},
}

var cfg Config

func init() {
	rootCmd.PersistentFlags().StringVarP( &cfg.SourceDir,     "dir",     "d", ".",   "源目录路径")
	rootCmd.PersistentFlags().StringVarP( &cfg.SourceString,  "from",    "f", "",    "要替换的源字符串")
	rootCmd.PersistentFlags().StringVarP( &cfg.TargetString,  "to",      "t", "",    "替换成的目标字符串")
	rootCmd.PersistentFlags().BoolVarP(   &cfg.Trial,         "test",    "T", false, "试验模式（不实际修改）")
	rootCmd.PersistentFlags().BoolVarP(   &cfg.Verbose,       "verbose", "v", false, "详细输出")
	rootCmd.PersistentFlags().IntVarP(    &cfg.Workers,       "workers", "w", 4,     "工人数")
}

func runApp() {
	// 参数验证
	if cfg.SourceString == "" {
		log.Fatal("必须指定要替换的源字符串（--from 参数）")
	}
	
	if cfg.TargetString == "" {
		log.Fatal("必须指定替换成的目标字符串（--to 参数）")
	}
	
	if cfg.Workers <= 0 {
		log.Fatal("工人数必须大于0")
	}
	
	// 确保源目录是绝对路径
	absSourceDir, err := filepath.Abs(cfg.SourceDir)
	if err != nil {
		log.Fatalf("无法获取源目录的绝对路径: %v", err)
	}
	cfg.SourceDir = absSourceDir
	
	Run(&cfg)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func Run(config *Config) {	
	fmt.Printf("开始字符串替换...:\n")
	fmt.Printf("  源目录: %s\n", config.SourceDir)
	fmt.Printf("  源字符串: '%s'\n", config.SourceString)
	fmt.Printf("  目标字符串: '%s'\n", config.TargetString)
	fmt.Printf("  工人数: %d\n", config.Workers)
	fmt.Printf("  试验模式: %v\n", config.Trial)
	fmt.Println()
	
	result := &Result{}
	err := processDirectory(config, result)
	if err != nil {
		log.Fatalf("处理目录时发生错误: %v", err)
	}
	
	fmt.Printf("\n最终结果:\n")
	fmt.Printf("  发现文件数: %d\n", atomic.LoadInt32(&result.FilesFound))
	fmt.Printf("  处理文件数: %d\n", atomic.LoadInt32(&result.FilesProcessed))
	fmt.Printf("  匹配文件数: %d\n", atomic.LoadInt32(&result.FilesMatches))
	fmt.Printf("  匹配替换数: %d\n", atomic.LoadInt32(&result.Matches))
	fmt.Printf("  错误: %d\n", atomic.LoadInt32(&result.Errors))
	
	if config.Trial {
		fmt.Println("\n注意：本次运行在试验模式下，未实际执行替换操作.")
	}
}

func processDirectory(config *Config, result *Result) error {
	// Channel for file paths
	fileChan := make(chan string, 1000)
	
	// Wait group for workers
	var wg sync.WaitGroup
	
	// Start worker goroutines
	for i := 0; i < config.Workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			processFiles(config, result, fileChan, workerID)
		}(i)
	}
	
	// Walk directory and send files to channel
	err := filepath.Walk(config.SourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			atomic.AddInt32(&result.Errors, 1)
			if config.Verbose {
				log.Printf("访问目录 %s 时发生错误: %v", path, err)
			}
			return nil
		}
		
		// Skip hidden directories and their contents based on attributes
		if info.IsDir() {
			hidden, err := isHidden(path, info)
			if err != nil {
				if config.Verbose {
					log.Printf("检查目录 %s 隐藏属性时发生错误: %v", path, err)
				}
			}
			
			if hidden {
				if config.Verbose {
					fmt.Printf("跳过隐藏目录: %s\n", path)
				}
				return filepath.SkipDir
			}
			return nil
		}
		
		// Skip non-regular files and hidden files
		if !info.Mode().IsRegular() {
			return nil
		}
		
		hidden, err := isHidden(path, info)
		if err != nil {
			if config.Verbose {
				log.Printf("检查目录 %s 隐藏属性时发生错误: %v", path, err)
			}
		}
		
		if hidden {
			if config.Verbose {
				fmt.Printf("跳过隐藏文件: %s\n", path)
			}
			return nil
		}
		
		// NEW: Skip binary files
		isBinary, err := isBinaryFile(path)
		if err != nil {
			if config.Verbose {
				log.Printf("检查二进制文件 %s 时发生错误: %v", path, err)
			}
		}

		if isBinary {
			if config.Verbose {
			  fmt.Printf("跳过二进制文件: %s\n", path)
			}
			return nil
		}

		atomic.AddInt32(&result.FilesFound, 1)
		fileChan <- path
		return nil
	})
	
	close(fileChan)
	wg.Wait()
	
	return err
}

func processFiles(config *Config, result *Result, fileChan <-chan string, workerID int) {
	for filePath := range fileChan {
		err := processSingleFile(config, result, filePath)
		if err != nil && config.Verbose {
			log.Printf("工人 %d: 处理文件 %s 时发生错误: %v", workerID, filePath, err)
		}
	}
}

func processSingleFile(config *Config, result *Result, filePath string) error {
	atomic.AddInt32(&result.FilesProcessed, 1)
	
	// Check if file contains the search string
	contains, matchCount, err := fileContainsString(filePath, config.SourceString)
	if err != nil {
		atomic.AddInt32(&result.Errors, 1)
		return fmt.Errorf("检查文件 %s 时发生错误: %w", filePath, err)
	}
	
	if !contains {
		// if config.Verbose {
		// 	 fmt.Printf("在文件 %s 中没有匹配字符串\n", filePath)
		// }
		return nil
	}
	
	if config.Verbose {
		fmt.Printf("发现 %4d 处匹配字符串: %s\n", matchCount, filePath)
	}
	
	if config.Trial {
		fmt.Printf("[试验] 替换 %d 处字符串: %s\n", matchCount, filePath)
		atomic.AddInt32(&result.Matches, int32(matchCount))
  	atomic.AddInt32(&result.FilesMatches, 1);
		return nil
	}
	
	// Perform actual replacement
	replacedCount, err := replaceInFile(filePath, config.SourceString, config.TargetString)
	if err != nil {
		atomic.AddInt32(&result.Errors, 1)
		return fmt.Errorf("替换 %s 文件时发生错误: %w", filePath, err)
	}
	
	atomic.AddInt32(&result.Matches, int32(replacedCount))
	atomic.AddInt32(&result.FilesMatches, 1);
	fmt.Printf("替换 %d 处字符串: %s\n", replacedCount, filePath)
	
	return nil
}

func fileContainsString(filePath, searchStr string) (bool, int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return false, 0, err
	}
	defer file.Close()
	
	matchCount := 0
	scanner := bufio.NewScanner(file)
	
	for scanner.Scan() {
		line := scanner.Text()
		count := strings.Count(line, searchStr)
		matchCount += count
	}
	
	if err := scanner.Err(); err != nil {
		return false, 0, err
	}
	
	return matchCount > 0, matchCount, nil
}

func replaceInFile(filePath, searchStr, replaceStr string) (int, error) {
	// Create temporary file
	tempFile := filePath + ".tmp"
	
	inputFile, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer inputFile.Close()
	
	outputFile, err := os.Create(tempFile)
	if err != nil {
		return 0, err
	}
	defer outputFile.Close()
	
	replacementCount := 0
	reader := bufio.NewReader(inputFile)
	writer := bufio.NewWriter(outputFile)
	
	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return replacementCount, err
		}
		
		// Perform replacement on the line (excluding newline character)
		var lineContent string
		if strings.HasSuffix(line, "\n") {
			lineContent = line[:len(line)-1]
		} else {
			lineContent = line
		}
		
		newLineContent := strings.ReplaceAll(lineContent, searchStr, replaceStr)
		
		// Count replacements
		count := (len(lineContent) - len(strings.ReplaceAll(lineContent, searchStr, ""))) / len(searchStr)
		replacementCount += count
		
		// Write the processed line
		if _, writeErr := writer.WriteString(newLineContent); writeErr != nil {
			return replacementCount, writeErr
		}
		
		// Add appropriate newline
		if err == nil {
			// Normal line - use system-appropriate newline
			if _, writeErr := writer.WriteString(getNewline()); writeErr != nil {
				return replacementCount, writeErr
			}
		}
		
		if err == io.EOF {
			break
		}
	}
	
	if err := writer.Flush(); err != nil {
		return replacementCount, err
	}
	
	// Close files before renaming
	inputFile.Close()
	outputFile.Close()
	
	// Replace original file with temporary file
	if err := os.Rename(tempFile, filePath); err != nil {
		return replacementCount, err
	}
	
	return replacementCount, nil
}

// getNewline returns the appropriate newline character for the current platform
func getNewline() string {
	// On Windows, use \r\n, otherwise use \n
	if runtime.GOOS == "windows" {
		return "\r\n"
	}
	return "\n"
}

// isHidden checks if a file or directory is hidden based on system attributes
func isHidden(path string, info os.FileInfo) (bool, error) {
	// Always skip current and parent directory entries
	name := info.Name()
	if name == "." || name == ".." {
		return false, nil
	}
	
	return isHiddenDir(path, info)
}


