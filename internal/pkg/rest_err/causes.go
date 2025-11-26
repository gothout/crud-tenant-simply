package rest_err

type Causes struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func NewCause(field, message string) Causes {
	return Causes{
		Field:   field,
		Message: message,
	}
}
