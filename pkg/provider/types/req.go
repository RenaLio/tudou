package types

import (
	"mime/multipart"
	"net/http"
)

type Request struct {
	Model    string
	IsStream bool
	// 如果请求体是json格式，这里就是json字符串的byte数组
	Payload     []byte
	FormPayload *FormPayload
	Format      Format
	Headers     http.Header
}

// FormPayload 承载 multipart/form-data 的数据
type FormPayload struct {
	Fields map[string]string                `json:"fields,omitempty"` // 普通表单字段
	Files  map[string]*multipart.FileHeader `json:"files,omitempty"`  // 文件字段，key 为表单字段名
}

// FileData 表示一个上传的文件
type FileData struct {
	Filename    string `json:"filename"`              // 原始文件名
	ContentType string `json:"contentType,omitempty"` // MIME 类型，如 "image/png"
	Data        []byte `json:"data"`                  // 文件二进制内容
}
