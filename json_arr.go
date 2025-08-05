package pilot

import (
	"errors"
	"strconv"

	"github.com/google/uuid"
)

type JsonArray struct {
	data [][]byte
}

func NewJsonArray() *JsonArray {
	return &JsonArray{
		data: make([][]byte, 0),
	}
}

func (this *JsonArray) Parse(json []byte) error {
	i := 0
	for {
		if (json)[i] == '[' {
			i++
			break
		}
		i++
		if i == len(json) {
			return errors.New("Expected array")
		}
	}
	valueStart := 1
	curly_delim := 0
	square_delim := 0
	quote := false
	for i < len(json)-1 {
		valueStart = i
		for i < len(json)-1 {
			if (i < 1 || json[i-1] != '\\') && json[i] == '"' {
				quote = !quote
			}
			if !quote && json[i] == '{' {
				curly_delim++
			}
			if !quote && json[i] == '}' {
				curly_delim--
			}
			if !quote && json[i] == '[' {
				square_delim++
			}
			if !quote && json[i] == ']' {
				square_delim--
			}
			if !quote && curly_delim <= 0 && square_delim <= 0 && (json[i] == ',' || json[i] == ']') {
				break
			}
			i++
		}
		this.data = append(this.data, json[valueStart:i])
		i++
	}
	return nil
}

func (json *JsonArray) GetTrimmedData(index int) ([]byte, error) {
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
	return val[innerIdx:innerEnd+1], nil
}

func (json *JsonArray) GetString(index int) (*string, error) {
	if index < 0 || index >= len(json.data) {
		return nil, NoFieldError(strconv.Itoa(index))
	}
	val := (*json).data[index]
	str := string(val[1 : len(val)-1])
	return &str, nil
}

func (json *JsonArray) GetInt32(index int) (*int32, error) {
	d, verr := json.GetTrimmedData(index)
	if verr != nil {
		return nil, verr
	}
	i, err := strconv.ParseInt(string(d), 10, 32)
	if err != nil {
		return nil, InvalidFieldError(strconv.Itoa(index), "int32")
	}
	i_sized := int32(i)
	return &i_sized, nil
}

func (json *JsonArray) GetInt64(index int) (*int64, error) {
	d, verr := json.GetTrimmedData(index)
	if verr != nil {
		return nil, verr
	}
	i, err := strconv.ParseInt(string(d), 10, 64)
	if err != nil {
		return nil, InvalidFieldError(strconv.Itoa(index), "int64")
	}
	return &i, nil
}

func (json *JsonArray) GetFloat32(index int) (*float32, error) {
	d, verr := json.GetTrimmedData(index)
	if verr != nil {
		return nil, verr
	}
	f, err := strconv.ParseFloat(string(d), 32)
	if err != nil {
		return nil, InvalidFieldError(strconv.Itoa(index), "float32")
	}
	f_sized := float32(f)
	return &f_sized, nil
}

func (json *JsonArray) GetFloat64(index int) (*float64, error) {
	d, verr := json.GetTrimmedData(index)
	if verr != nil {
		return nil, verr
	}
	f, err := strconv.ParseFloat(string(d), 64)
	if err != nil {
		return nil, InvalidFieldError(strconv.Itoa(index), "float64")
	}
	return &f, nil
}

func (json *JsonArray) GetBool(index int) (*bool, error) {
	d, verr := json.GetTrimmedData(index)
	if verr != nil {
		return nil, verr
	}
	b, err := strconv.ParseBool(string(d))
	if err != nil {
		return nil, InvalidFieldError(strconv.Itoa(index), "bool")
	}
	return &b, nil
}

func (json *JsonArray) GetUuid(index int) (*uuid.UUID, error) {
	d, verr := json.GetTrimmedData(index)
	if verr != nil {
		return nil, verr
	}
	id, err := uuid.ParseBytes(d)
	if err != nil {
		return nil, InvalidFieldError(strconv.Itoa(index), "uuid")
	}
	return &id, nil
}

func (json *JsonArray) GetObject(index int) (*JsonObject, error) {
	if index < 0 || index >= len(json.data) {
		return nil, NoFieldError(strconv.Itoa(index))
	}
	val := (*json).data[index]
	obj := NewJsonObject()
	err := obj.Parse(val)
	if err != nil {
		return nil, CouldNotParseError(strconv.Itoa(index))
	}
	return obj, nil
}

func (json *JsonArray) GetArray(index int) (*JsonArray, error) {
	if index < 0 || index >= len(json.data) {
		return nil, NoFieldError(strconv.Itoa(index))
	}
	val := (*json).data[index]
	arr := NewJsonArray()
	err := arr.Parse(val)
	if err != nil {
		return nil, CouldNotParseError(strconv.Itoa(index))
	}
	return arr, nil
}

func (json *JsonArray) GetData(index int) (*[]byte, error) {
	if index < 0 || index >= len(json.data) {
		return nil, NoFieldError(strconv.Itoa(index))
	}
	return &(*json).data[index], nil
}

func (json *JsonArray) Length() int {
	return len(json.data)
}
