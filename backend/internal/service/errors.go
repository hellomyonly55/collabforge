package service

import "errors"

type AppError struct {
	Status        int
	Code, Message string
}

func (e *AppError) Error() string                 { return e.Code }
func Fail(status int, code, message string) error { return &AppError{status, code, message} }
func AsAppError(err error) *AppError {
	var app *AppError
	if errors.As(err, &app) {
		return app
	}
	return &AppError{500, "INTERNAL_ERROR", "服务暂不可用"}
}
func unavailable() error {
	return Fail(503, "SERVICE_UNAVAILABLE", "服务暂不可用，请稍后重试")
}
func unauthorized() error { return Fail(401, "UNAUTHENTICATED", "登录已失效，请重新登录") }
