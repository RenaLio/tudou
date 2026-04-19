package constants

type ctxKeyType string

var userIdKey ctxKeyType = "user_id"

func UserIdKey() ctxKeyType {
	return userIdKey
}

var claimsKey ctxKeyType = "claims"

func ClaimsKey() ctxKeyType {
	return claimsKey
}

var traceIdKey ctxKeyType = "trace_id"

func TraceIdKey() ctxKeyType {
	return traceIdKey
}

var requestIdKey = traceIdKey

func RequestIdKey() ctxKeyType {
	return requestIdKey
}

var tokenIdKey ctxKeyType = "token_id"

func TokenIdKey() ctxKeyType {
	return tokenIdKey
}

var groupIdKey ctxKeyType = "group_id"

func GroupIdKey() ctxKeyType {
	return groupIdKey
}
