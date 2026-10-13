package v1

import "net/http"

var (
	// 通用错误
	ErrBadRequest          = newError(http.StatusBadRequest, "Bad Request")
	ErrUnauthorized        = newError(http.StatusUnauthorized, "Unauthorized")
	ErrNotFound            = newError(http.StatusNotFound, "Not Found")
	ErrInternalServerError = newError(http.StatusInternalServerError, "Internal Server Error")

	// 业务错误
	ErrEmailAlreadyUse = newError(1001, "The email is already in use.")
)
