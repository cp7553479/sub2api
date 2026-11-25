package service

import infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"

var (
	ErrUserNotActive      = infraerrors.Forbidden("USER_NOT_ACTIVE", "user is not active")
	ErrEmailExists        = infraerrors.Conflict("EMAIL_EXISTS", "email already exists")
	ErrServiceUnavailable = infraerrors.ServiceUnavailable("SERVICE_UNAVAILABLE", "service temporarily unavailable")
)
