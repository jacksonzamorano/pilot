package pilot

import (
	"bufio"
	"encoding/json"
	"log"
	"net"
	"strconv"
	"strings"
)

// genericResponse represents the standard JSON response format used by framework response helpers.
// This struct provides a consistent API response structure with a status indicator and message.
//
// Fields:
//   - Status: Boolean indicating success (true) or failure (false)
//   - Message: Human-readable description of the result or error
//
// JSON Output Example:
//
//	{"status": true, "message": "Operation completed successfully"}
//	{"status": false, "message": "Invalid input provided"}
type genericResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
}

// HttpResponse represents a complete HTTP response with headers, body, and status code.
// This struct supports both regular body responses and streaming responses for large content.
//
// Fields:
//   - StatusCode: HTTP status code using type-safe enum
//   - Headers: Map of HTTP response headers
//   - Body: Response content as byte array (used when Writer is nil)
//   - Writer: Buffered reader for streaming responses (optional)
//   - WriterSize: Size of streamed content when using Writer
type HttpResponse struct {
	StatusCode StatusCode
	Headers    map[string]string
	Body       []byte
	Writer     *bufio.Reader
	WriterSize int64
}

// StringResponse creates a plain text HTTP response.
// Returns a 200 OK response with "text/plain" content type.
func StringResponse(body string) *HttpResponse {
	res := NewHttpResponse()
	res.StatusCode = StatusOK
	res.Headers["Content-Type"] = "text/plain"
	res.Body = []byte(body)
	return res
}

// ErrorResponse creates a 500 Internal Server Error response from an error.
// Logs the actual error for debugging but sends a generic message to prevent information leakage.
func ErrorResponse(body error) *HttpResponse {
	log.Printf("[ERROR]: %v", body.Error())
	errorResponse := genericResponse{
		Status:  false,
		Message: "An error occurred and your request could not be completed.",
	}
	json, _ := json.Marshal(errorResponse)
	res := NewHttpResponse()
	res.StatusCode = StatusInternalServerError
	res.Headers["Content-Type"] = "application/json"
	res.Body = []byte(json)
	return res
}

// BadRequestResponse creates a 400 Bad Request response with a custom error message.
// Used for client errors such as invalid input or malformed requests.
func BadRequestResponse(message string) *HttpResponse {
	res := JsonResponse(genericResponse{
		Status:  false,
		Message: message,
	})
	res.StatusCode = StatusBadRequest
	return res
}

// ErrorMessageResponse creates a 500 Internal Server Error response with a custom message.
// Unlike ErrorResponse, this sends the provided message directly to the client.
// Use with caution to avoid information leakage.
func ErrorMessageResponse(body string) *HttpResponse {
	errorResponse := genericResponse{
		Status:  false,
		Message: body,
	}
	json, _ := json.Marshal(errorResponse)
	res := NewHttpResponse()
	res.StatusCode = StatusInternalServerError
	res.Headers["Content-Type"] = "application/json"
	res.Body = []byte(json)
	return res
}

// ForbiddenResponse creates a 403 Forbidden response.
// Used when the client lacks permission to access the resource.
func ForbiddenResponse(message string) *HttpResponse {
	res := JsonResponse(genericResponse{
		Status:  false,
		Message: message,
	})
	res.StatusCode = StatusForbidden
	return res
}

// NotFoundResponse creates a 404 Not Found response.
// Used when the requested resource doesn't exist.
func NotFoundResponse(err string) *HttpResponse {
	res := JsonResponse(genericResponse{
		Status:  false,
		Message: err,
	})
	res.StatusCode = StatusNotFound
	return res
}

// ValidationErrorResponse creates a 400 Bad Request response from a validation error.
// Sends the error message directly to the client for validation feedback.
func ValidationErrorResponse(body error) *HttpResponse {
	errorResponse := genericResponse{
		Status:  false,
		Message: body.Error(),
	}
	json, _ := json.Marshal(errorResponse)
	res := NewHttpResponse()
	res.StatusCode = StatusBadRequest
	res.Headers["Content-Type"] = "application/json"
	res.Body = []byte(json)
	return res
}

// SuccessStringResponse creates a standardized JSON success response with a message.
// Returns a 200 OK response with success status and custom message.
func SuccessStringResponse(message string) *HttpResponse {
	return JsonResponse(genericResponse{
		Status:  true,
		Message: message,
	})
}

// JsonResponse creates a JSON HTTP response from any serializable Go data.
// Returns a 200 OK response with "application/json" content type.
func JsonResponse(body any) *HttpResponse {
	res := NewHttpResponse()
	res.StatusCode = StatusOK
	res.Headers["Content-Type"] = "application/json"
	res.Body, _ = json.Marshal(body)
	return res
}

// BufferedResponse creates a streaming HTTP response for large content.
// Uses a buffered reader to stream content without loading it all into memory.
func BufferedResponse(writer *bufio.Reader, length int64) *HttpResponse {
	res := NewHttpResponse()
	res.Writer = writer
	res.WriterSize = length
	res.StatusCode = StatusOK
	return res
}

// SetHeader adds or updates an HTTP response header.
func (self *HttpResponse) SetHeader(key string, value string) {
	self.Headers[key] = value
}

// SetStatus updates the HTTP status code for this response.
func (self *HttpResponse) SetStatus(status StatusCode) {
	self.StatusCode = status
}

// ApplyCors adds CORS headers to the response.
// Used internally by the framework to handle cross-origin requests.
func (self *HttpResponse) ApplyCors(origin *string, headers *string, methods *string) {
	self.SetHeader("Access-Control-Allow-Origin", *origin)
	self.SetHeader("Access-Control-Allow-Headers", *headers)
	self.SetHeader("Access-Control-Allow-Methods", *methods)
}

// Write sends the HTTP response to the client over the TCP connection.
// Formats and transmits the complete HTTP response including status line, headers, and body.
// Used internally by the framework.
func (self *HttpResponse) Write(stream net.Conn) {
	var output strings.Builder
	output.WriteString("HTTP/1.1 ")
	output.WriteString(strconv.Itoa(int(self.StatusCode)))
	output.WriteString(" ")
	output.WriteString(StatusCodeDescriptions[self.StatusCode])
	output.WriteString("\r\n")
	for key, value := range self.Headers {
		output.WriteString(key)
		output.WriteString(": ")
		output.WriteString(value)
		output.WriteString("\r\n")
	}
	output.WriteString("Content-Length: ")
	if self.Writer != nil {
		output.WriteString(strconv.Itoa(int(self.WriterSize)))
	} else {
		output.WriteString(strconv.Itoa(len(self.Body)))
	}
	output.WriteString("\r\n\r\n")

	value := output.String()
	write := 0
	for write < len(value) {
		n, err := stream.Write([]byte(value[write:]))
		if err != nil {
			return
		}
		write += n
	}
	if self.Writer != nil {
		self.Writer.WriteTo(stream)
	} else {
		write = 0
		for write < len(self.Body) {
			n, err := stream.Write(self.Body[write:])
			if err != nil {
				return
			}
			write += n
		}
	}
}

// NewHttpResponse creates a new HttpResponse with default values.
// Returns a response with 200 OK status and empty headers/body.
func NewHttpResponse() *HttpResponse {
	return &HttpResponse{
		StatusCode: StatusOK,
		Headers:    make(map[string]string),
		Body:       []byte{},
	}
}
