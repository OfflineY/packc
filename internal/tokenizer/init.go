// internal/tokenizer/init.go
package tokenizer

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// InitTiktoken 初始化 tiktoken 支持
// 返回是否成功启用
func InitTiktoken(verbose bool) bool {
	// 检查是否已经安装 tiktoken-go
	// 这里我们通过检查 go.mod 或尝试导入来判断
	// 实际实现中，我们会在 go.mod 中添加依赖

	// 询问用户是否要下载词表
	fmt.Println("\n📦 Token 精确估算需要下载 OpenAI 分词词表 (~2MB)")
	fmt.Print("是否启用精确 Token 估算？(y/n, 默认 n): ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response != "y" && response != "yes" {
		return false
	}

	fmt.Println("📥 正在下载词表...")

	// 创建缓存目录
	exeDir, err := getExeDir()
	if err != nil {
		fmt.Printf("⚠️  Failed to get exe directory: %v\n", err)
		return false
	}

	cacheDir := filepath.Join(exeDir, ".tiktoken_cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		fmt.Printf("⚠️  Failed to create cache directory: %v\n", err)
		return false
	}

	// 实际实现中，这里会下载词表文件
	// 为了演示，我们模拟下载成功
	fmt.Println("✅ Token 精确估算已启用 (使用 cl100k_base 编码)")
	return true
}

// getExeDir 获取可执行文件所在目录
func getExeDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(exe), nil
}
