package perrors

type Error struct {
	// 错误类别
	Kind Kind
	// 哪个阶段出错，比如 client.chat.transform_request
	Op string
	// 提供者
	Provider string
	// 当前格式
	Format string
	// 模型名称
	Model string

	// HTTP状态码
	HTTPStatus int
	// 是否建议重试
	Retryable bool
	// 是否短暂故障
	Temporary bool

	// 如果从上游 header 拿到了，就带上
	RequestID string
	// 比如 rate_limit_exceeded
	UpstreamCode string
	// 简化描述
	SafeMessage string
	// 底层原始 error
	Cause error
}

func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.SafeMessage != "" {
		return e.SafeMessage
	}
	if e.Cause != nil {
		return e.Op + ": " + e.Cause.Error()
	}
	return e.Op + ": " + string(e.Kind)
}

func (e *Error) Unwrap() error { return e.Cause }

func New(kind Kind, op, provider, format, msg string, cause error) *Error {
	return &Error{
		Kind:        kind,
		Op:          op,
		Provider:    provider,
		Format:      format,
		SafeMessage: msg,
		Cause:       cause,
	}
}
