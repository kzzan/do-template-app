package v1

import "net/http"

var (
	// 通用错误
	ErrBadRequest          = newError(http.StatusBadRequest, "Bad Request")
	ErrUnauthorized        = newError(http.StatusUnauthorized, "Unauthorized")
	ErrForbidden           = newError(http.StatusForbidden, "Forbidden")
	ErrNotFound            = newError(http.StatusNotFound, "Not Found")
	ErrBadGateway          = newError(http.StatusBadGateway, "Bad Gateway")
	ErrInternalServerError = newError(http.StatusInternalServerError, "Internal Server Error")

	// 业务错误
	ErrUsernameAlreadyUse  = newError(1001, "The username is already in use.")
	ErrSuperAdminProtected = newError(1002, "The super admin cannot be deleted.")
)
