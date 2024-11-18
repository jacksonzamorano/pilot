package pilot

type JsonFieldError struct {
	field     string
	valueType string
	found     bool
	parsed    bool
}

func NoFieldError(field string) *JsonFieldError {
	return &JsonFieldError{field, "", false, true}
}
func InvalidFieldError(field string, valueType string) *JsonFieldError {
	return &JsonFieldError{field, valueType, true, true}
}
func CouldNotParseError(field string) *JsonFieldError {
	return &JsonFieldError{field, "", false, false}
}
func (this *JsonFieldError) AddPath(field string) {
	(*this).field = field + "." + (*this).field
}

func (this *JsonFieldError) Error() string {
	if this.found {
		return "Field " + this.field + " is invalid. Expected " + this.valueType
	} else {
		return "Invalid JSON recieved."
	}
}

type JsonDecodable interface {
	Decode(json []byte) error
}

func skipUntil(buffer *[]byte, i *int, until byte) {
	for (*i) < len(*buffer) {
		if (*buffer)[*i] == until {
			return
		}
		(*i)++
	}
}
func skipThrough(buffer *[]byte, i *int, until byte) {
	for (*i) < len(*buffer) {
		if (*buffer)[*i] == until {
			(*i)++
			return
		}
		(*i)++
	}
}
func skipToValue(buffer *[]byte, i *int) {
	for (*i) < len(*buffer) {
		if (*buffer)[*i] == ' ' || (*buffer)[*i] == ':' {
			(*i)++
			return
		}
		(*i)++
	}
}
