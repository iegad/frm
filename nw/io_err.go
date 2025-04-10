package nw

import "errors"

var (
	ErrInvalidHeader    = errors.New("header is invalid")
	ErrInvalidBufSize   = errors.New("buf size is invalid")
	ErrConfigNil        = errors.New("IOServiceConfig is nil")
	ErrNoListen         = errors.New("no protocol need to be listen")
	ErrBlend            = errors.New("blend is invalid")
	ErrServiceNil       = errors.New("service is nil")
	ErrTcpEPInvalid     = errors.New("tcp endpoint is invalid")
	ErrWsEPInvalid      = errors.New("ws endpoint is invalid")
	ErrWsMsgTypeInvalid = errors.New("ws message type is invalid")
)
