package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/RenaLio/tudou/pkg/httpclient"
	"github.com/RenaLio/tudou/pkg/provider/platforms/base"
	"github.com/RenaLio/tudou/pkg/provider/plog"
	"github.com/RenaLio/tudou/pkg/provider/types"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	opts := &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelDebug,
	}
	h := slog.NewTextHandler(os.Stdout, opts)
	logger := slog.New(h)
	slog.SetDefault(logger)

	// loadEnv
	if err := godotenv.Load(); err != nil {
		slog.Info("未找到 .env，继续使用系统环境变量")
	}
	plog.SetLevel(plog.LevelDebug)

	r := gin.Default()
	r.Use(CORSMiddleware())

	httpClient, err := httpclient.GetDefineClient(httpclient.Config{
		Timeout: -1,
	})
	if err != nil {
		panic(err)
	}

	clientBaseURL := envOrDefault("CLIENT_BASE_URL", "https://api.openai.com")
	clientAPIKey := envOrDefault("CLIENT_API_KEY", "")

	client := base.NewClient(
		httpClient,
		clientBaseURL,
		clientAPIKey,
		"demo",
		[]types.Ability{
			types.AbilityChat,
			types.AbilityClaudeMessages,
			//types.AbilityChatCompletions,
			//types.AbilityResponses,
		},
	)

	relay := NewRelayService(client)
	handler := NewHandler(relay)
	r.POST("/v1/chat/completions", handler.ChatCompletion)
	r.POST("/v1/responses", handler.Responses)
	r.POST("/v1/messages", handler.ClaudeMessages)

	r.Run(":8080")
}

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		c.Header("Access-Control-Allow-Origin", c.GetHeader("Origin"))
		c.Header("Access-Control-Allow-Credentials", "true")

		if method == "OPTIONS" {
			c.Header("Access-Control-Allow-Methods", c.GetHeader("Access-Control-Request-Method"))
			c.Header("Access-Control-Allow-Headers", c.GetHeader("Access-Control-Request-Headers"))
			c.Header("Access-Control-Max-Age", "7200")
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func envOrDefault(key string, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
