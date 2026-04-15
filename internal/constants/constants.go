package constants

type ctxKeyType struct{}

var userIdKey ctxKeyType = struct{}{}

func UserIdKey() ctxKeyType {
	return userIdKey
}

var claimsKey ctxKeyType = struct{}{}

func ClaimsKey() ctxKeyType {
	return claimsKey
}

var traceIdKey ctxKeyType = struct{}{}

func TraceIdKey() ctxKeyType {
	return traceIdKey
}

var requestIdKey = traceIdKey

func RequestIdKey() ctxKeyType {
	return requestIdKey
}
