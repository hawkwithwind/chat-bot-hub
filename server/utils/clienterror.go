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
	STATUS_INCONSISTENT    ClientErrorCode = 3001
	METHOD_UNSUPPORTED     ClientErrorCode = 3002
)

type AuthError struct {
	error
}

func NewAuthError(err error) error {
	return &AuthError{
		err,
	}
}

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

type ErrorMessage struct {
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

type Paging struct {
	Page      int64 `json:"page,omitempty"`
	PageCount int64 `json:"pagecount,omitempty"`
	PageSize  int64 `json:"pagesize,omitempty"`
}

type CommonResponse struct {
	Code    int          `json:"code"`
	Message string       `json:"message,omitempty"`
	Ts      int64        `json:"ts"`
	Error   ErrorMessage `json:"error,omitempty""`
	Body    interface{}  `json:"body,omitempty""`
	Paging  Paging       `json:"paging,omitempty"`
}
