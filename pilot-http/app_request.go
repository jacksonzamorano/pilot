package pilot_http

import (
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
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
func (req *HttpRequest) QueryInt32(key string) *int32 {
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

func ParseRequest(incoming *net.Conn) *HttpRequest {
	(*incoming).SetReadDeadline(time.Now().Add(time.Second * 2))
	req := HttpRequest{
		Path:        "",
		Method:      "",
		Body:        nil,
		Headers:     make(map[string]string),
		QueryString: "",
	}

	buf := NewBuf(incoming)

	req.Method = HttpMethods[string(buf.ReadUntil(' '))]
	req.Path, req.QueryString = buf.ReadPath()
	if strings.HasSuffix(req.Path, "/") && len(req.Path) > 1 {
		req.Path = req.Path[:len(req.Path)-1]
	}
	buf.ShiftThrough('\n')
	for {
		if buf.EndsHeader() {
			break
		}
		key := buf.ReadHeaderKey()
		value := buf.ReadLine()
		req.Headers[string(key)] = string(value)
	}
	header, ok := req.Headers["Content-Length"]
	if ok {
		bc, err := strconv.Atoi(header);
		if err == nil {
			req.Body = buf.ReadExact(bc)
			if len(req.Body) != bc {
				return nil
			}
		}
	}

	return &req
}
