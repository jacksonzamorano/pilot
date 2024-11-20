package pilot_http

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/url"
	"strconv"
	"strings"
)

type HttpRequest struct {
	Path        string
	QueryString string
	Method      HttpMethod
	Body        []byte
	Headers     map[string]string
	_tempMap    *map[string]string
}

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

func ParseRequest(incoming *net.Conn) *HttpRequest {
	// (*incoming).SetReadDeadline(time.Now().Add(time.Second * 2))
	req := HttpRequest{
		Path:        "",
		Method:      "",
		Body:        nil,
		Headers:     make(map[string]string),
		QueryString: "",
	}

	bufReader := bufio.NewReader(*incoming)
	bytes, err := bufReader.ReadBytes(' ')
	if err != nil {
		return nil
	}
	req.Method = HttpMethods[string(bytes)]
	bytes, err = bufReader.ReadBytes(' ')
	if err != nil {
		return nil
	}
	req.Path = string(bytes)
	bytes, err = bufReader.ReadBytes(' ')
	if err != nil {
		log.Println(err)
		return nil
	}
	qryIdx := strings.Index(req.Path, "?") 
	if qryIdx > -1 {
		req.QueryString = req.Path[qryIdx+1:]
		req.Path = req.Path[0:qryIdx]
	}
	bufReader.ReadBytes('\n')
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
		header = header[0:len(header)-2]
		split := strings.Split(header, ":")
		req.Headers[split[0]] = split[1][1:]
	}

	// Read body
	if req.Headers["Content-Length"] != "" {
		bodyLength, _ := strconv.Atoi(req.Headers["Content-Length"])
		body := make([]byte, bodyLength)
		_, err = bufReader.Read(body)
		if err != nil {
			return nil
		}
		req.Body = body
	}

	req.Dump()

	return &req
}
