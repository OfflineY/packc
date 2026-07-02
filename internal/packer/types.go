// internal/packer/types.go
package packer

import "strings"

// FileContent 打包用的文件内容
type FileContent struct {
	RelPath string // 文件相对路径（用于显示）
	Content string // 瘦身后的文件内容
}

// ProjectStats 项目统计信息
type ProjectStats struct {
	TotalFiles  int            // 总文件数
	TotalSize   int64          // 总大小
	FileTypeMap map[string]int // 文件类型统计（扩展名 -> 数量）
	RootDir     string         // 根目录
}

// NewProjectStats 创建统计信息
func NewProjectStats(files []FileContent, rootDir string) ProjectStats {
	stats := ProjectStats{
		TotalFiles:  len(files),
		FileTypeMap: make(map[string]int),
		RootDir:     rootDir,
	}

	for _, file := range files {
		ext := ""
		if idx := strings.LastIndex(file.RelPath, "."); idx != -1 {
			ext = file.RelPath[idx:]
		}
		if ext == "" {
			ext = "[no extension]"
		}
		stats.FileTypeMap[ext]++
	}

	return stats
}
