// internal/config/config.go
package config

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config 保存所有配置项
type Config struct {
	Paths               []string // 待处理的文件/目录（由 main 设置）
	Extensions          []string // 要打包的扩展名，nil 表示自动识别
	MaxSizeStr          string   // 用户原始大小字符串（如 "1MB"）
	MaxSizeBytes        int64    // 解析后的字节数
	OutputFile          string   // 输出文件名
	KeepComments        bool     // 是否保留注释
	SkipDeps            bool     // 是否跳过依赖目录
	ShowStats           bool     // 是否显示统计信息
	BackupEnabled       bool     // 是否启用备份（仅 INI）
	BackupRetentionDays int      // 备份保留天数（仅 INI）
}

// defaultConfig 返回硬编码的默认值
func defaultConfig() *Config {
	return &Config{
		Extensions:          []string{".go", ".js", ".py", ".java", ".c", ".cpp", ".rs", ".md", ".txt"},
		MaxSizeStr:          "1MB",
		MaxSizeBytes:        1 * 1024 * 1024,
		OutputFile:          "packc_output.txt",
		KeepComments:        false,
		SkipDeps:            true,
		ShowStats:           true,
		BackupEnabled:       true,
		BackupRetentionDays: 7,
	}
}

// getExeDir 返回可执行文件所在目录
func getExeDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exe)
}

// loadIni 读取 INI 文件，若不存在则创建默认，并更新 cfg
func loadIni(cfg *Config) error {
	exeDir := getExeDir()
	iniPath := filepath.Join(exeDir, "packc.ini")

	if _, err := os.Stat(iniPath); os.IsNotExist(err) {
		// 创建默认 INI 文件
		defaultContent := `[Default]
# 默认识别扩展名，逗号分隔
Extensions = .go,.js,.py,.java,.c,.cpp,.rs,.md,.txt
# 单个文件最大大小，支持 KB/MB/GB
MaxSize = 1MB
# 默认输出文件名
Output = packc_output.txt
# 保留注释
KeepComments = false
# 跳过依赖目录node_modules, vendor等 
SkipDeps = true
# Token统计
ShowStats = true
# 启用备份
BackupEnabled = true
BackupRetentionDays = 7
`
		if err := os.WriteFile(iniPath, []byte(defaultContent), 0644); err != nil {
			return fmt.Errorf("failed to create default config file: %w", err)
		}
		fmt.Printf("📄 Created default config file at %s\n", iniPath)
		return nil // 使用已有的默认值（已由 defaultConfig 设置）
	}

	// 读取并解析 INI（仅支持 [Default] 节）
	data, err := os.ReadFile(iniPath)
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}
	lines := strings.Split(string(data), "\n")
	sectionFound := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			sectionFound = strings.EqualFold(line, "[Default]")
			continue
		}
		if !sectionFound {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		switch key {
		case "Extensions":
			if value == "" {
				cfg.Extensions = nil
			} else {
				exts := strings.Split(value, ",")
				clean := make([]string, 0, len(exts))
				for _, e := range exts {
					e = strings.TrimSpace(e)
					if e != "" {
						if !strings.HasPrefix(e, ".") {
							e = "." + e
						}
						clean = append(clean, e)
					}
				}
				cfg.Extensions = clean
			}
		case "MaxSize":
			cfg.MaxSizeStr = value
			if bytes, err := parseSize(value); err == nil {
				cfg.MaxSizeBytes = bytes
			} else {
				fmt.Printf("Warning: invalid MaxSize '%s' in config, using default 1MB\n", value)
				cfg.MaxSizeStr = "1MB"
				cfg.MaxSizeBytes = 1 * 1024 * 1024
			}
		case "Output":
			cfg.OutputFile = value
		case "KeepComments":
			cfg.KeepComments = strings.EqualFold(value, "true")
		case "SkipDeps":
			cfg.SkipDeps = strings.EqualFold(value, "true")
		case "ShowStats":
			cfg.ShowStats = strings.EqualFold(value, "true")
		case "BackupEnabled":
			cfg.BackupEnabled = strings.EqualFold(value, "true")
		case "BackupRetentionDays":
			if days, err := strconv.Atoi(value); err == nil && days > 0 {
				cfg.BackupRetentionDays = days
			}
		}
	}
	return nil
}

