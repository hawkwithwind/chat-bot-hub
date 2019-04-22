package utils

type ClientErrorCode int

const (
	OK                     ClientErrorCode = 0
	UNKNOWN                ClientErrorCode = 1
	IGNORED                ClientErrorCode = 2
	PARAM_REQUIRED         ClientErrorCode = 1001
	PARAM_INVALID          ClientErrorCode = 1002
	RESOURCE_INSUFFICIENT  ClientErrorCode = 2001
	RESOURCE_ACCESS_DENIED ClientErrorCode = 2002
	RESOURCE_NOT_FOUND     ClientErrorCode = 2003
	RESOURCE_QUOTA_LIMIT   ClientErrorCode = 2004
	BOT_STATUS_INCONSISTENT ClientErrorCode = 3001
	BOT_METHOD_UNSUPPORTED ClientErrorCode = 3002
)

type ClientError struct {
	error
	Code ClientErrorCode
}

func NewClientError(code ClientErrorCode, err error) error {
	return &ClientError{error: err, Code: code}
}

func (err *ClientError) ErrorCode() ClientErrorCode {
	return err.Code
}
