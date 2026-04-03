package perrors

type Kind string

const (
	KindInvalidRequest    Kind = "invalid_request"
	KindUnsupportedFormat Kind = "unsupported_format"
	KindTransformRequest  Kind = "transform_request"
	KindTransformResponse Kind = "transform_response"
	KindBuildRequest      Kind = "build_request"
	KindFetchResponse     Kind = "fetch_response"
	KindBadResponse       Kind = "bad_response"
	KindTransport         Kind = "transport"
	KindTimeout           Kind = "timeout"
	KindCanceled          Kind = "canceled"
	KindUpstreamProtocol  Kind = "upstream_protocol"
	KindStreamRead        Kind = "stream_read"
	KindStreamParse       Kind = "stream_parse"
	KindInternal          Kind = "internal"
)
