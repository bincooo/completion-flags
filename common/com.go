package common

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"io"
	"strings"
)

func ExtractMessages(ctx *gin.Context) (messages []interface{}, err error) {
	var obj map[string]interface{}
	bio, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		return nil, err
	}

	ctx.Set(BYTES_FIELD, bio)
	if err = json.Unmarshal(bio, &obj); err != nil {
		return
	}

	if msgs, ok := obj["messages"]; ok {
		if m, o := msgs.([]interface{}); o {
			if len(m) == 0 {
				return nil, errors.New("field messages is required")
			}
			messages = m
			return
		}
	}

	return nil, errors.New("field messages is required")
}

func CoverMessages(ctx *gin.Context, messages []interface{}) (err error) {
	value, exists := ctx.Get(BYTES_FIELD)
	if !exists {
		return
	}

	bio := value.([]byte)
	var obj map[string]interface{}
	if err = json.Unmarshal(bio, &obj); err != nil {
		return
	}

	obj["messages"] = messages
	bio, err = json.Marshal(obj)
	if err != nil {
		return
	}

	ctx.Request.Body = &BytesReadCloser{
		bytes.NewReader(bio),
	}
	return
}

func IsCompletion(ctx *gin.Context) bool {
	return strings.HasSuffix(ctx.Request.RequestURI, "/v1/chat/completions")
}

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
