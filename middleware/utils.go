package middleware

import (
	"fmt"

	"xunkecloudAPI/common"
	"xunkecloudAPI/constant"
	"xunkecloudAPI/logger"

	"github.com/gin-gonic/gin"
)

func abortWithOpenAiMessage(c *gin.Context, statusCode int, message string, code ...string) {
	codeStr := ""
	if len(code) > 0 {
		codeStr = code[0]
	}
	userId := c.GetInt("id")
	// 获取令牌信息（从上下文中获取，只记录前8位和后8位用于识别）
	tokenKey := common.GetContextKeyString(c, constant.ContextKeyTokenKey)
	tokenDisplay := ""
	if len(tokenKey) > 16 {
		tokenDisplay = fmt.Sprintf("%s...%s", tokenKey[:8], tokenKey[len(tokenKey)-8:])
	} else if len(tokenKey) > 0 {
		tokenDisplay = tokenKey
	}

	c.JSON(statusCode, gin.H{
		"error": gin.H{
			"message": common.MessageWithRequestId(message, c.GetString(common.RequestIdKey)),
			"type":    "new_api_error",
			"code":    codeStr,
		},
	})
	c.Abort()

	// 根据是否有令牌信息调整日志格式
	if tokenDisplay != "" {
		logger.LogError(c.Request.Context(), fmt.Sprintf("user %d | token %s | %s", userId, tokenDisplay, message))
	} else {
		logger.LogError(c.Request.Context(), fmt.Sprintf("user %d | %s", userId, message))
	}
}

func abortWithMidjourneyMessage(c *gin.Context, statusCode int, code int, description string) {
	c.JSON(statusCode, gin.H{
		"description": description,
		"type":        "new_api_error",
		"code":        code,
	})
	c.Abort()
	logger.LogError(c.Request.Context(), description)
}
