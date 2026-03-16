package errors

type errorCode uint16

const (
	errorCodeBadRequest    errorCode = 400
	errorCodeUnauthorized  errorCode = 401
	errorCodeForbidden     errorCode = 403
	errorCodeNotFound      errorCode = 404
	errorCodeConflict      errorCode = 409
	errorCodeUnprocessable errorCode = 422
	errorTooManyRequests   errorCode = 429
	errorCodeInternal      errorCode = 500
)
