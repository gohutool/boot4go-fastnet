package fastnet

import (
	"crypto/tls"
	routine "github.com/gohutool/boot4go-routine"
	util4go "github.com/gohutool/boot4go-util"
	"github.com/gohutool/boot4go-util/data"
	"io"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

/**
* golang-sample源代码，版权归锦翰科技（深圳）有限公司所有。
* <p>
* 文件名称 : server.go
* 文件路径 :
* 作者 : DavidLiu
× Email: david.liu@ginghan.com
*
* 创建日期 : 2022/6/2 14:06
* 修改历史 : 1. [2022/6/2 14:06] 创建文件 by LongYong
*/

// RequestCtx contains incoming request and manages outgoing response.
//
// It is forbidden copying RequestCtx instances.
//
// RequestHandler should avoid holding references to incoming RequestCtx and/or
// its members after the return.
// If holding RequestCtx references after the return is unavoidable
// (for instance, ctx is passed to a separate goroutine and ctx lifetime cannot
// be controlled), then the RequestHandler MUST call ctx.TimeoutError()
// before return.
//
// It is unsafe modifying/reading RequestCtx instance from concurrently
// running goroutines. The only exception is TimeoutError*, which may be called
// while other goroutines accessing RequestCtx.
type RequestCtx struct {
	connID uint64
	isTLS  bool
	// Connect time
	connTime   time.Time
	remoteAddr net.Addr
	// Last access time
	time       time.Time
	c          net.Conn
	s          *server
	Bytebuffer *data.ByteBuffer
	mu         sync.Mutex

	eventChannel *routine.EventChannel[writeEventChannelObject]
}

// ConnID returns unique connection ID.
//
// This ID may be used to match distinct requests to the same incoming
// connection.
func (ctx *RequestCtx) ConnID() uint64 {
	return ctx.connID
}

// Time returns RequestHandler call time.
func (ctx *RequestCtx) Time() time.Time {
	return ctx.time
}

// ConnTime returns the time the server started serving the connection
// the current request came from.
func (ctx *RequestCtx) ConnTime() time.Time {
	return ctx.connTime
}

// RemoteAddr returns client address for the given request.
//
// Always returns non-nil result.
func (ctx *RequestCtx) RemoteAddr() net.Addr {
	if ctx.remoteAddr != nil {
		return ctx.remoteAddr
	}
	if ctx.c == nil {
		return zeroTCPAddr
	}
	addr := ctx.c.RemoteAddr()
	if addr == nil {
		return zeroTCPAddr
	}
	return addr
}

func (ctx *RequestCtx) reset() {

	ctx.connID = 0
	ctx.connTime = zeroTime
	ctx.remoteAddr = nil
	ctx.time = zeroTime
	ctx.c = nil
	ctx.s.writeEventChain.ReturnOne(ctx.eventChannel)
	ctx.eventChannel = nil

	ctx.s.bufferPool.Put(ctx.Bytebuffer)
	ctx.Bytebuffer = nil

}

// GetReadByteBuffer Maybe concurrency matter

func (ctx *RequestCtx) GetReadByteBuffer() *data.ByteBuffer {
	return ctx.Bytebuffer
}

func (ctx *RequestCtx) Write(b []byte) (int, error) {
	n, err := ctx.c.Write(b)

	if err != nil {
		if ctx.s.OnWriteError(ctx, err) {
			ctx.c.Close()
		}
	}

	ctx.s.OnWrite(ctx, n)

	return n, err
}

func (ctx *RequestCtx) WriteToChannel(b []byte) {
	ctx.eventChannel.AddEvent(&writeEventChannelObject{
		requestCtx: ctx,
		b:          b,
	})
	//ctx.eventChannel.AddEvent(writeEventObjectPool.getObject(ctx, b))
}

func (ctx *RequestCtx) GetConn() net.Conn {
	return ctx.c
}

func (ctx *RequestCtx) CloseConn(err error) {
	ctx.s.OnClose(ctx, err)
	ctx.c.Close()
}

func (ctx *RequestCtx) Error(err error) {
	if netErr, ok := err.(*net.OpError); ok && netErr.Timeout() {
		Logger.Debug("recv timeout  %v \n", err)
	} else {
		Logger.Debug("recv error  %v \n", err)
	}

	if ctx.s.OnReadError(ctx, err) {
		ctx.CloseConn(err)
	}
}

// IsTLS returns true if the underlying connection is tls.Conn.
//
// tls.Conn is an encrypted connection (aka SSL, HTTPS).
func (ctx *RequestCtx) IsTLS() bool {
	// cast to (connTLSer) instead of (*tls.Conn), since it catches
	// cases with overridden tls.Conn such as:
	//
	// type customConn struct {
	//     *tls.Conn
	//
	//     // other custom fields here
	// }

	// perIPConn wraps the net.Conn in the Conn field
	if pic, ok := ctx.c.(*util4go.PerIPConn); ok {
		_, ok := pic.Conn.(connTLSer)
		return ok
	}

	_, ok := ctx.c.(connTLSer)
	return ok
}

type connTLSer interface {
	Handshake() error
	ConnectionState() tls.ConnectionState
}

// RequestHandler must process incoming requests.
//
// RequestHandler must call ctx.TimeoutError() before returning
// if it keeps references to ctx and/or its' members after the return.
// Consider wrapping RequestHandler into TimeoutHandler if response time
// must be limited.
type RequestHandler func(ctx *RequestCtx)

type OnReadError func(ctx *RequestCtx, err error) bool

type OnWriteError func(ctx *RequestCtx, err error) bool

type OnClose func(ctx *RequestCtx, err error)

type OnData func(ctx *RequestCtx, b []byte) error

type OnConnect func(ctx *RequestCtx) bool

type ByteDataHandler func(ctx *RequestCtx, read int) error

type OnWrite func(ctx *RequestCtx, write int)

var globalConnID uint64

func nextConnID() uint64 {
	return atomic.AddUint64(&globalConnID, 1)
}

type ByteBufferDecoder func(bytebuffer *data.ByteBuffer, nread int) (*[][]byte, error)

type Server = server

func NewServer(options ...Option) server {
	opts := LoadOptions(options...)

	s := server{}

	s.MaxIdleWorkerDuration = opts.MaxIdleWorkerDuration
	s.LogAllErrors = opts.LogAllErrors
	s.Concurrency = opts.MaxWorkersCount

	if opts.MaxPackageFrameSize <= 0 {
		s.MaxPackageFrameSize = defaultMaxPackageFrameSize
	} else {
		s.MaxPackageFrameSize = opts.MaxPackageFrameSize
	}

	if opts.WriterEventChannelSize <= 0 {
		s.WriterEventChannelSize = defaultWriterEventChannelSize
	} else {
		s.WriterEventChannelSize = opts.WriterEventChannelSize
	}

	s.Handler = DefaultRequestHandler

	s.OnClose = DummyOnClose
	s.OnReadError = DummyOnReadError
	s.OnWriteError = DummyOnWriteError
	s.ByteDataHandler = DefaultByteDataHandler
	s.OnWrite = DummyOnWrite
	s.OnConnect = DummyOnConnect
	s.OnData = DummyOnData

	s.ReadBufferSize = defaultReadBufferSize
	s.WriteBufferSize = defaultWriteBufferSize

	return s
}

type server struct {

	// Handler for processing incoming requests.
	//
	// Take into account that no `panic` recovery is done by `fasthttp` (thus any `panic` will take down the entire server).
	// Instead the user should use `recover` to handle these situations.
	Handler RequestHandler
	// read byte from socket
	ByteDataHandler ByteDataHandler

	// ErrorHandler for returning a response in case of an error while receiving or parsing the request.
	//
	// The following is a non-exhaustive list of errors that can be expected as argument:
	//   * io.EOF
	//   * TimeOut
	OnReadError OnReadError

	OnWriteError OnWriteError

	OnClose OnClose

	OnConnect OnConnect

	OnData OnData

	// write byte to socket
	OnWrite OnWrite

	ByteBufferDecoder ByteBufferDecoder

	// The maximum number of concurrent connections the server may serve.
	//
	// DefaultConcurrency is used if not set.
	//
	// Concurrency only works if you either call Serve once, or only ServeConn multiple times.
	// It works with ListenAndServe as well.
	Concurrency uint32

	WriterEventChannelSize int

	MaxPackageFrameSize uint

	// Per-connection buffer size for requests' reading.
	// This also limits the maximum header size.
	//
	// Increase this buffer if your clients send multi-KB RequestURIs
	// and/or multi-KB headers (for example, BIG cookies).
	//
	// Default buffer size is used if not set.
	ReadBufferSize int

	// Per-connection buffer size for responses' writing.
	//
	// Default buffer size is used if not set.
	WriteBufferSize int

	maxByteBuffer uint

	// ReadTimeout is the amount of time allowed to read
	// the full request including body. The connection's read
	// deadline is reset when the connection opens, or for
	// keep-alive connections after the first byte has been read.
	//
	// By default, request read timeout is unlimited.
	ReadTimeout time.Duration

	// Maximum number of concurrent client connections allowed per IP.
	//
	// By default, unlimited number of concurrent connections
	// may be established to the server from a single IP address.
	MaxConnsPerIP int

	// Maximum number of read bytes per connection.
	//
	// By default, unlimited number of read bytes may be served per connection.
	MaxReadBytesPerConn int64

	// MaxIdleWorkerDuration is the maximum idle time of a single worker in the underlying
	// worker pool of the Server. Idle workers beyond this time will be cleared.
	MaxIdleWorkerDuration time.Duration

	// SleepWhenConcurrencyLimitsExceeded is a duration to be slept of if
	// the concurrency limit in exceeded (default [when is 0]: don't sleep
	// and accept new connections immediately).
	SleepWhenConcurrencyLimitsExceeded time.Duration

	// Logs all errors, including the most frequent
	// 'connection reset by peer', 'broken pipe' and 'connection timeout'
	// errors. Such errors are common in production serving real-world
	// clients.
	//
	// By default, the most frequent errors such as
	// 'connection reset by peer', 'broken pipe' and 'connection timeout'
	// are suppressed in order to limit output log traffic.
	LogAllErrors bool

	ctxPool    sync.Pool
	readerPool sync.Pool
	bufferPool data.Pool

	// We need to know our listeners and idle connections so we can close them in Shutdown().
	ln []net.Listener

	// Lock for Server
	mu sync.Mutex
	// Lock for Conn
	connMu sync.Mutex

	open int32
	stop int32

	done chan struct{}

	connections uint32

	perIPConnCounter util4go.PerIPConnCounter

	writeEventChain *routine.CyclicDistributionEventChain[writeEventChannelObject]
}

type writeEventChannelObject struct {
	requestCtx *RequestCtx
	b          []byte
}

type writeEventChannelObjectPool struct {
	pool sync.Pool
}

var writeEventObjectPool = writeEventChannelObjectPool{}

func (wp *writeEventChannelObjectPool) releaseObject(object *writeEventChannelObject) {
	object.requestCtx = nil
	object.b = nil
	wp.pool.Put(object)
}

func (wp *writeEventChannelObjectPool) getObject(ctx *RequestCtx, b []byte) *writeEventChannelObject {
	obj := wp.pool.Get()

	if obj == nil {
		return &writeEventChannelObject{
			requestCtx: ctx,
			b:          b,
		}
	}

	rtn := obj.(*writeEventChannelObject)

	rtn.requestCtx = ctx
	rtn.b = b

	return rtn
}

// Serve blocks until the given listener returns permanent error.
func (s *server) Serve(ln net.Listener) error {
	Logger.Debug("Listen on %v", ln.Addr())

	var lastPerIPErrorTime time.Time
	var lastOverflowErrorTime time.Time
	maxWorkersCount := s.Concurrency

	s.mu.Lock()
	{
		s.ln = append(s.ln, ln)

		if s.done == nil {
			s.done = make(chan struct{})
		}
	}
	s.mu.Unlock()

	var err error

	wp, err := routine.NewPool[net.Conn](routine.WithLogAllErrors(s.LogAllErrors),
		routine.WithMaxIdleWorkerDuration(s.MaxIdleWorkerDuration), routine.WithMaxWorkersCount(maxWorkersCount))

	if err != nil {
		return err
	}

	if s.writeEventChain == nil {
		s.writeEventChain = routine.NewCyclicDistributionEventChain[writeEventChannelObject](s.WriterEventChannelSize)
	}

	wp.Start()

	s.writeEventChain.Start(routine.EventHander[writeEventChannelObject](func(job routine.EventChannel[writeEventChannelObject],
		t *writeEventChannelObject) error {
		_, err := t.requestCtx.Write(t.b)
		// writeEventObjectPool.releaseObject(t)
		return err

		//totalN := len(t.b)
		//write := 0
		//for {
		//	n, err := t.conn.Write(t.b[write:])
		//	if err != nil {
		//		return err
		//	}
		//	write += n
		//
		//	if write >= totalN {
		//		return nil
		//	}
		//}
		// return nil
	}))

	// Count our waiting to accept a connection as an open connection.
	// This way we can't get into any weird state where just after accepting
	// a connection Shutdown is called which reads open as 0 because it isn't
	// incremented yet.
	atomic.AddInt32(&s.open, 1)
	defer atomic.AddInt32(&s.open, -1)

	for {
		var c net.Conn
		if c, err = acceptConn(s, ln, &lastPerIPErrorTime); err != nil {
			wp.Stop()
			s.writeEventChain.Stop()
			if err == io.EOF {
				return nil
			}
			return err
		}

		Logger.Debug("Accept a client %v at %v", c.RemoteAddr(), c.LocalAddr())
		atomic.AddInt32(&s.open, 1)
		if !wp.Submit(routine.WorkerFunc(func() error {
			return s.serveConn(c)
		})) {
			atomic.AddInt32(&s.open, -1)
			ctx := s.acquireRequestCtx(c)
			s.wrapContext(c, ctx)

			s.OnReadError(ctx, ManyConcurrency)
			s.OnClose(ctx, ManyConcurrency)
			s.releaseCtx(ctx)

			c.Close()

			if time.Since(lastOverflowErrorTime) > time.Minute {
				Logger.Error("The incoming connection cannot be served, because %d concurrent connections are served. "+
					"Try increasing Server.Concurrency", maxWorkersCount)
				lastOverflowErrorTime = time.Now()
			}

			// The current server reached concurrency limit,
			// so give other concurrently running servers a chance
			// accepting incoming connections on the same address.
			//
			// There is a hope other servers didn't reach their
			// concurrency limits yet :)
			if s.SleepWhenConcurrencyLimitsExceeded > 0 {
				time.Sleep(s.SleepWhenConcurrencyLimitsExceeded)
			}
		}
	}

	s.writeEventChain.Stop()
	wp.Stop()

	return nil
}

var DefaultByteDataHandler = func(ctx *RequestCtx, nread int) error {
	if ctx.s.ByteBufferDecoder != nil {
		bytesArray, err := ctx.s.ByteBufferDecoder(ctx.Bytebuffer, nread)

		if err != nil {
			return err
		}

		if bytesArray != nil {
			for _, one := range *bytesArray {
				if err1 := ctx.s.OnData(ctx, one); err1 != nil {
					return err1
				}
			}
		}
	}

	return nil
}

var DefaultRequestHandler = func(ctx *RequestCtx) {
	var readCount int64 = 0
	var maxReadBytesPerConn int64 = ctx.s.MaxReadBytesPerConn
	var maxPackageFrameSize uint = ctx.s.MaxPackageFrameSize

	b := ctx.acquireReader()
	defer ctx.releaseReader(b)

	//b := make([]byte, ctx.s.ReadBufferSize)

	for {
		//
		if ctx.s.ReadTimeout > 0 {
			ctx.c.SetReadDeadline(time.Now().Add(ctx.s.ReadTimeout))
		}

		nread, err := ctx.c.Read(b)

		atomic.AddInt64(&readCount, int64(nread))

		if err == io.EOF {
			ctx.CloseConn(nil)
			//p.Put(b)
			break
		}

		if nread > 0 {

			if uint(ctx.Bytebuffer.Len()) >= maxPackageFrameSize {
				Logger.Warning("the length of byte buffer is exceed of maxPackageFrameSize[%v], connection is reset",
					maxPackageFrameSize)
				ctx.Bytebuffer.Reset()
				ctx.CloseConn(ByteBufferExceedMaxPackageSize)
			}

			ctx.Bytebuffer.Write(b[:nread])

			err = ctx.s.ByteDataHandler(ctx, nread)

			//ctx.c.Write(append([]byte{}, b[:nread]...))
		} else {
			//netErr, ok := err.(net.Error); ok && netErr.Timeout()
			if ctx.s.ReadTimeout > 0 && err != nil {
				ctx.CloseConn(err)
				//p.Put(b)
				break
			}
		}

		if err != nil {
			ctx.Error(err)
			break
		}

		if maxReadBytesPerConn > 0 {
			if readCount > maxReadBytesPerConn {
				//ctx.s.OnClose(ctx, ManyBytesPerConnection)
				//ctx.c.Close()
				ctx.Error(ManyBytesPerConnection)
				break
			}
		}
	}
}

func (s *server) serveConn(c net.Conn) (err error) {

	requestCtx := s.acquireRequestCtx(c)
	defer s.serveConnCleanup()
	defer s.releaseCtx(requestCtx)

	s.initContext(c, requestCtx)

	if s.OnConnect(requestCtx) {
		s.Handler(requestCtx)
	} else {
		requestCtx.CloseConn(nil)
	}

	return nil
}

func (s *server) initContext(c net.Conn, requestCtx *RequestCtx) {
	connTime := time.Now()
	connID := nextConnID()
	requestCtx.connTime = connTime
	requestCtx.connID = connID
	requestCtx.isTLS = requestCtx.IsTLS()
	requestCtx.s = s
	requestCtx.remoteAddr = c.RemoteAddr()
	requestCtx.Bytebuffer = s.bufferPool.Get()
}

func (s *server) wrapContext(c net.Conn, requestCtx *RequestCtx) {
	connTime := time.Now()
	requestCtx.connTime = connTime
	requestCtx.connID = 0
	requestCtx.isTLS = requestCtx.IsTLS()
	requestCtx.s = s
	requestCtx.remoteAddr = c.RemoteAddr()
	requestCtx.Bytebuffer = s.bufferPool.Get()
}

func (s *server) closeConn() {
	s.connMu.Lock()
	// close
	s.connMu.Unlock()
}

// Shutdown gracefully shuts down the server without interrupting any active connections.
// Shutdown works by first closing all open listeners and then waiting indefinitely for all connections to return to idle and then shut down.
//
// When Shutdown is called, Serve, ListenAndServe, and ListenAndServeTLS immediately return nil.
// Make sure the program doesn't exit and waits instead for Shutdown to return.
//
// Shutdown does not close keepalive connections so its recommended to set ReadTimeout and IdleTimeout to something else than 0.
func (s *server) Shutdown() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	atomic.StoreInt32(&s.stop, 1)
	defer atomic.StoreInt32(&s.stop, 0)

	if s.ln == nil {
		return nil
	}

	for _, ln := range s.ln {
		if ln != nil {
			if err := ln.Close(); err != nil {
				return err
			}
		}
	}

	if s.done != nil {
		close(s.done)
	}

	s.closeConn()

	// Closing the listener will make Serve() call Stop on the worker pool.
	// Setting .stop to 1 will make serveConn() break out of its loop.
	// Now we just have to wait until all workers are done.
	for {
		if open := atomic.LoadInt32(&s.open); open == 0 {
			break
		}
		// This is not an optimal solution but using a sync.WaitGroup
		// here causes data races as it's hard to prevent Add() to be called
		// while Wait() is waiting.
		time.Sleep(time.Millisecond * 100)
	}

	s.done = nil
	s.ln = nil
	return nil
}

