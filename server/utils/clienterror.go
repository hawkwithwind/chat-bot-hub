package utils

type ClientErrorCode int

const (
	UNKNOWN                ClientErrorCode = 0
	PARAM_REQUIRED         ClientErrorCode = 1001
	PARAM_INVALID          ClientErrorCode = 1002
	RESOURCE_INSUFFICIENT  ClientErrorCode = 2001
	RESOURCE_ACCESS_DENIED ClientErrorCode = 2002
	RESOURCE_NOT_FOUND     ClientErrorCode = 2003
)

type ClientError struct {
	Err       error
	ErrorCode ClientErrorCode
}

func NewClientError(code ClientErrorCode, err error) error {
	return &ClientError{err: err, errorCode: code}
}

func (err *ClientError) ErrorCode() ClientErrorCode {
	return err.errorCode
}

func (err *ClientError) Error() string {
	return err.err.Error()
}

