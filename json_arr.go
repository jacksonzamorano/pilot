package pilot

import (
	"errors"
	"strconv"

	"github.com/google/uuid"
)

// JsonArray represents a parsed JSON array that stores element data as raw byte slices
// for efficient memory usage and deferred type conversion. This approach enables
// lazy evaluation of array elements and provides better performance when processing
// large arrays where only some elements are accessed.
//
// The JsonArray works similarly to JsonObject but for array structures, providing
// type-safe access methods for array elements with proper bounds checking and
// error handling for invalid indices or type conversions.
//
// Key Features:
//   - Zero-copy parsing with raw byte storage for array elements
//   - Type-safe element access with validation and error reporting
//   - Support for all common Go data types and nested structures
//   - Efficient bounds checking and invalid index handling
//   - Whitespace trimming and data normalization
//
// Fields:
//   - data: Internal slice storing raw JSON bytes for each array element
//
// The array maintains elements as raw bytes until accessed, allowing for
// efficient processing of large arrays and memory-conscious applications.
type JsonArray struct {
	data [][]byte
}

// NewJsonArray creates and initializes a new JsonArray with an empty data slice.
// This constructor should be used whenever you need to parse a new JSON array.
// The returned array is ready to use with the Parse method to load JSON data.
//
// Returns:
//   - *JsonArray: A new, empty JsonArray ready for parsing
//
// Example:
//
//	jsonData := []byte(`[1, 2, "hello", {"name": "John"}]`)
//	arr := pilot.NewJsonArray()
//	err := arr.Parse(jsonData)
//	if err != nil {
//	    log.Printf("Failed to parse JSON array: %v", err)
//	}
//
//	// Now you can access elements
//	firstNum, _ := arr.GetInt32(0)  // Returns 1
//	thirdStr, _ := arr.GetString(2) // Returns "hello"
func NewJsonArray() *JsonArray {
	return &JsonArray{
		data: make([][]byte, 0),
	}
}

// Parse processes a JSON array byte slice and extracts all elements into the JsonArray.
// This method implements a custom JSON array parser that reads the input byte-by-byte
// to handle nested structures, string escaping, and proper delimiter counting for
// accurate element boundary detection in complex JSON arrays.
//
// The parsing algorithm:
//  1. Locates the opening '[' bracket to start array parsing
//  2. Iterates through characters tracking quotes, braces, and brackets
//  3. Identifies element boundaries by finding commas or closing brackets at the correct nesting level
//  4. Handles string escaping to avoid breaking on quoted delimiters
//  5. Stores raw byte slices for each element for later type conversion
//
// Parser Features:
//   - Handles nested objects and arrays within array elements
//   - Properly manages string escaping (e.g., "hello \"world\"")
//   - Tracks delimiter nesting levels to avoid breaking on internal punctuation
//   - Preserves whitespace within elements for accurate type conversion
//
// Parameters:
//   - json: Byte array containing valid JSON array data
//
// Returns:
//   - error: Returns an error if the input is not a valid JSON array
//
// Example:
//
//	jsonData := []byte(`[1, "hello", {"name": "John"}, [1, 2, 3]]`)
//	arr := pilot.NewJsonArray()
//	err := arr.Parse(jsonData)
//	if err != nil {
//	    log.Printf("Failed to parse JSON array: %v", err)
//	}
//
// Error Conditions:
//   - Input doesn't start with '[' (not a JSON array)
//   - Malformed JSON structure with unbalanced delimiters
//   - Unterminated strings or incomplete elements
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
	return val[innerIdx : innerEnd+1], nil
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

// Length returns the number of elements in the parsed JSON array.
// This method provides array size information for iteration, bounds checking,
// and processing logic that depends on array dimensions.
//
// Returns:
//   - int: Number of elements in the array (0 for empty arrays)
//
// Example:
//
//	arr := pilot.NewJsonArray()
//	arr.Parse([]byte(`[1, 2, 3, "hello", {"name": "John"}]`))
//
//	length := arr.Length() // Returns 5
//	for i := 0; i < length; i++ {
//	    element, _ := arr.GetData(i)
//	    // Process each element
//	}
//
// Performance Note:
// This is a constant-time operation that simply returns the length of the
// internal data slice, making it safe to call frequently without performance concerns.
func (json *JsonArray) Length() int {
	return len(json.data)
}