func (s *server) serveConnCleanup() {
	atomic.AddInt32(&s.open, -1)
	atomic.AddUint32(&s.connections, ^uint32(0))
}

// GetOpenConnectionsCount returns a number of opened connections.
//
// This function is intended be used by monitoring systems
func (s *server) GetOpenConnectionsCount() int32 {
	if atomic.LoadInt32(&s.stop) == 0 {
		// Decrement by one to avoid reporting the extra open value that gets
		// counted while the server is listening.
		return atomic.LoadInt32(&s.open) - 1
	}
	// This is not perfect, because s.stop could have changed to zero
	// before we load the value of s.open. However, in the common case
	// this avoids underreporting open connections by 1 during server shutdown.
	return atomic.LoadInt32(&s.open)
}

func acceptConn(s *server, ln net.Listener, lastPerIPErrorTime *time.Time) (net.Conn, error) {
	for {
		c, err := ln.Accept()
		if err != nil {
			if c != nil {
				panic("BUG: net.Listener returned non-nil conn and non-nil error")
			}
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				Logger.Error("Timeout error when accepting new connections: %v", netErr)
				time.Sleep(time.Second)
				continue
			}
			if err != io.EOF && !strings.Contains(err.Error(), "use of closed network connection") {
				Logger.Error("Permanent error when accepting new connections: %v", err)
				return nil, err
			}
			return nil, io.EOF
		}
		if c == nil {
			panic("BUG: net.Listener returned (nil, nil)")
		}
		if s.MaxConnsPerIP > 0 {
			pic := wrapPerIPConn(s, c)
			if pic == nil {
				if time.Since(*lastPerIPErrorTime) > time.Minute {
					Logger.Error("The number of connections from %s exceeds MaxConnsPerIP=%d",
						util4go.GetConnIP4(c), s.MaxConnsPerIP)
					*lastPerIPErrorTime = time.Now()
				}
				continue
			}
			c = pic
		}
		return c, nil
	}
}

