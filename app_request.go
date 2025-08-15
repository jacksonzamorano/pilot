package pilot

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// RouteRequest encapsulates all context and resources needed by route handlers.
// Provides access to HTTP request details, database connection, application context, and typed route state.
type RouteRequest[T any] struct {
	Request  *HttpRequest
	Database *sql.DB
	Context  context.Context
	State    *T
}

// HttpRequest represents a parsed HTTP request with convenient access methods.
// Provides structured access to headers, body content, query parameters, and path components.
type HttpRequest struct {
	Path        string
	QueryString string
	Method      HttpMethod
	Body        []byte
	Headers     map[string]string
	IpAddress   string
	_tempMap    *map[string]string
}

// QueryMap parses the query string into a map of key-value pairs with URL decoding.
// Handles complete query string parsing including URL encoding/decoding and multiple parameters.
func (req *HttpRequest) QueryMap() map[string]string {
	res := make(map[string]string)
	keyStart := 0
	keyEnd := 0
	valueStart := 0
	valueEnd := 0
	for keyEnd < len(req.QueryString) {
		if req.QueryString[keyEnd] == '=' {
			valueStart = keyEnd + 1
			valueEnd = valueStart
			for valueEnd < len(req.QueryString) && req.QueryString[valueEnd] != '&' {
				valueEnd++
			}
			key, _ := url.QueryUnescape(req.QueryString[keyStart:keyEnd])
			value, _ := url.QueryUnescape(req.QueryString[valueStart:valueEnd])
			res[key] = value
			keyStart = valueEnd + 1
			keyEnd = keyStart
		}
		keyEnd++
	}
	return res
}

// QueryGetInt32 extracts a query parameter as a 32-bit signed integer.
// Returns nil if parameter is missing or cannot be parsed as int32.
func (req *HttpRequest) QueryGetInt32(key string) *int32 {
	if req._tempMap == nil {
		m := req.QueryMap()
		req._tempMap = &m
	}
	val, ok := (*req._tempMap)[key]
	if ok {
		num, err := strconv.Atoi(val)
		if err == nil {
			v := int32(num)
			return &v
		}
	}
	return nil
}

// QueryGetInt64 extracts a query parameter as a 64-bit signed integer.
// Returns nil if parameter is missing or cannot be parsed as int64.
func (req *HttpRequest) QueryGetInt64(key string) *int64 {
	if req._tempMap == nil {
		m := req.QueryMap()
		req._tempMap = &m
	}
	val, ok := (*req._tempMap)[key]
	if ok {
		num, err := strconv.Atoi(val)
		if err == nil {
			v := int64(num)
			return &v
		}
	}
	return nil
}

// QueryGetString extracts a query parameter as a URL-decoded string.
// Returns nil if parameter is missing, but returns pointer to empty string for empty parameters.
func (req *HttpRequest) QueryGetString(key string) *string {
	if req._tempMap == nil {
		m := req.QueryMap()
		req._tempMap = &m
	}
	val, ok := (*req._tempMap)[key]
	if ok {
		return &val
	}
	return nil
}

// QueryGetUUID extracts and validates a query parameter as a UUID.
// Returns nil if parameter is missing or not a valid UUID format.
func (req *HttpRequest) QueryGetUUID(key string) *uuid.UUID {
	if req._tempMap == nil {
		m := req.QueryMap()
		req._tempMap = &m
	}
	val, ok := (*req._tempMap)[key]
	if !ok {
		return nil
	}
	g, err := uuid.Parse(val)
	if err != nil {
		return nil
	}
	return &g
}

// Dump outputs a formatted representation of the HTTP request for debugging.
// Prints all request components including method, path, query string, headers, and body.
func (req *HttpRequest) Dump() {
	fmt.Printf("Method: %v\n", req.Method)
	fmt.Printf("Path: %v\n", req.Path)
	fmt.Printf("QueryString: %v\n", req.QueryString)
	for k, v := range req.Headers {
		fmt.Printf("Header: '%v': '%v'\n", k, v)
	}
	if req.Body != nil {
		fmt.Printf("Body: %v\n", string(req.Body))
	}
}

// ParseRequest reads and parses an HTTP request from a TCP connection.
// Implements complete HTTP/1.1 request parser with timeout handling.
// Returns nil for malformed requests or connection errors.
func ParseRequest(incoming *net.Conn) *HttpRequest {
	(*incoming).SetReadDeadline(time.Now().Add(time.Second * 10))
	req := HttpRequest{
		Path:        "",
		Method:      "",
		Body:        nil,
		Headers:     make(map[string]string),
		QueryString: "",
		IpAddress:   (*incoming).RemoteAddr().String(),
	}

	bufReader := bufio.NewReader(*incoming)
	bytes, err := bufReader.ReadBytes(' ')
	if err != nil {
		return nil
	}
	req.Method = HttpMethods[string(bytes[:len(bytes)-1])]
	bytes, err = bufReader.ReadBytes(' ')
	if err != nil {
		return nil
	}
	req.Path = string(bytes[:len(bytes)-1])
	bytes, err = bufReader.ReadBytes('\n')
	if err != nil {
		log.Println(err)
		return nil
	}
	qryIdx := strings.Index(req.Path, "?")
	if qryIdx > -1 {
		req.QueryString = req.Path[qryIdx+1:]
		req.Path = req.Path[0:qryIdx]
	}
	for {
		bytes, err = bufReader.ReadBytes('\n')
		if err != nil {
			return nil
		}
		if bytes[0] == '\r' {
			break
		}
		if bytes[0] == '\n' {
			continue
		}
		header := string(bytes)
		header = header[0 : len(header)-2]
		split := strings.Split(header, ":")
		req.Headers[split[0]] = split[1][1:]
	}

	// Read body
	if req.Headers["Content-Length"] != "" {
		bodyLength, _ := strconv.Atoi(req.Headers["Content-Length"])
		body := make([]byte, bodyLength)
		_, err = io.ReadFull(bufReader, body)
		if err != nil {
			return nil
		}
		req.Body = body
	}

	return &req
}
