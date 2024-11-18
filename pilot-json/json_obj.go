package pilot_json

import (
	"errors"
	"strconv"
	"time"
)

func (this *JsonObject) Parse(json *[]byte) error {
	if len(*json) == 0 {
		return nil
	}
	if (*json)[0] != '{' {
		return errors.New("Expected object")
	}
	keyStart := 0
	keyEnd := 0
	valueStart := 0
	valueEnd := 0
	i := 0
	quote := false
	curly_delim := 0
	square_delim := 0
	for i < len((*json)) {
		skipThrough(&(*json), &i, '"')
		keyStart = i
		skipUntil(&(*json), &i, '"')
		keyEnd = i
		skipToValue(&(*json), &i)
		valueStart = i
		for i < len((*json)) {
			if (*json)[i-1] != '\\' && (*json)[i] == '"' {
				quote = !quote
			}
			if !quote && (*json)[i] == '{' {
				curly_delim++
			}
			if !quote && (*json)[i] == '}' {
				curly_delim--
			}
			if !quote && (*json)[i] == '[' {
				square_delim++
			}
			if !quote && (*json)[i] == ']' {
				square_delim--
			}
			if !quote && curly_delim <= 0 && square_delim <= 0 && ((*json)[i] == ',' || (*json)[i] == '}') {
				break
			}
			i++
		}
		valueEnd = i

		this.data[string((*json)[keyStart:keyEnd])] = (*json)[valueStart:valueEnd]

		i++
	}
	return nil
}

type JsonObject struct {
	data map[string][]byte
}

func NewJsonObject() *JsonObject {
	return &JsonObject{
		data: make(map[string][]byte),
	}
}

func (json *JsonObject) GetString(key string) (*string, *JsonFieldError) {
	val, ok := (*json).data[key]
	if ok {
		str := string(val[1 : len(val)-1])
		return &str, nil
	}
	return nil, NoFieldError(key)
}

func (json *JsonObject) GetInt32(key string) (*int32, *JsonFieldError) {
	val, ok := (*json).data[key]
	if ok {
		i, err := strconv.ParseInt(string(val), 10, 32)
		if err != nil {
			return nil, InvalidFieldError(key, "int32")
		}
		i_sized := int32(i)
		return &i_sized, nil
	}
	return nil, NoFieldError(key)
}

func (json *JsonObject) GetInt64(key string) (*int64, *JsonFieldError) {
	val, ok := (*json).data[key]
	if ok {
		i, err := strconv.ParseInt(string(val), 10, 64)
		if err != nil {
			return nil, InvalidFieldError(key, "int64")
		}
		return &i, nil
	}
	return nil, NoFieldError(key)
}

func (json *JsonObject) GetFloat32(key string) (*float32, *JsonFieldError) {
	val, ok := (*json).data[key]
	if ok {
		str := string(val)
		f, err := strconv.ParseFloat(str, 32)
		if err != nil {
			return nil, InvalidFieldError(key, "float32")
		}
		f_sized := float32(f)
		return &f_sized, nil
	}
	return nil, NoFieldError(key)
}

func (json *JsonObject) GetFloat64(key string) (*float64, *JsonFieldError) {
	val, ok := (*json).data[key]
	if ok {
		str := string(val)
		f, err := strconv.ParseFloat(str, 64)
		if err != nil {
			return nil, InvalidFieldError(key, "float64")
		}
		return &f, nil
	}
	return nil, NoFieldError(key)
}

func (json *JsonObject) GetBool(key string) (*bool, *JsonFieldError) {
	val, ok := (*json).data[key]
	if ok {
		str := string(val)
		b, err := strconv.ParseBool(str)
		if err != nil {
			return nil, InvalidFieldError(key, "bool")
		}
		return &b, nil
	}
	return nil, NoFieldError(key)
}

func (json *JsonObject) GetObject(key string) (*JsonObject, *JsonFieldError) {
	val, ok := (*json).data[key]
	if ok {
		obj := NewJsonObject()
		err := obj.Parse(&val)
		if err != nil {
			return nil, CouldNotParseError(key)
		}
		return obj, nil
	}
	return nil, NoFieldError(key)
}

func (json *JsonObject) GetArray(key string) (*JsonArray, *JsonFieldError) {
	val, ok := (*json).data[key]
	if ok {
		arr := NewJsonArray()
		err := arr.Parse(&val)
		if err != nil {
			return nil, CouldNotParseError(key)
		}
		return arr, nil
	}
	return nil, NoFieldError(key)
}

func (json *JsonObject) GetData(key string) (*[]byte, *JsonFieldError) {
	val, ok := (*json).data[key]
	if ok {
		return &val, nil
	}
	return nil, NoFieldError(key)
}

func (json *JsonObject) GetTime(key string) (*time.Time, *JsonFieldError) {
	val, ok := (*json).data[key]
	if ok {
		if len(val) < 2 {
			return nil, CouldNotParseError(key)
		}
		t, err := time.Parse(time.RFC3339, string(val[1:len(val)-1]))
		if err != nil {
			return &t, CouldNotParseError(key)
		}
		return &t, nil
	}
	return nil, NoFieldError(key)
}