func wrapPerIPConn(s *server, c net.Conn) net.Conn {
	ip := util4go.GetUint32IP(c)
	if ip == 0 {
		return c
	}
	n := s.perIPConnCounter.Register(ip)
	if n > s.MaxConnsPerIP {
		s.perIPConnCounter.Unregister(ip)
		ctx := s.acquireRequestCtx(c)
		s.OnReadError(ctx, ManyRequests)
		s.OnClose(ctx, ManyRequests)
		s.initContext(c, ctx)

		c.Close()
		return nil
	}

	return util4go.AcquirePerIPConn(c, ip, &s.perIPConnCounter)
}

func (s *server) acquireRequestCtx(c net.Conn) (ctx *RequestCtx) {
	v := s.ctxPool.Get()
	if v == nil {
		ctx = new(RequestCtx)
	} else {
		ctx = v.(*RequestCtx)
	}

	ctx.c = c
	if ch, err := s.writeEventChain.BorrowOne(); err != nil {
		Logger.Warning("acquireRequestCtx error : %v", err)
	} else {
		ctx.eventChannel = ch
	}

	return ctx
}

func (s *server) releaseCtx(ctx *RequestCtx) {
	ctx.reset()
	s.ctxPool.Put(ctx)
}

func (ctx *RequestCtx) acquireReader() []byte {
	v := ctx.s.readerPool.Get()
	if v == nil {
		n := ctx.s.ReadBufferSize
		if n <= 0 {
			n = defaultReadBufferSize
		}
		return make([]byte, n)
	}
	r := v.([]byte)
	return r
}

func (ctx *RequestCtx) releaseReader(r []byte) {
	ctx.s.readerPool.Put(r)
}
