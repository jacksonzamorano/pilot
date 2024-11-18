package pilot

func PathListFromString(path string) []string {
	route := []string{}
	start := 1
	end := 1
	for end < len(path) {
		if path[end] == '/' {
			route = append(route, path[start:end])
			start = end + 1
		}
		end++
	}
	if start != end || start == 1 {
		route = append(route, path[start:end])
	}
	return route
}

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
