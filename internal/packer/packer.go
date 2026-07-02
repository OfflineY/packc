// internal/packer/packer.go
package packer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// PackResult 打包结果统计
type PackResult struct {
	TotalFiles   int    // 打包的文件总数
	TotalSize    int64  // 打包后的总大小（字节）
	OutputPath   string // 输出文件路径
	BackupPath   string // 备份路径（如果有备份）
	BackupCount  int    // 备份目录中的文件数
	DeletedCount int    // 本次清理的过期备份数
}

// Packer 打包器
type Packer struct {
	OutputFile          string // 输出文件路径
	BackupEnabled       bool   // 是否启用备份
	BackupRetentionDays int    // 备份保留天数
	Verbose             bool   // 是否输出详细信息
}

// NewPacker 创建打包器
func NewPacker(outputFile string, backupEnabled bool, backupRetentionDays int) *Packer {
	return &Packer{
		OutputFile:          outputFile,
		BackupEnabled:       backupEnabled,
		BackupRetentionDays: backupRetentionDays,
		Verbose:             true,
	}
}

// Pack 打包多个文件内容
func (p *Packer) Pack(files []FileContent) (PackResult, error) {
	result := PackResult{
		TotalFiles: len(files),
	}

	// 1. 处理备份
	if p.BackupEnabled {
		if err := p.handleBackup(&result); err != nil {
			fmt.Printf("⚠️  Warning: backup failed: %v\n", err)
		}
	}

	// 2. 创建输出目录
	outputDir := filepath.Dir(p.OutputFile)
	if outputDir != "" && outputDir != "." {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return result, fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	// 3. 写入打包内容
	file, err := os.Create(p.OutputFile)
	if err != nil {
		return result, fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	// 4. 写入项目概览
	if err := p.writeOverview(file, files); err != nil {
		return result, fmt.Errorf("failed to write overview: %w", err)
	}

	// 5. 写入每个文件
	var totalSize int64
	// 先计算概览的大小
	overviewSize := int64(0)
	for i, fc := range files {
		// 写入文件边界标记（包含完整相对路径）
		separator := fmt.Sprintf("\n# FILE: %s\n", fc.RelPath)
		if _, err := file.WriteString(separator); err != nil {
			return result, fmt.Errorf("failed to write separator: %w", err)
		}
		totalSize += int64(len(separator))

		// 写入文件内容
		if _, err := file.WriteString(fc.Content); err != nil {
			return result, fmt.Errorf("failed to write content for %s: %w", fc.RelPath, err)
		}
		totalSize += int64(len(fc.Content))

		if p.Verbose && (i+1)%10 == 0 {
			fmt.Printf("  📝 Packed %d/%d files\n", i+1, len(files))
		}
	}

	// 加上概览的大小
	totalSize += overviewSize

	result.TotalSize = totalSize
	result.OutputPath = p.OutputFile

	if p.Verbose {
		fmt.Printf("  ✅ Packed %d files, total size: %s\n",
			len(files), formatSize(totalSize))
	}

	return result, nil
}

// writeOverview 写入项目概览
func (p *Packer) writeOverview(w *os.File, files []FileContent) error {

	// 写入目录树
	if _, err := w.WriteString("# DIRECTORY STRUCTURE\n"); err != nil {
		return err
	}
	tree := BuildTree(files)
	treeStr := RenderTree(tree, "", true)
	if _, err := w.WriteString(treeStr); err != nil {
		return err
	}

	// 分隔符
	if _, err := w.WriteString("\n# ===== FILE CONTENTS =====\n\n"); err != nil {
		return err
	}

	return nil
}

// handleBackup 处理备份逻辑
func (p *Packer) handleBackup(result *PackResult) error {
	// 检查输出文件是否存在
	if _, err := os.Stat(p.OutputFile); os.IsNotExist(err) {
		return nil // 文件不存在，无需备份
	}

	// 获取可执行文件所在目录
	exeDir, err := getExeDir()
	if err != nil {
		return fmt.Errorf("failed to get executable directory: %w", err)
	}

	// 创建 backup 目录
	backupDir := filepath.Join(exeDir, "backup")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// 生成备份文件名（带时间戳）
	baseName := filepath.Base(p.OutputFile)
	ext := filepath.Ext(baseName)
	nameWithoutExt := strings.TrimSuffix(baseName, ext)
	timestamp := time.Now().Format("2006-01-02_150405")
	backupName := fmt.Sprintf("%s_%s%s", nameWithoutExt, timestamp, ext)
	backupPath := filepath.Join(backupDir, backupName)

	// 移动文件到 backup
	if err := os.Rename(p.OutputFile, backupPath); err != nil {
		return fmt.Errorf("failed to move file to backup: %w", err)
	}

	result.BackupPath = backupPath

	// 清理过期备份
	deletedCount, err := p.cleanOldBackups(backupDir)
	if err != nil {
		fmt.Printf("⚠️  Warning: failed to clean old backups: %v\n", err)
	}
	result.DeletedCount = deletedCount

	// 统计备份数量
	backupCount, err := countBackups(backupDir)
	if err == nil {
		result.BackupCount = backupCount
	}

	if p.Verbose {
		fmt.Printf("  💾 Backed up to: %s\n", backupPath)
		if deletedCount > 0 {
			fmt.Printf("  🗑️  Cleaned %d old backup(s)\n", deletedCount)
		}
	}

	return nil
}

// cleanOldBackups 清理超过保留天数的备份文件
func (p *Packer) cleanOldBackups(backupDir string) (int, error) {
	if p.BackupRetentionDays <= 0 {
		return 0, nil
	}

	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return 0, err
	}

	cutoffTime := time.Now().AddDate(0, 0, -p.BackupRetentionDays)
	var deletedCount int

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// 获取文件修改时间
		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoffTime) {
			filePath := filepath.Join(backupDir, entry.Name())
			if err := os.Remove(filePath); err == nil {
				deletedCount++
				if p.Verbose {
					fmt.Printf("  🗑️  Removed old backup: %s\n", entry.Name())
				}
			}
		}
	}

	return deletedCount, nil
}

// countBackups 统计 backup 目录中的文件数
func countBackups(backupDir string) (int, error) {
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			count++
		}
	}
	return count, nil
}

// getExeDir 返回可执行文件所在目录
func getExeDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(exe), nil
}

// formatSize 格式化文件大小
func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
