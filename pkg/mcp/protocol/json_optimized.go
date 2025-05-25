// Package protocol provides optimized JSON encoding/decoding for MCP
package protocol

import (
	"bytes"
	"encoding/json"
	"io"
	"sync"
	"unsafe"
)

// OptimizedJSONCodec provides high-performance JSON encoding/decoding
// with buffer reuse and minimal allocations
type OptimizedJSONCodec struct {
	encoderPool sync.Pool
	decoderPool sync.Pool
	bufferPool  sync.Pool
}

// NewOptimizedJSONCodec creates a new optimized JSON codec
func NewOptimizedJSONCodec() *OptimizedJSONCodec {
	return &OptimizedJSONCodec{
		encoderPool: sync.Pool{
			New: func() interface{} {
				return &encoderWrapper{
					buf: bytes.NewBuffer(make([]byte, 0, 4096)),
				}
			},
		},
		decoderPool: sync.Pool{
			New: func() interface{} {
				return &decoderWrapper{}
			},
		},
		bufferPool: sync.Pool{
			New: func() interface{} {
				return bytes.NewBuffer(make([]byte, 0, 4096))
			},
		},
	}
}

// encoderWrapper wraps a buffer and encoder
type encoderWrapper struct {
	buf *bytes.Buffer
	enc *json.Encoder
}

// decoderWrapper wraps a decoder
type decoderWrapper struct {
	dec *json.Decoder
}

// Encode efficiently encodes a value to JSON
func (c *OptimizedJSONCodec) Encode(v interface{}) ([]byte, error) {
	// Get encoder from pool
	ew := c.encoderPool.Get().(*encoderWrapper)
	defer c.encoderPool.Put(ew)
	
	// Reset buffer
	ew.buf.Reset()
	
	// Create new encoder if needed
	if ew.enc == nil {
		ew.enc = json.NewEncoder(ew.buf)
		ew.enc.SetEscapeHTML(false) // Avoid HTML escaping overhead
	}
	
	// Encode value
	if err := ew.enc.Encode(v); err != nil {
		return nil, err
	}
	
	// Get bytes without trailing newline
	data := ew.buf.Bytes()
	if len(data) > 0 && data[len(data)-1] == '\n' {
		data = data[:len(data)-1]
	}
	
	// Make a copy to return (buffer will be reused)
	result := make([]byte, len(data))
	copy(result, data)
	
	return result, nil
}

// EncodeToWriter efficiently encodes a value directly to a writer
func (c *OptimizedJSONCodec) EncodeToWriter(w io.Writer, v interface{}) error {
	// Get encoder from pool
	ew := c.encoderPool.Get().(*encoderWrapper)
	defer c.encoderPool.Put(ew)
	
	// Reset buffer
	ew.buf.Reset()
	
	// Create encoder for the writer directly
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	
	return enc.Encode(v)
}

// Decode efficiently decodes JSON data
func (c *OptimizedJSONCodec) Decode(data []byte, v interface{}) error {
	// Get decoder from pool
	dw := c.decoderPool.Get().(*decoderWrapper)
	defer c.decoderPool.Put(dw)
	
	// Create decoder with data
	reader := bytes.NewReader(data)
	dw.dec = json.NewDecoder(reader)
	dw.dec.UseNumber() // Preserve number precision
	
	return dw.dec.Decode(v)
}

// DecodeFromReader efficiently decodes from a reader
func (c *OptimizedJSONCodec) DecodeFromReader(r io.Reader, v interface{}) error {
	// Get decoder from pool
	dw := c.decoderPool.Get().(*decoderWrapper)
	defer c.decoderPool.Put(dw)
	
	// Create decoder for reader
	dw.dec = json.NewDecoder(r)
	dw.dec.UseNumber()
	
	return dw.dec.Decode(v)
}

// FastMarshal provides zero-copy JSON marshaling for specific types
func (c *OptimizedJSONCodec) FastMarshal(v interface{}) ([]byte, error) {
	switch val := v.(type) {
	case *JSONRPCRequest:
		return c.marshalRequest(val)
	case *JSONRPCResponse:
		return c.marshalResponse(val)
	case *JSONRPCError:
		return c.marshalError(val)
	default:
		return c.Encode(v)
	}
}

// marshalRequest efficiently marshals a JSON-RPC request
func (c *OptimizedJSONCodec) marshalRequest(req *JSONRPCRequest) ([]byte, error) {
	buf := c.bufferPool.Get().(*bytes.Buffer)
	defer c.bufferPool.Put(buf)
	buf.Reset()
	
	buf.WriteString(`{"jsonrpc":"`)
	buf.WriteString(req.JSONRPC)
	buf.WriteString(`"`)
	
	if req.ID != nil {
		buf.WriteString(`,"id":`)
		if err := json.NewEncoder(buf).Encode(req.ID); err != nil {
			return nil, err
		}
		// Remove trailing newline
		if buf.Len() > 0 && buf.Bytes()[buf.Len()-1] == '\n' {
			buf.Truncate(buf.Len() - 1)
		}
	}
	
	buf.WriteString(`,"method":"`)
	buf.WriteString(req.Method)
	buf.WriteString(`"`)
	
	if req.Params != nil {
		buf.WriteString(`,"params":`)
		if err := json.NewEncoder(buf).Encode(req.Params); err != nil {
			return nil, err
		}
		// Remove trailing newline
		if buf.Len() > 0 && buf.Bytes()[buf.Len()-1] == '\n' {
			buf.Truncate(buf.Len() - 1)
		}
	}
	
	buf.WriteByte('}')
	
	// Make a copy to return
	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())
	
	return result, nil
}

