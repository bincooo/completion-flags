package flags

import (
	"github.com/bincooo/completion-flags/common"
	"github.com/gin-gonic/gin"
	"net/http"
)

func EmbedFlags(ctx *gin.Context) {
	if !common.IsCompletion(ctx) {
		return
	}

	messages, err := common.ExtractMessages(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	messages = common.XmlFlags(messages)
	if err = common.CoverMessages(ctx, messages); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
}
