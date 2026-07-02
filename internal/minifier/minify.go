// internal/minifier/minify.go
package minifier

import (
	"io"
	"regexp"
	"strings"
)

// MinifyResult 保存瘦身后的内容及统计信息
type MinifyResult struct {
	Content      string // 瘦身后的内容
	OriginalSize int    // 原始大小（字节）
	MinifiedSize int    // 瘦身后大小（字节）
	CommentLines int    // 移除的注释行数
	BlankLines   int    // 移除的空行数
}

// Minify 对文件内容进行瘦身处理
func Minify(content string, ext string, keepComments bool) MinifyResult {
	originalSize := len(content)
	result := MinifyResult{
		OriginalSize: originalSize,
	}

	// 如果不保留注释，根据扩展名移除注释
	if !keepComments {
		content = removeComments(content, ext)
	}

	// 统计注释行数（简化统计）
	lines := strings.Split(content, "\n")
	originalLines := len(lines)

	// 压缩内容
	content = compressContent(content)

	// 统计瘦身效果
	compressedLines := strings.Split(content, "\n")
	result.CommentLines = originalLines - len(compressedLines)
	result.BlankLines = strings.Count(content, "\n\n")
	result.Content = content
	result.MinifiedSize = len(content)

	return result
}

// removeComments 根据文件扩展名移除注释
func removeComments(content string, ext string) string {
	switch ext {
	case ".go":
		return removeGoComments(content)
	case ".js", ".ts", ".jsx", ".tsx", ".c", ".cpp", ".h", ".hpp", ".java", ".rs", ".php", ".swift", ".kt":
		return removeCstyleComments(content)
	case ".py", ".rb", ".sh", ".bash":
		return removePythonStyleComments(content)
	case ".html", ".xml":
		return removeHtmlComments(content)
	case ".css", ".scss":
		return removeCssComments(content)
	case ".sql":
		return removeSqlComments(content)
	default:
		// 对于文本文件，只移除行首的 # 注释
		return removeGenericComments(content)
	}
}

// removeCstyleComments 移除 C 风格注释（// 和 /* */）
func removeCstyleComments(content string) string {
	// 使用状态机处理多行注释
	var result strings.Builder
	inBlockComment := false
	inString := false
	inChar := false
	escapeNext := false

	i := 0
	for i < len(content) {
		ch := content[i]

		// 处理转义字符
		if escapeNext {
			result.WriteByte(ch)
			escapeNext = false
			i++
			continue
		}

		// 处理字符串字面量（保持内部内容不变）
		if !inBlockComment {
			if ch == '"' && !inChar {
				if i > 0 && content[i-1] != '\\' {
					inString = !inString
				}
			}
			if ch == '\'' && !inString {
				if i > 0 && content[i-1] != '\\' {
					inChar = !inChar
				}
			}
		}

		// 如果在字符串或字符中，直接写入并继续
		if inString || inChar {
			result.WriteByte(ch)
			i++
			continue
		}

		// 处理块注释
		if !inBlockComment && i+1 < len(content) && content[i:i+2] == "/*" {
			inBlockComment = true
			i += 2
			continue
		}
		if inBlockComment && i+1 < len(content) && content[i:i+2] == "*/" {
			inBlockComment = false
			i += 2
			continue
		}
		if inBlockComment {
			i++
			continue
		}

		// 处理行注释
		if !inBlockComment && i+1 < len(content) && content[i:i+2] == "//" {
			// 找到行尾
			for i < len(content) && content[i] != '\n' {
				i++
			}
			// 保留换行符
			if i < len(content) && content[i] == '\n' {
				result.WriteByte('\n')
				i++
			}
			continue
		}

		result.WriteByte(ch)
		i++
	}

	return result.String()
}

// removeGoComments 专门处理 Go 语言的注释（包括 doc comment）
func removeGoComments(content string) string {
	// Go 的注释语法与 C 风格相同，但多了 //go: 指令需要保留
	// 简单实现：使用 C 风格移除，但保留 //go: 指令
	lines := strings.Split(content, "\n")
	var result []string
	for _, line := range lines {
		// 检查是否包含 //go: 指令
		if strings.Contains(line, "//go:") {
			result = append(result, line)
			continue
		}
		// 检查是否只有注释（忽略空白）
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") {
			continue
		}
		result = append(result, line)
	}
	return strings.Join(result, "\n")
}