// ParseFlags 解析命令行参数，返回最终的 Config
func ParseFlags() *Config {
	cfg := defaultConfig()
	// 加载 INI 覆盖默认值
	if err := loadIni(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
	}

	// 准备 flag 默认值（从当前 cfg 读取）
	extStr := strings.Join(cfg.Extensions, ",")
	maxSizeStr := cfg.MaxSizeStr
	outputStr := cfg.OutputFile
	keepComments := cfg.KeepComments
	skipDeps := cfg.SkipDeps
	showStats := cfg.ShowStats

	// 定义命令行参数
	flag.StringVar(&extStr, "ext", extStr, "要打包的扩展名，逗号分隔，例如 '.go,.js'")
	flag.StringVar(&maxSizeStr, "max-size", maxSizeStr, "单个文件最大大小，例如 '500KB', '2MB'")
	flag.StringVar(&outputStr, "o", outputStr, "输出文件名")
	flag.BoolVar(&keepComments, "keep-comments", keepComments, "保留注释（默认移除）")
	flag.BoolVar(&skipDeps, "skip-deps", skipDeps, "跳过常见的依赖目录（node_modules, vendor等）")
	flag.BoolVar(&showStats, "stats", showStats, "显示统计信息（Token数等）")

	flag.Usage = func() {
		fmt.Printf("Usage: packc [options] [<path>...]\n")
		fmt.Printf("Options:\n")
		flag.PrintDefaults()
		fmt.Printf("\nIf no path is given, current directory '.' is used.\n")
		fmt.Printf("Configuration file 'packc.ini' in the same directory as the executable.\n")
	}

	flag.Parse()

	// 用命令行参数覆盖 cfg
	if extStr != "" {
		parts := strings.Split(extStr, ",")
		exts := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				if !strings.HasPrefix(p, ".") {
					p = "." + p
				}
				exts = append(exts, p)
			}
		}
		cfg.Extensions = exts
	} else {
		cfg.Extensions = nil
	}

	if maxSizeStr != "" {
		cfg.MaxSizeStr = maxSizeStr
		if bytes, err := parseSize(maxSizeStr); err == nil {
			cfg.MaxSizeBytes = bytes
		} else {
			fmt.Fprintf(os.Stderr, "Warning: invalid max-size '%s', keeping config value\n", maxSizeStr)
		}
	}

	cfg.OutputFile = outputStr
	cfg.KeepComments = keepComments
	cfg.SkipDeps = skipDeps
	cfg.ShowStats = showStats

	// Paths 由 main.go 从 flag.Args() 设置
	return cfg
}

// parseSize 将大小字符串（如 "1KB"）转换为字节数
func parseSize(s string) (int64, error) {
	s = strings.ToUpper(strings.TrimSpace(s))
	if s == "" {
		return 0, fmt.Errorf("empty string")
	}
	var numStr string
	var unit int64 = 1
	if strings.HasSuffix(s, "KB") {
		numStr = strings.TrimSuffix(s, "KB")
		unit = 1024
	} else if strings.HasSuffix(s, "MB") {
		numStr = strings.TrimSuffix(s, "MB")
		unit = 1024 * 1024
	} else if strings.HasSuffix(s, "GB") {
		numStr = strings.TrimSuffix(s, "GB")
		unit = 1024 * 1024 * 1024
	} else if strings.HasSuffix(s, "B") {
		numStr = strings.TrimSuffix(s, "B")
		unit = 1
	} else {
		numStr = s
		unit = 1
	}
	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, err
	}
	return int64(num * float64(unit)), nil
}
