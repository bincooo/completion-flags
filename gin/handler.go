package gin

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/bincooo/completion-flags/common"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"strings"
)

const _BYTES_FIELD = "__BYTES_FILED__"

type byteRC struct {
	*bytes.Reader
}

func (b byteRC) Close() error {
	b.Reset(nil)
	return nil
}

func EmbedFlags(ctx *gin.Context) {
	if !isCompletion(ctx) {
		ctx.Next()
		return
	}

	messages, err := extractMessages(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	messages = common.XmlFlags(messages)
	if err = coverMessages(ctx, messages); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.Next()
}

func extractMessages(ctx *gin.Context) (messages []interface{}, err error) {
	var obj map[string]interface{}
	bio, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		return nil, err
	}

	ctx.Set(_BYTES_FIELD, bio)
	if err = json.Unmarshal(bio, &obj); err != nil {
		return
	}

	if curr, ok := obj["messages"]; ok {
		if m, o := curr.([]interface{}); o {
			if len(m) == 0 {
				return nil, errors.New("field messages is required")
			}
			messages = m
			return
		}
	}

	return nil, errors.New("field messages is required")
}

func coverMessages(ctx *gin.Context, messages []interface{}) (err error) {
	value, exists := ctx.Get(_BYTES_FIELD)
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

	ctx.Request.Body = &byteRC{
		bytes.NewReader(bio),
	}
	return
}

func isCompletion(ctx *gin.Context) bool {
	return strings.Contains(ctx.Request.RequestURI, "/v1/chat/completions")
}
