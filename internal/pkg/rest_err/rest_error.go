package rest_err

import "net/http"

type RestErr struct {
	TraceID *string  `json:"trace_id"`
	Message string   `json:"message"`
	Err     string   `json:"error"`
	Code    int      `json:"code"`
	Causes  []Causes `json:"causes,omitempty"`
}

func (r *RestErr) Error() string {
	return r.Message
}

func NewRestErr(trace_id *string, message, err string, code int, causes []Causes) *RestErr {
	return &RestErr{
		TraceID: trace_id,
		Message: message,
		Err:     err,
		Code:    code,
		Causes:  causes,
	}
}

func NewBadRequestError(trace_id *string, message string) *RestErr {
	return NewRestErr(trace_id, message, ErrBadRequest, http.StatusBadRequest, nil)
}

func NewBadRequestValidationError(trace_id *string, message string, causes []Causes) *RestErr {
	return NewRestErr(trace_id, message, ErrBadRequest, http.StatusBadRequest, causes)
}

func NewInternalServerError(trace_id *string, message string, causes []Causes) *RestErr {
	return NewRestErr(trace_id, message, ErrInternalServerError, http.StatusInternalServerError, causes)
}

func NewNotFoundError(trace_id *string, message string) *RestErr {
	return NewRestErr(trace_id, message, ErrNotFound, http.StatusNotFound, nil)
}

func NewForbiddenError(trace_id *string, message string) *RestErr {
	return NewRestErr(trace_id, message, ErrForbidden, http.StatusForbidden, nil)
}

func NewExternalProviderError(trace_id *string, message string, causes []Causes) *RestErr {
	return NewRestErr(trace_id, message, ErrExternalProvider, http.StatusBadGateway, causes)
}

func NewConflictValidationError(trace_id *string, message string, causes []Causes) *RestErr {
	return NewRestErr(trace_id, message, ErrConflict, http.StatusConflict, causes)
}
