package common

import "strings"

func extractMessage(message map[string]interface{}) *string {
	var str string
	if c, ok := message["content"]; ok {
		if str, ok = c.(string); !ok {
			return nil
		}
	}
	return &str
}

func split(value string) []string {
	contentL := len(value)
	for i := 0; i < contentL; i++ {
		if value[i] == ':' {
			if i < 1 || value[i-1] != '\\' {
				return []string{
					strings.ReplaceAll(value[:i], "\\:", ":"), value[i+1:],
				}
			}
		}
	}
	return nil
}

// 判断切片是否包含子元素， condition：自定义判断规则
func containFor[T comparable](slice []T, condition func(item T) bool) bool {
	if len(slice) == 0 {
		return false
	}

	for idx := 0; idx < len(slice); idx++ {
		if condition(slice[idx]) {
			return true
		}
	}
	return false
}

// int 取绝对值
func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
