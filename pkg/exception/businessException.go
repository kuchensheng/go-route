package exception

import "fmt"

type BusinessException struct {
	StatusCode int
	Code       int         `json:"code"`
	Message    string      `json:"message"`
	Data       interface{} `json:"data"`
}

func (e *BusinessException) Error() string {
	return e.Message
}

func (e *BusinessException) New(err error, statusCode, code int, message string, data interface{}, msgArgs ...interface{}) *BusinessException {
	m := err.Error()
	if message != "" {
		m = message
	}
	if len(msgArgs) > 0 {
		m = fmt.Sprintf(message, msgArgs)
	}
	c := 400
	if code != 0 {
		c = code
	}
	sc := c
	if statusCode != 0 {
		sc = statusCode
	}
	return &BusinessException{
		StatusCode: sc,
		Code:       c,
		Message:    m,
		Data:       data,
	}
}
