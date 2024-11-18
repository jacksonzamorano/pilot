package pilot

import (
	"net"
)

const BUFFER_SIZE = 1024

type HttpBuf struct {
	buffer []byte
	con    *net.Conn
}

func NewBuf(con *net.Conn) *HttpBuf {
	return &HttpBuf{buffer: []byte{}, con: con}
}

func (buf *HttpBuf) BufferIn() int {
	next_read := make([]byte, BUFFER_SIZE)
	size, _ := (*buf.con).Read(next_read)
	buf.buffer = append(buf.buffer, next_read...)
	return size
}

func (buf *HttpBuf) ReadUntilKeep(delim byte) []byte {
	for {
		for i := range buf.buffer {
			if buf.buffer[i] == delim {
				res := []byte(buf.buffer[0:i])
				buf.buffer = buf.buffer[i+1:]
				return res
			}
		}
		if buf.BufferIn() == 0 {
			return nil
		}
	}
}
func (buf *HttpBuf) ReadPath() (string, string) {
	path := ""
	query := ""
	for {
		i := 0
		for i < len(buf.buffer) {
			if buf.buffer[i] == '?' {
				path = string(buf.buffer[0:i])
				buf.buffer = buf.buffer[i+1:]
				i = 0
			} else if buf.buffer[i] == ' ' {
				if path == "" {
					path = string(buf.buffer[0:i])
				} else {
					query = string(buf.buffer[0:i])
				}
				return path, query
			}
			i++
		}
		if buf.BufferIn() == 0 {
			return "", ""
		}
	}
}
func (buf *HttpBuf) ReadUntil(delim byte) []byte {
	for {
		for i := range buf.buffer {
			if buf.buffer[i] == delim {
				res := []byte(buf.buffer[0:i])
				buf.buffer = buf.buffer[i+1:]
				return res
			}
		}
		if buf.BufferIn() == 0 {
			return nil
		}
	}
}

func (buf *HttpBuf) ShiftThrough(delim byte) {
	for {
		for i := range buf.buffer {
			if buf.buffer[i] == delim {
				buf.buffer = buf.buffer[i+1:]
				return
			}
		}
		if buf.BufferIn() == 0 {
			return
		}
	}
}
func (buf *HttpBuf) ReadHeaderKey() []byte {
	start := -1
	for {
		if start == -1 {
			for i := 0; i < len(buf.buffer); i++ {
				if buf.buffer[i] > 32 {
					start = i
					break
				}
			}
		}
		if start > -1 {
			for i := start; i < len(buf.buffer); i++ {
				if buf.buffer[i] == ':' {
					res := []byte(buf.buffer[start:i])
					buf.buffer = buf.buffer[i+1:]
					return res
				}
			}
		}
		if buf.BufferIn() == 0 {
			return nil
		}
	}
}
func (buf *HttpBuf) ReadLine() []byte {
	start := -1
	for {
		if start == -1 {
			for i := 0; i < len(buf.buffer); i++ {
				if buf.buffer[i] > 32 {
					start = i
					break
				}
			}
			continue
		}
		for i := start; i < len(buf.buffer); i++ {
			if buf.buffer[i] == '\n' || buf.buffer[i] == '\r' {
				res := []byte(buf.buffer[start:i])
				buf.buffer = buf.buffer[i:]
				return res
			}
		}
		if buf.BufferIn() == 0 {
			return nil
		}
	}
}

func (buf *HttpBuf) ShiftThroughWhitespace() {
	for {
		for i := range buf.buffer {
			if buf.buffer[i] > 32 {
				buf.buffer = buf.buffer[i:]
				return
			}
		}
		if buf.BufferIn() == 0 {
			return
		}
	}
}

func (buf *HttpBuf) Peek() *byte {
	if len(buf.buffer) == 0 {
		if buf.BufferIn() == 0 {
			return nil
		}
	}
	return &buf.buffer[0]
}
func (buf *HttpBuf) EndsHeader() bool {
	if len(buf.buffer) < 4 && buf.BufferIn() == 0 {
		return true
	} else {
		if buf.buffer[0] == '\r' && buf.buffer[1] == '\n' && buf.buffer[2] == '\r' && buf.buffer[3] == '\n' {
			buf.buffer = buf.buffer[4:]
			return true
		} else if buf.buffer[0] == '\n' && buf.buffer[1] == '\n' {
			buf.buffer = buf.buffer[2:]
			return true
		}
	}
	return false
}

func (buf *HttpBuf) ReadExact(count int) []byte {
	for len(buf.buffer) < count {
		bufIn := buf.BufferIn()
		if bufIn == 0 {
			break
		}
	}
	return buf.buffer[:count]
}
