// Package mcp provides a high-performance Model Context Protocol implementation
package mcp

import (
	"bytes"
	"encoding/json"
	"sync"
	
	"mcp-memory/pkg/mcp/protocol"
)

// BufferPool manages a pool of byte buffers to reduce allocations
type BufferPool struct {
	pool sync.Pool
}

// NewBufferPool creates a new buffer pool with a specified initial size
func NewBufferPool(initialSize int) *BufferPool {
	return &BufferPool{
		pool: sync.Pool{
			New: func() interface{} {
				return bytes.NewBuffer(make([]byte, 0, initialSize))
			},
		},
	}
}

// Get retrieves a buffer from the pool
func (bp *BufferPool) Get() *bytes.Buffer {
	buf := bp.pool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

// Put returns a buffer to the pool
func (bp *BufferPool) Put(buf *bytes.Buffer) {
	// Only return buffers that aren't too large to avoid memory bloat
	if buf.Cap() <= 1024*1024 { // 1MB limit
		bp.pool.Put(buf)
	}
}

// JSONEncoderPool manages a pool of JSON encoders
type JSONEncoderPool struct {
	pool sync.Pool
}

// NewJSONEncoderPool creates a new JSON encoder pool
func NewJSONEncoderPool() *JSONEncoderPool {
	return &JSONEncoderPool{
		pool: sync.Pool{
			New: func() interface{} {
				return &pooledEncoder{
					buf: bytes.NewBuffer(make([]byte, 0, 4096)),
				}
			},
		},
	}
}

// pooledEncoder wraps a buffer and encoder
type pooledEncoder struct {
	buf *bytes.Buffer
	enc *json.Encoder
}

// Get retrieves an encoder from the pool
func (ep *JSONEncoderPool) Get() (*pooledEncoder, *bytes.Buffer) {
	pe := ep.pool.Get().(*pooledEncoder)
	pe.buf.Reset()
	pe.enc = json.NewEncoder(pe.buf)
	return pe, pe.buf
}

// Put returns an encoder to the pool
func (ep *JSONEncoderPool) Put(pe *pooledEncoder) {
	if pe.buf.Cap() <= 1024*1024 { // 1MB limit
		ep.pool.Put(pe)
	}
}

// JSONDecoderPool manages a pool of JSON decoders
type JSONDecoderPool struct {
	pool sync.Pool
}

// NewJSONDecoderPool creates a new JSON decoder pool
func NewJSONDecoderPool() *JSONDecoderPool {
	return &JSONDecoderPool{
		pool: sync.Pool{
			New: func() interface{} {
				return &pooledDecoder{}
			},
		},
	}
}

// pooledDecoder wraps a decoder
type pooledDecoder struct {
	dec *json.Decoder
}

// Get retrieves a decoder from the pool with a reader
func (dp *JSONDecoderPool) Get(data []byte) *json.Decoder {
	pd := dp.pool.Get().(*pooledDecoder)
	reader := bytes.NewReader(data)
	pd.dec = json.NewDecoder(reader)
	pd.dec.UseNumber() // Preserve number precision
	return pd.dec
}

// Put returns a decoder to the pool
func (dp *JSONDecoderPool) Put(pd *pooledDecoder) {
	dp.pool.Put(pd)
}

// SlicePool manages pools of slices of different sizes
type SlicePool struct {
	smallPool  sync.Pool  // For slices up to 1KB
	mediumPool sync.Pool  // For slices up to 16KB
	largePool  sync.Pool  // For slices up to 256KB
}

// NewSlicePool creates a new slice pool
func NewSlicePool() *SlicePool {
	return &SlicePool{
		smallPool: sync.Pool{
			New: func() interface{} {
				s := make([]byte, 1024)
				return &s
			},
		},
		mediumPool: sync.Pool{
			New: func() interface{} {
				s := make([]byte, 16*1024)
				return &s
			},
		},
		largePool: sync.Pool{
			New: func() interface{} {
				s := make([]byte, 256*1024)
				return &s
			},
		},
	}
}

// Get retrieves a slice of at least the requested size
func (sp *SlicePool) Get(size int) []byte {
	if size <= 1024 {
		return (*sp.smallPool.Get().(*[]byte))[:size]
	} else if size <= 16*1024 {
		return (*sp.mediumPool.Get().(*[]byte))[:size]
	} else if size <= 256*1024 {
		return (*sp.largePool.Get().(*[]byte))[:size]
	}
	// For very large sizes, allocate directly
	return make([]byte, size)
}

// Put returns a slice to the appropriate pool
func (sp *SlicePool) Put(slice []byte) {
	// Clear sensitive data before returning to pool
	for i := range slice {
		slice[i] = 0
	}
	
	size := cap(slice)
	if size <= 1024 {
		sp.smallPool.Put(&slice)
	} else if size <= 16*1024 {
		sp.mediumPool.Put(&slice)
	} else if size <= 256*1024 {
		sp.largePool.Put(&slice)
	}
	// Don't pool very large slices
}

// GlobalPools provides access to global pool instances
var GlobalPools = struct {
	Buffers      *BufferPool
	JSONEncoders *JSONEncoderPool
	JSONDecoders *JSONDecoderPool
	Slices       *SlicePool
}{
	Buffers:      NewBufferPool(4096),
	JSONEncoders: NewJSONEncoderPool(),
	JSONDecoders: NewJSONDecoderPool(),
	Slices:       NewSlicePool(),
}

// RequestPool manages a pool of request objects
type RequestPool struct {
	pool sync.Pool
}

// NewRequestPool creates a new request pool
func NewRequestPool() *RequestPool {
	return &RequestPool{
		pool: sync.Pool{
			New: func() interface{} {
				return &protocol.JSONRPCRequest{}
			},
		},
	}
}

// Get retrieves a request from the pool
func (rp *RequestPool) Get() *protocol.JSONRPCRequest {
	req := rp.pool.Get().(*protocol.JSONRPCRequest)
	// Reset the request
	req.ID = nil
	req.Method = ""
	req.Params = nil
	return req
}

// Put returns a request to the pool
func (rp *RequestPool) Put(req *protocol.JSONRPCRequest) {
	rp.pool.Put(req)
}

// ResponsePool manages a pool of response objects
type ResponsePool struct {
	pool sync.Pool
}

// NewResponsePool creates a new response pool
func NewResponsePool() *ResponsePool {
	return &ResponsePool{
		pool: sync.Pool{
			New: func() interface{} {
				return &protocol.JSONRPCResponse{}
			},
		},
	}
}

// Get retrieves a response from the pool
func (rp *ResponsePool) Get() *protocol.JSONRPCResponse {
	resp := rp.pool.Get().(*protocol.JSONRPCResponse)
	// Reset the response
	resp.ID = nil
	resp.Result = nil
	resp.Error = nil
	return resp
}

// Put returns a response to the pool
func (rp *ResponsePool) Put(resp *protocol.JSONRPCResponse) {
	rp.pool.Put(resp)
}

// ErrorResponsePool manages a pool of error response objects
type ErrorResponsePool struct {
	pool sync.Pool
}

// NewErrorResponsePool creates a new error response pool
func NewErrorResponsePool() *ErrorResponsePool {
	return &ErrorResponsePool{
		pool: sync.Pool{
			New: func() interface{} {
				return &protocol.JSONRPCError{}
			},
		},
	}
}

// Get retrieves an error response from the pool
func (ep *ErrorResponsePool) Get() *protocol.JSONRPCError {
	err := ep.pool.Get().(*protocol.JSONRPCError)
	// Reset the error response
	err.Code = 0
	err.Message = ""
	err.Data = nil
	return err
}

// Put returns an error response to the pool
func (ep *ErrorResponsePool) Put(err *protocol.JSONRPCError) {
	ep.pool.Put(err)
}

// GlobalRequestPools provides access to global request/response pools
var GlobalRequestPools = struct {
	Requests       *RequestPool
	Responses      *ResponsePool
	ErrorResponses *ErrorResponsePool
}{
	Requests:       NewRequestPool(),
	Responses:      NewResponsePool(),
	ErrorResponses: NewErrorResponsePool(),
}