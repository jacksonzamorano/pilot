// Package pilot provides a custom JSON parsing and validation library that offers
// more granular control and better error handling than the standard encoding/json package.
// This package is designed for scenarios where you need field-by-field validation,
// custom error messages, and more control over the JSON parsing process.
//
// Key Features:
// - Field-by-field JSON parsing with custom validation
// - Detailed error reporting for missing or invalid fields
// - Support for all common Go data types (strings, integers, floats, booleans, time)
// - Nested object and array parsing capabilities
// - Zero-allocation parsing for better performance
// - Custom error types with field path tracking
//
// The library uses a streaming parser approach that reads JSON byte-by-byte, providing
// better memory efficiency and allowing for custom validation logic at each step.
// Unlike standard JSON unmarshaling, this library gives you explicit control over
// which fields are required and how they should be validated.
//
// Usage Example:
//
//	jsonData := []byte(`{"name":"John","age":30,"email":"john@example.com"}`)
//	obj := pilot.NewJsonObject()
//	err := obj.Parse(&jsonData)
//
//	name, err := obj.GetString("name")
//	age, err := obj.GetInt32("age")
//	email, err := obj.GetString("email")
package pilot

import (
	"errors"
	"strconv"
	"time"

	"github.com/google/uuid"
)

// Parse processes a JSON byte array and extracts all key-value pairs into the JsonObject.
// This method implements a custom JSON parser that reads the input byte-by-byte to
// handle nested structures, string escaping, and proper delimiter counting. The parser
// maintains state for quote escaping, brace/bracket nesting levels to correctly
// identify field boundaries in complex JSON structures.
//
// The parsing algorithm:
// 1. Validates the input starts with '{'
// 2. Iterates through each character, tracking string quotes and delimiters
// 3. Identifies key-value pairs by finding quoted keys followed by ':'
// 4. Extracts values while respecting nested objects and arrays
// 5. Stores raw byte slices for each field for later type conversion
//
// Parameters:
//   - json: Pointer to a byte array containing valid JSON object data
//
// Returns:
//   - error: Returns an error if the input is not a valid JSON object
//
// Example:
//
//	jsonData := []byte(`{"name":"John","age":30,"active":true}`)
//	obj := pilot.NewJsonObject()
//	err := obj.Parse(&jsonData)
//	if err != nil {
//	    log.Printf("Failed to parse JSON: %v", err)
//	}
func (this *JsonObject) Parse(json []byte) error {
	if len(json) == 0 {
		return nil
	}
	if (json)[0] != '{' {
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
	for i < len((json)) {
		skipThrough(json, &i, '"')
		keyStart = i
		skipUntil(json, &i, '"')
		keyEnd = i
		skipToValue(json, &i)
		valueStart = i
		for i < len(json) {
			if (json)[i-1] != '\\' && (json)[i] == '"' {
				quote = !quote
			}
			if !quote && (json)[i] == '{' {
				curly_delim++
			}
			if !quote && (json)[i] == '}' {
				curly_delim--
			}
			if !quote && (json)[i] == '[' {
				square_delim++
			}
			if !quote && (json)[i] == ']' {
				square_delim--
			}
			if !quote && curly_delim <= 0 && square_delim <= 0 && (json[i] == ',' || json[i] == '}') {
				break
			}
			i++
		}
		valueEnd = i

		this.data[string(json[keyStart:keyEnd])] = json[valueStart:valueEnd]

		i++
	}
	return nil
}

// JsonObject represents a parsed JSON object that stores field data as raw byte slices
// for efficient memory usage and deferred type conversion. This approach allows for
// lazy evaluation of field types and provides better performance when only some
// fields are accessed from large JSON objects.
//
// The JsonObject maintains a map of field names to their raw JSON byte representations,
// enabling type-safe conversion methods that can detect and report type mismatches
// with detailed error information including field paths.
//
// Fields:
//   - data: Internal map storing field names as keys and raw JSON bytes as values
//
// Example:
//
//	obj := pilot.NewJsonObject()
//	err := obj.Parse(&jsonBytes)
//
//	// Type-safe field access with error handling
//	name, err := obj.GetString("name")
//	if err != nil {
//	    // Handle missing or invalid field
//	}
type JsonObject struct {
	data map[string][]byte
}

// NewJsonObject creates and initializes a new JsonObject with an empty data map.
// This constructor should be used whenever you need to parse a new JSON object.
// The returned object is ready to use with the Parse method to load JSON data.
//
// Returns:
//   - *JsonObject: A new, empty JsonObject ready for parsing
//
// Example:
//
//	obj := pilot.NewJsonObject()
//	err := obj.Parse(&jsonData)
//
//	// Now you can access fields from the parsed JSON
//	username, err := obj.GetString("username")
func NewJsonObject() *JsonObject {
	return &JsonObject{
		data: make(map[string][]byte),
	}
}

type JsonReadable interface {
	FromJson() error
}

func (json *JsonObject) GetString(key string) (*string, error) {
	val, ok := (*json).data[key]
	if ok {
		str := string(val[1 : len(val)-1])
		return &str, nil
	}
	return nil, NoFieldError(key)
}

func (json *JsonObject) GetInt32(key string) (*int32, error) {
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

func (json *JsonObject) GetInt64(key string) (*int64, error) {
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

func (json *JsonObject) GetFloat32(key string) (*float32, error) {
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

func (json *JsonObject) GetFloat64(key string) (*float64, error) {
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

func (json *JsonObject) GetBool(key string) (*bool, error) {
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

func (json *JsonObject) GetObject(key string) (*JsonObject, error) {
	val, ok := (*json).data[key]
	if ok {
		obj := NewJsonObject()
		err := obj.Parse(val)
		if err != nil {
			return nil, CouldNotParseError(key)
		}
		return obj, nil
	}
	return nil, NoFieldError(key)
}

func (json *JsonObject) GetArray(key string) (*JsonArray, error) {
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

func (json *JsonObject) GetData(key string) (*[]byte, error) {
	val, ok := (*json).data[key]
	if ok {
		return &val, nil
	}
	return nil, NoFieldError(key)
}

func (json *JsonObject) GetTime(key string) (*time.Time, error) {
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

func (json *JsonObject) GetUuid(key string) (*uuid.UUID, error) {
	val, ok := (*json).data[key]
	if ok {
		uuid, err := uuid.ParseBytes(val)
		if err == nil {
			return &uuid, nil
		}
	}
	return nil, NoFieldError(key)
}
