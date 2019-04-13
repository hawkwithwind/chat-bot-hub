package utils

type ClientErrorCode int

const (
	PARAM_REQUIRED ClientErrorCode = 1001
	PARAM_INVALID ClientErrorCode = 1002
	RESOURCE_INSUFFICIENT = 2001
	RESOURCE_ACCESS_DENIED = 2002
	RESOURCE_NOT_FOUND ClientErrorCode = 2003
)

type ClientError struct {
	err       error
	errorCode ClientErrorCode
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