// removePythonStyleComments 移除 Python 风格注释（# 和三引号字符串）
func removePythonStyleComments(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	inTripleQuote := false

	for _, line := range lines {
		// 简单检测三引号（不处理嵌套情况）
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, `"""`) || strings.HasPrefix(trimmed, `'''`) {
			inTripleQuote = !inTripleQuote
			// 如果三引号在同一行结束
			if strings.Count(trimmed, `"""`) >= 2 || strings.Count(trimmed, `'''`) >= 2 {
				inTripleQuote = false
			}
			continue
		}
		if inTripleQuote {
			continue
		}

		// 处理行注释
		if strings.Contains(line, "#") {
			// 检查 # 是否在字符串中（简单处理）
			parts := strings.SplitN(line, "#", 2)
			if len(parts) > 0 {
				// 检查 # 之前是否有未闭合的引号
				beforeHash := parts[0]
				// 简单计数引号数量（不考虑转义）
				singleQuotes := strings.Count(beforeHash, "'")
				doubleQuotes := strings.Count(beforeHash, `"`)
				if singleQuotes%2 == 0 && doubleQuotes%2 == 0 {
					// # 不在字符串中，保留 # 之前的内容
					result = append(result, beforeHash)
					continue
				}
			}
		}
		result = append(result, line)
	}

	return strings.Join(result, "\n")
}

// removeHtmlComments 移除 HTML/XML 注释
func removeHtmlComments(content string) string {
	re := regexp.MustCompile(`<!--.*?-->`)
	return re.ReplaceAllString(content, "")
}

// removeCssComments 移除 CSS 注释（/* */）
func removeCssComments(content string) string {
	re := regexp.MustCompile(`/\*.*?\*/`)
	return re.ReplaceAllString(content, "")
}

// removeSqlComments 移除 SQL 注释（-- 和 /* */）
func removeSqlComments(content string) string {
	// 先移除块注释
	re := regexp.MustCompile(`/\*.*?\*/`)
	content = re.ReplaceAllString(content, "")

	lines := strings.Split(content, "\n")
	var result []string
	for _, line := range lines {
		// 处理行注释
		if strings.Contains(line, "--") {
			parts := strings.SplitN(line, "--", 2)
			result = append(result, parts[0])
		} else {
			result = append(result, line)
		}
	}
	return strings.Join(result, "\n")
}

// removeGenericComments 通用注释移除（只移除行首的 #）
func removeGenericComments(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		result = append(result, line)
	}
	return strings.Join(result, "\n")
}

// compressContent 压缩内容：移除多余空行、行尾空格、文件末尾换行
func compressContent(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	lastLineEmpty := false

	for _, line := range lines {
		// 移除行尾空格
		trimmed := strings.TrimRight(line, " \t")

		// 检查是否为空行
		if len(strings.TrimSpace(trimmed)) == 0 {
			if !lastLineEmpty {
				// 保留一个空行
				result = append(result, "")
				lastLineEmpty = true
			}
			continue
		}
		result = append(result, trimmed)
		lastLineEmpty = false
	}

	// 删除文件末尾的空行
	for len(result) > 0 && strings.TrimSpace(result[len(result)-1]) == "" {
		result = result[:len(result)-1]
	}

	return strings.Join(result, "\n")
}

// MinifyReader 从 io.Reader 读取并瘦身
func MinifyReader(r io.Reader, ext string, keepComments bool) (MinifyResult, error) {
	content, err := io.ReadAll(r)
	if err != nil {
		return MinifyResult{}, err
	}
	return Minify(string(content), ext, keepComments), nil
}

// MinifyFile 直接瘦身文件内容（传入文件内容字符串）
func MinifyFile(content string, filename string, keepComments bool) MinifyResult {
	// 获取文件扩展名
	ext := ""
	if idx := strings.LastIndex(filename, "."); idx != -1 {
		ext = filename[idx:]
	}
	return Minify(content, ext, keepComments)
}