// marshalResponse efficiently marshals a JSON-RPC response
func (c *OptimizedJSONCodec) marshalResponse(resp *JSONRPCResponse) ([]byte, error) {
	buf := c.bufferPool.Get().(*bytes.Buffer)
	defer c.bufferPool.Put(buf)
	buf.Reset()
	
	buf.WriteString(`{"jsonrpc":"`)
	buf.WriteString(resp.JSONRPC)
	buf.WriteString(`"`)
	
	if resp.ID != nil {
		buf.WriteString(`,"id":`)
		if err := json.NewEncoder(buf).Encode(resp.ID); err != nil {
			return nil, err
		}
		// Remove trailing newline
		if buf.Len() > 0 && buf.Bytes()[buf.Len()-1] == '\n' {
			buf.Truncate(buf.Len() - 1)
		}
	}
	
	if resp.Error != nil {
		buf.WriteString(`,"error":`)
		if err := json.NewEncoder(buf).Encode(resp.Error); err != nil {
			return nil, err
		}
		// Remove trailing newline
		if buf.Len() > 0 && buf.Bytes()[buf.Len()-1] == '\n' {
			buf.Truncate(buf.Len() - 1)
		}
	} else if resp.Result != nil {
		buf.WriteString(`,"result":`)
		if err := json.NewEncoder(buf).Encode(resp.Result); err != nil {
			return nil, err
		}
		// Remove trailing newline
		if buf.Len() > 0 && buf.Bytes()[buf.Len()-1] == '\n' {
			buf.Truncate(buf.Len() - 1)
		}
	}
	
	buf.WriteByte('}')
	
	// Make a copy to return
	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())
	
	return result, nil
}

// marshalError efficiently marshals a JSON-RPC error
func (c *OptimizedJSONCodec) marshalError(err *JSONRPCError) ([]byte, error) {
	buf := c.bufferPool.Get().(*bytes.Buffer)
	defer c.bufferPool.Put(buf)
	buf.Reset()
	
	buf.WriteString(`{"code":`)
	buf.WriteString(itoa(err.Code))
	buf.WriteString(`,"message":"`)
	buf.WriteString(escapeString(err.Message))
	buf.WriteString(`"`)
	
	if err.Data != nil {
		buf.WriteString(`,"data":`)
		if err := json.NewEncoder(buf).Encode(err.Data); err != nil {
			return nil, err
		}
		// Remove trailing newline
		if buf.Len() > 0 && buf.Bytes()[buf.Len()-1] == '\n' {
			buf.Truncate(buf.Len() - 1)
		}
	}
	
	buf.WriteByte('}')
	
	// Make a copy to return
	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())
	
	return result, nil
}

// itoa converts an integer to string without allocation
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	
	var b [20]byte
	pos := len(b)
	neg := i < 0
	if neg {
		i = -i
	}
	
	for i > 0 {
		pos--
		b[pos] = byte('0' + i%10)
		i /= 10
	}
	
	if neg {
		pos--
		b[pos] = '-'
	}
	
	return string(b[pos:])
}

// escapeString escapes a string for JSON without allocations for common cases
func escapeString(s string) string {
	// Fast path: check if escaping is needed
	needsEscape := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '"' || c == '\\' || c < 0x20 {
			needsEscape = true
			break
		}
	}
	
	if !needsEscape {
		return s
	}
	
	// Slow path: build escaped string
	var buf bytes.Buffer
	buf.Grow(len(s) + 10)
	
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '"':
			buf.WriteString(`\"`)
		case '\\':
			buf.WriteString(`\\`)
		case '\n':
			buf.WriteString(`\n`)
		case '\r':
			buf.WriteString(`\r`)
		case '\t':
			buf.WriteString(`\t`)
		default:
			if c < 0x20 {
				buf.WriteString(`\u00`)
				buf.WriteByte(hexDigit(c >> 4))
				buf.WriteByte(hexDigit(c & 0xF))
			} else {
				buf.WriteByte(c)
			}
		}
	}
	
	return buf.String()
}

// hexDigit returns the hex digit for a value
func hexDigit(n byte) byte {
	if n < 10 {
		return '0' + n
	}
	return 'a' + n - 10
}

// GlobalOptimizedCodec provides a global instance of the optimized codec
var GlobalOptimizedCodec = NewOptimizedJSONCodec()

// UnsafeString converts bytes to string without allocation
// WARNING: The byte slice MUST NOT be modified after this call
func UnsafeString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// UnsafeBytes converts string to bytes without allocation
// WARNING: The returned byte slice MUST NOT be modified
func UnsafeBytes(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(&s))
}

// StreamingDecoder provides streaming JSON decoding for large responses
type StreamingDecoder struct {
	reader io.Reader
	dec    *json.Decoder
	buffer []byte
}

// NewStreamingDecoder creates a new streaming decoder
func NewStreamingDecoder(r io.Reader) *StreamingDecoder {
	return &StreamingDecoder{
		reader: r,
		dec:    json.NewDecoder(r),
		buffer: make([]byte, 4096),
	}
}

// Decode decodes the next JSON value from the stream
func (sd *StreamingDecoder) Decode(v interface{}) error {
	return sd.dec.Decode(v)
}

// HasMore returns true if there might be more data to decode
func (sd *StreamingDecoder) HasMore() bool {
	return sd.dec.More()
}