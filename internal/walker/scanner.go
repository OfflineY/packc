// internal/walker/scanner.go
package walker

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileInfo 保存扫描到的单个文件信息
type FileInfo struct {
	Path    string // 文件绝对路径
	RelPath string // 相对于扫描根目录的路径（用于输出显示）
	Size    int64  // 文件大小（字节）
	Ext     string // 文件扩展名（含点）
}

// Scanner 负责扫描和过滤文件
type Scanner struct {
	Extensions   []string // 允许的扩展名，nil 表示使用内置默认列表
	MaxSizeBytes int64    // 最大文件大小
	SkipDeps     bool     // 是否跳过依赖目录
	RootDir      string   // 扫描根目录
	verbose      bool
}

// NewScanner 创建扫描器
func NewScanner(root string, extensions []string, maxSize int64, skipDeps bool) *Scanner {
	return &Scanner{
		RootDir:      root,
		Extensions:   extensions,
		MaxSizeBytes: maxSize,
		SkipDeps:     skipDeps,
		verbose:      true,
	}
}

// defaultExtensions 返回内置的常见代码扩展名
func defaultExtensions() []string {
	return []string{
		".go", ".js", ".ts", ".jsx", ".tsx",
		".py", ".java", ".c", ".cpp", ".h", ".hpp",
		".rs", ".rb", ".php", ".swift", ".kt",
		".md", ".txt", ".json", ".yaml", ".yml",
		".toml", ".xml", ".html", ".css", ".scss",
		".sql", ".sh", ".bash",
	}
}

// isAllowedExtension 检查文件扩展名是否在允许列表中
func (s *Scanner) isAllowedExtension(ext string) bool {
	if len(s.Extensions) == 0 {
		// 使用内置默认列表
		for _, defaultExt := range defaultExtensions() {
			if strings.EqualFold(ext, defaultExt) {
				return true
			}
		}
		return false
	}
	for _, allowed := range s.Extensions {
		if strings.EqualFold(ext, allowed) {
			return true
		}
	}
	return false
}

// isDependencyDir 判断是否为常见的依赖/构建目录
func isDependencyDir(dirName string) bool {
	// 小写化进行比较
	lower := strings.ToLower(dirName)
	deps := []string{
		"node_modules", "vendor", ".git", ".svn", ".hg",
		"dist", "build", "target", "out", "bin", "obj",
		"__pycache__", ".venv", "venv", "env",
		"bower_components", "jspm_packages",
	}
	for _, dep := range deps {
		if lower == dep {
			return true
		}
	}
	return false
}

// shouldSkipDir 判断是否应该跳过该目录
func (s *Scanner) shouldSkipDir(path string) bool {
	if !s.SkipDeps {
		return false
	}
	// 获取目录名（最后一段）
	dirName := filepath.Base(path)
	if isDependencyDir(dirName) {
		return true
	}
	// 检查是否以 . 开头（隐藏目录，如 .git）
	if strings.HasPrefix(dirName, ".") && len(dirName) > 1 {
		return true
	}
	return false
}

// Scan 执行扫描，返回所有符合条件的文件信息
func (s *Scanner) Scan() ([]FileInfo, error) {
	var files []FileInfo
	var skippedTooLarge int
	var skippedExt int
	var skippedDeps int

	// 确保 RootDir 存在
	rootInfo, err := os.Stat(s.RootDir)
	if err != nil {
		return nil, fmt.Errorf("failed to stat root directory: %w", err)
	}

	// 如果根路径是单个文件，直接处理
	if !rootInfo.IsDir() {
		// 检查扩展名
		if s.isAllowedExtension(filepath.Ext(s.RootDir)) {
			if rootInfo.Size() <= s.MaxSizeBytes {
				files = append(files, FileInfo{
					Path:    s.RootDir,
					RelPath: filepath.Base(s.RootDir),
					Size:    rootInfo.Size(),
					Ext:     filepath.Ext(s.RootDir),
				})
			} else {
				skippedTooLarge++
			}
		} else {
			skippedExt++
		}
		// 打印统计
		if s.verbose {
			fmt.Printf("✅ Scan complete: %d files, skipped %d (too large), %d (wrong extension), %d (deps)\n",
				len(files), skippedTooLarge, skippedExt, skippedDeps)
		}
		return files, nil
	}

	// 遍历目录
	err = filepath.WalkDir(s.RootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			// 遇到权限问题等，跳过
			fmt.Printf("⚠️  Warning: cannot access %s: %v\n", path, err)
			return nil
		}

		// 如果是目录，判断是否需要跳过
		if d.IsDir() {
			if s.shouldSkipDir(path) {
				if s.verbose {
					fmt.Printf("⏭️  Skipping dependency directory: %s\n", path)
				}
				return filepath.SkipDir // 跳过整个目录
			}
			return nil
		}

		// 是文件，获取文件信息
		info, err := d.Info()
		if err != nil {
			fmt.Printf("⚠️  Warning: cannot get info for %s: %v\n", path, err)
			return nil
		}

		// 检查扩展名
		ext := filepath.Ext(path)
		if !s.isAllowedExtension(ext) {
			skippedExt++
			return nil
		}

		// 检查文件大小
		fileSize := info.Size()
		if fileSize > s.MaxSizeBytes {
			if s.verbose {
				fmt.Printf("⏭️  Skipping large file: %s (%.2f MB > %d MB)\n",
					path, float64(fileSize)/(1024*1024), s.MaxSizeBytes/(1024*1024))
			}
			skippedTooLarge++
			return nil
		}

		// 计算相对路径
		relPath, err := filepath.Rel(s.RootDir, path)
		if err != nil {
			relPath = filepath.Base(path)
		}
		// 统一使用 Unix 风格路径（对输出友好）
		relPath = filepath.ToSlash(relPath)

		files = append(files, FileInfo{
			Path:    path,
			RelPath: relPath,
			Size:    fileSize,
			Ext:     ext,
		})

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("scan failed: %w", err)
	}

	// 输出统计信息
	if s.verbose {
		fmt.Printf("✅ Scan complete: %d files, %d too large, %d wrong extension, %d dependency skipped\n",
			len(files), skippedTooLarge, skippedExt, skippedDeps)
	}

	return files, nil
}
