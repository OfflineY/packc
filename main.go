// main.go
package main

import (
	"flag"
	"fmt"
	"os"
	"packc/internal/config"
	"packc/internal/minifier"
	"packc/internal/packer"
	"packc/internal/tokenizer"
	"packc/internal/walker"
)

func main() {
	// 检查是否只有 init 命令
	if len(os.Args) >= 2 && os.Args[1] == "init" {
		// 初始化 tiktoken
		fmt.Println("🔧 Initializing packc...")
		if tokenizer.InitTiktoken(true) {
			fmt.Println("✅ Initialization complete!")
		} else {
			fmt.Println("❌ Initialization failed or cancelled")
		}
		return
	}

	// 正常执行打包流程
	cfg := config.ParseFlags()

	paths := flag.Args()
	if len(paths) == 0 {
		paths = []string{"."}
	}
	cfg.Paths = paths

	if err := run(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ Pack completed successfully!")
}

func run(cfg *config.Config) error {
	// 打印配置
	printConfig(cfg)

	// 扫描所有路径
	var allFiles []walker.FileInfo
	for _, path := range cfg.Paths {
		fmt.Printf("🔍 Scanning: %s\n", path)
		scanner := walker.NewScanner(path, cfg.Extensions, cfg.MaxSizeBytes, cfg.SkipDeps)
		files, err := scanner.Scan()
		if err != nil {
			fmt.Printf("⚠️  Warning: failed to scan %s: %v\n", path, err)
			continue
		}
		allFiles = append(allFiles, files...)
	}

	if len(allFiles) == 0 {
		return fmt.Errorf("no files found to process")
	}

	fmt.Printf("\n📦 Found %d files to process\n", len(allFiles))

	// 准备打包内容
	packContents := make([]packer.FileContent, 0, len(allFiles))
	var totalOriginalSize int64
	var totalMinifiedSize int64
	var totalCommentLines int

	fmt.Println("\n🔧 Minifying files...")
	for i, file := range allFiles {
		// 读取文件内容
		content, err := os.ReadFile(file.Path)
		if err != nil {
			fmt.Printf("  ⚠️  Warning: cannot read %s: %v\n", file.RelPath, err)
			continue
		}

		// 瘦身
		result := minifier.Minify(string(content), file.Ext, cfg.KeepComments)

		// 统计
		totalOriginalSize += int64(result.OriginalSize)
		totalMinifiedSize += int64(result.MinifiedSize)
		totalCommentLines += result.CommentLines

		// 添加到打包列表
		packContents = append(packContents, packer.FileContent{
			RelPath: file.RelPath,
			Content: result.Content,
		})

		// 显示进度（每10个文件显示一次）
		if (i+1)%10 == 0 || i == len(allFiles)-1 {
			fmt.Printf("  ✅ Processed %d/%d files\n", i+1, len(allFiles))
		}
	}

	if len(packContents) == 0 {
		return fmt.Errorf("no files could be processed")
	}

	// 输出瘦身统计
	if cfg.ShowStats {
		savedBytes := totalOriginalSize - totalMinifiedSize
		savedPercent := float64(savedBytes) / float64(totalOriginalSize) * 100
		fmt.Printf("\n📊 Minification stats:\n")
		fmt.Printf("  Original size: %s\n", formatSize(totalOriginalSize))
		fmt.Printf("  Minified size: %s\n", formatSize(totalMinifiedSize))
		fmt.Printf("  Saved: %s (%.1f%%)\n", formatSize(savedBytes), savedPercent)
		fmt.Printf("  Removed comment lines: %d\n", totalCommentLines)
	}

	// 打包
	fmt.Printf("\n📦 Packing into %s...\n", cfg.OutputFile)
	packerInstance := packer.NewPacker(cfg.OutputFile, cfg.BackupEnabled, cfg.BackupRetentionDays)
	result, err := packerInstance.Pack(packContents)
	if err != nil {
		return fmt.Errorf("packing failed: %w", err)
	}

	// 显示打包结果
	if cfg.ShowStats {
		fmt.Printf("\n📊 Pack result:\n")
		fmt.Printf("  Files packed: %d\n", result.TotalFiles)
		fmt.Printf("  Output file: %s\n", result.OutputPath)
		fmt.Printf("  Total size: %s\n", formatSize(result.TotalSize))
		if result.BackupPath != "" {
			fmt.Printf("  Backup saved: %s\n", result.BackupPath)
		}
		if result.BackupCount > 0 {
			fmt.Printf("  Backups in folder: %d\n", result.BackupCount)
		}
		if result.DeletedCount > 0 {
			fmt.Printf("  Old backups removed: %d\n", result.DeletedCount)
		}
	}

	// Token 估算
	if cfg.ShowStats {
		fmt.Println("\n🔢 Estimating tokens...")

		// 读取打包后的文件
		packedContent, err := os.ReadFile(cfg.OutputFile)
		if err != nil {
			fmt.Printf("⚠️  Warning: cannot read output file: %v\n", err)
		} else {
			// 创建估算器
			estimator := tokenizer.NewEstimator()

			// 估算 Token
			result := estimator.Estimate(string(packedContent))

			// 输出估算结果（简洁版）
			fmt.Println(tokenizer.FormatEstimate(result))

			// 如果用户想要详细信息，可以加 -verbose 参数
			// 这里默认显示简单信息
		}
	}

	return nil
}

func printConfig(cfg *config.Config) {
	fmt.Printf("📋 Configuration:\n")
	fmt.Printf("  Paths: %v\n", cfg.Paths)
	fmt.Printf("  Extensions: %v\n", cfg.Extensions)
	fmt.Printf("  MaxSize: %s (%d bytes)\n", cfg.MaxSizeStr, cfg.MaxSizeBytes)
	fmt.Printf("  Output: %s\n", cfg.OutputFile)
	fmt.Printf("  KeepComments: %v\n", cfg.KeepComments)
	fmt.Printf("  SkipDeps: %v\n", cfg.SkipDeps)
	fmt.Printf("  ShowStats: %v\n", cfg.ShowStats)
	fmt.Printf("  BackupEnabled: %v\n", cfg.BackupEnabled)
	fmt.Printf("  BackupRetentionDays: %d\n", cfg.BackupRetentionDays)
	fmt.Println()
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
