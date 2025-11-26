package rest_err

import "net/http"

type RestErr struct {
	Message string   `json:"message"`
	Err     string   `json:"error"`
	Code    int      `json:"code"`
	Causes  []Causes `json:"causes,omitempty"`
}

func (r *RestErr) Error() string {
	return r.Message
}

func NewRestErr(message, err string, code int, causes []Causes) *RestErr {
	return &RestErr{
		Message: message,
		Err:     err,
		Code:    code,
		Causes:  causes,
	}
}

func NewBadRequestError(message string) *RestErr {
	return NewRestErr(message, ErrBadRequest, http.StatusBadRequest, nil)
}

func NewBadRequestValidationError(message string, causes []Causes) *RestErr {
	return NewRestErr(message, ErrBadRequest, http.StatusBadRequest, causes)
}

func NewInternalServerError(message string, causes []Causes) *RestErr {
	return NewRestErr(message, ErrInternalServerError, http.StatusInternalServerError, causes)
}

func NewNotFoundError(message string) *RestErr {
	return NewRestErr(message, ErrNotFound, http.StatusNotFound, nil)
}

func NewForbiddenError(message string) *RestErr {
	return NewRestErr(message, ErrForbidden, http.StatusForbidden, nil)
}

func NewExternalProviderError(message string, causes []Causes) *RestErr {
	return NewRestErr(message, ErrExternalProvider, http.StatusBadGateway, causes)
}

func NewConflictValidationError(message string, causes []Causes) *RestErr {
	return NewRestErr(message, ErrConflict, http.StatusConflict, causes)
}
