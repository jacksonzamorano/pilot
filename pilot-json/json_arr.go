package pilot_json

import (
	"errors"
	"strconv"
)

type JsonArray struct {
	data [][]byte
}

func NewJsonArray() *JsonArray {
	return &JsonArray{
		data: make([][]byte, 0),
	}
}

func (this *JsonArray) Parse(json *[]byte) error {
	i := 0
	for {
		if (*json)[i] == '[' {
			i++
			break
		}
		i++
		if i == len(*json) {
			return errors.New("Expected array")
		}
	}
	valueStart := 1
	curly_delim := 0
	square_delim := 0
	quote := false
	for i < len((*json))-1 {
		valueStart = i
		for i < len((*json))-1 {
			if (i < 1 || (*json)[i-1] != '\\') && (*json)[i] == '"' {
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
			if !quote && curly_delim <= 0 && square_delim <= 0 && ((*json)[i] == ',' || (*json)[i] == ']') {
				break
			}
			i++
		}
		this.data = append(this.data, (*json)[valueStart:i])
		i++
	}
	return nil
}

func (json *JsonArray) GetString(index int) (*string, *JsonFieldError) {
	if index < 0 || index >= len(json.data) {
		return nil, NoFieldError(strconv.Itoa(index))
	}
	val := (*json).data[index]
	str := string(val[1 : len(val)-1])
	return &str, nil
}

func (json *JsonArray) GetInt32(index int) (*int32, *JsonFieldError) {
	if index < 0 || index >= len(json.data) {
		return nil, NoFieldError(strconv.Itoa(index))
	}
	val := (*json).data[index]
	i, err := strconv.ParseInt(string(val), 10, 32)
	if err != nil {
		return nil, InvalidFieldError(strconv.Itoa(index), "int32")
	}
	i_sized := int32(i)
	return &i_sized, nil
}

func (json *JsonArray) GetInt64(index int) (*int64, *JsonFieldError) {
	if index < 0 || index >= len(json.data) {
		return nil, NoFieldError(strconv.Itoa(index))
	}
	val := (*json).data[index]
	innerIdx := 0
	innerEnd := len(val) - 1

	for innerIdx < innerEnd {
		if val[innerIdx] == ' ' || val[innerIdx] == '\n' || val[innerIdx] == '\t' {
			innerIdx++
			continue
		}
		break
	}
	for innerIdx < innerEnd {
		if val[innerEnd] == ' ' || val[innerEnd] == '\n' || val[innerEnd] == '\t' {
			innerEnd--
			continue
		}
		break
	}

	i, err := strconv.ParseInt(string(val[innerIdx:innerEnd]), 10, 64)
	if err != nil {
		return nil, InvalidFieldError(strconv.Itoa(index), "int64")
	}
	return &i, nil
}

func (json *JsonArray) GetFloat32(index int) (*float32, *JsonFieldError) {
	if index < 0 || index >= len(json.data) {
		return nil, NoFieldError(strconv.Itoa(index))
	}
	val := (*json).data[index]
	str := string(val)
	f, err := strconv.ParseFloat(str, 32)
	if err != nil {
		return nil, InvalidFieldError(strconv.Itoa(index), "float32")
	}
	f_sized := float32(f)
	return &f_sized, nil
}

func (json *JsonArray) GetFloat64(index int) (*float64, *JsonFieldError) {
	if index < 0 || index >= len(json.data) {
		return nil, NoFieldError(strconv.Itoa(index))
	}
	val := (*json).data[index]
	str := string(val)
	f, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return nil, InvalidFieldError(strconv.Itoa(index), "float64")
	}
	return &f, nil
}

func (json *JsonArray) GetBool(index int) (*bool, *JsonFieldError) {
	if index < 0 || index >= len(json.data) {
		return nil, NoFieldError(strconv.Itoa(index))
	}
	val := (*json).data[index]
	str := string(val)
	b, err := strconv.ParseBool(str)
	if err != nil {
		return nil, InvalidFieldError(strconv.Itoa(index), "bool")
	}
	return &b, nil
}

func (json *JsonArray) GetObject(index int) (*JsonObject, *JsonFieldError) {
	if index < 0 || index >= len(json.data) {
		return nil, NoFieldError(strconv.Itoa(index))
	}
	val := (*json).data[index]
	obj := NewJsonObject()
	err := obj.Parse(&val)
	if err != nil {
		return nil, CouldNotParseError(strconv.Itoa(index))
	}
	return obj, nil
}

func (json *JsonArray) GetArray(index int) (*JsonArray, *JsonFieldError) {
	if index < 0 || index >= len(json.data) {
		return nil, NoFieldError(strconv.Itoa(index))
	}
	val := (*json).data[index]
	arr := NewJsonArray()
	err := arr.Parse(&val)
	if err != nil {
		return nil, CouldNotParseError(strconv.Itoa(index))
	}
	return arr, nil
}

func (json *JsonArray) GetData(index int) (*[]byte, *JsonFieldError) {
	if index < 0 || index >= len(json.data) {
		return nil, NoFieldError(strconv.Itoa(index))
	}
	return &(*json).data[index], nil
}

func (json *JsonArray) Length() int {
	return len(json.data)
}
