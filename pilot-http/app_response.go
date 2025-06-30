package pilot_http

import (
	"bufio"
	"encoding/json"
	"net"
	"strconv"
	"strings"
)

type genericResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
}

type HttpResponse struct {
	StatusCode StatusCode
	Headers    map[string]string
	Body       []byte
	Writer     *bufio.Reader
	WriterSize int64
}

func StringResponse(body string) *HttpResponse {
	res := NewHttpResponse()
	res.StatusCode = StatusOK
	res.Headers["Content-Type"] = "text/plain"
	res.Body = []byte(body)
	return res
}
func ErrorResponse(body error) *HttpResponse {
	errorResponse := genericResponse{
		Status:  false,
		Message: body.Error(),
	}
	json, _ := json.Marshal(errorResponse)
	res := NewHttpResponse()
	res.StatusCode = StatusInternalServerError
	res.Headers["Content-Type"] = "application/json"
	res.Body = []byte(json)
	return res
}
func BadRequestResponse(message string) *HttpResponse {
	res := JsonResponse(genericResponse{
		Status:  false,
		Message: message,
	})
	res.StatusCode = StatusBadRequest
	return res
}
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
func ForbiddenResponse(message string) *HttpResponse {
	res := JsonResponse(genericResponse{
		Status:  false,
		Message: message,
	})
	res.StatusCode = StatusForbidden
	return res
}
func NotFoundResponse(err string) *HttpResponse {
	res := JsonResponse(genericResponse{
		Status:  false,
		Message: err,
	})
	res.StatusCode = StatusNotFound
	return res
}
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
func SuccessStringResponse(message string) *HttpResponse {
	return JsonResponse(genericResponse{
		Status:  true,
		Message: message,
	})
}
func JsonResponse(body any) *HttpResponse {
	res := NewHttpResponse()
	res.StatusCode = StatusOK
	res.Headers["Content-Type"] = "application/json"
	res.Body, _ = json.Marshal(body)
	return res
}
func BufferedResponse(writer *bufio.Reader, length int64) *HttpResponse {
	res := NewHttpResponse()
	res.Writer = writer
	res.WriterSize = length
	res.StatusCode = StatusOK
	return res
}

func (self *HttpResponse) SetHeader(key string, value string) {
	self.Headers[key] = value
}
func (self *HttpResponse) SetStatus(status StatusCode) {
	self.StatusCode = status
}
func (self *HttpResponse) ApplyCors(origin *string, headers *string, methods *string) {
	self.SetHeader("Access-Control-Allow-Origin", *origin)
	self.SetHeader("Access-Control-Allow-Headers", *headers)
	self.SetHeader("Access-Control-Allow-Methods", *methods)
}
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
func NewHttpResponse() *HttpResponse {
	return &HttpResponse{
		StatusCode: StatusOK,
		Headers:    make(map[string]string),
		Body:       []byte{},
	}
}
