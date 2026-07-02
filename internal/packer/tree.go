// internal/packer/tree.go
package packer

import (
	"strings"
)

// FileNode 目录树节点
type FileNode struct {
	Name     string
	IsDir    bool
	Children []*FileNode
}

// BuildTree 从文件列表构建目录树
func BuildTree(files []FileContent) *FileNode {
	root := &FileNode{
		Name:     ".",
		IsDir:    true,
		Children: []*FileNode{},
	}

	for _, file := range files {
		parts := strings.Split(file.RelPath, "/")
		current := root
		for i, part := range parts {
			isLast := (i == len(parts)-1)
			// 查找是否已有该节点
			var found *FileNode
			for _, child := range current.Children {
				if child.Name == part {
					found = child
					break
				}
			}
			if found == nil {
				found = &FileNode{
					Name:     part,
					IsDir:    !isLast,
					Children: []*FileNode{},
				}
				current.Children = append(current.Children, found)
			}
			current = found
		}
	}
	return root
}

// RenderTree 渲染目录树为字符串
func RenderTree(node *FileNode, prefix string, isLast bool) string {
	var result strings.Builder

	if node.Name != "." {
		// 渲染当前节点
		connector := "├── "
		if isLast {
			connector = "└── "
		}
		result.WriteString(prefix)
		result.WriteString(connector)
		if node.IsDir {
			result.WriteString(node.Name)
			result.WriteString("/\n")
		} else {
			result.WriteString(node.Name)
			result.WriteString("\n")
		}
	}

	// 递归渲染子节点
	if node.IsDir && len(node.Children) > 0 {
		newPrefix := prefix
		if node.Name != "." {
			if isLast {
				newPrefix += "    "
			} else {
				newPrefix += "│   "
			}
		}
		for i, child := range node.Children {
			childIsLast := (i == len(node.Children)-1)
			result.WriteString(RenderTree(child, newPrefix, childIsLast))
		}
	}

	return result.String()
}
