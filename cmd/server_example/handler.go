package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/RenaLio/tudou/pkg/provider/types"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

type Handler struct {
	Service *RelayService
}

func NewHandler(service *RelayService) *Handler {
	return &Handler{Service: service}
}

func (h *Handler) ChatCompletion(c *gin.Context) {
	h.handleByFormat(c, types.FormatChatCompletion)
}

func (h *Handler) Responses(c *gin.Context) {
	h.handleByFormat(c, types.FormatOpenAIResponses)
}

func (h *Handler) ClaudeMessages(c *gin.Context) {
	h.handleByFormat(c, types.FormatClaudeMessages)
}

func (h *Handler) handleByFormat(c *gin.Context, format types.Format) {
	req := new(types.Request)
	reqBody := c.Request.Body
	bodyBytes, err := io.ReadAll(reqBody)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req, err = validateRequest(bodyBytes)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Format = format
	req.Headers = http.Header{}
	resp, err := h.Service.RelayServiceFunc(c, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if !resp.IsStream {
		contentType := resp.Header.Get("Content-Type")
		if contentType == "" {
			contentType = "application/json"
		}
		c.Data(resp.StatusCode, contentType, resp.RawData)
		return
	}
	// 处理流式响应
	defer resp.Stream.Close()

	c.Status(resp.StatusCode)
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")

	if v := resp.Header.Get("Content-Type"); v != "" {
		c.Header("Content-Type", v)
	}
	if v := resp.Header.Get("Cache-Control"); v != "" {
		c.Header("Cache-Control", v)
	}

	//fmt.Println(resp.Header)
	c.Status(resp.StatusCode)
	c.Stream(func(w io.Writer) bool {
		data, err := resp.Stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return false
			}
			fmt.Println("stream recv error:", err.Error())
			return false
		}

		if _, err := w.Write(data.Content); err != nil {
			fmt.Println("stream write error:", err.Error())
			return false
		}
		c.Writer.Flush()

		return !data.Finished
	})

}

func validateRequest(raw []byte) (*types.Request, error) {
	req := new(types.Request)
	req.Payload = raw
	if gjson.GetBytes(raw, "model").Exists() {
		req.Model = gjson.GetBytes(raw, "model").String()
	} else {
		return nil, errors.New("model is required")
	}

	if gjson.GetBytes(raw, "stream").Bool() {
		req.IsStream = true
	}

	//fmt.Printf("req: %v\n", req)

	return req, nil
}
