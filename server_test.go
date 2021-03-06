package fastnet

import (
	"fmt"
	"github.com/gohutool/boot4go-fastnet/codec"
	routine "github.com/gohutool/boot4go-routine"
	"github.com/gohutool/boot4go-util/data"
	"io"
	"net"
	"net/http"
	_ "net/http/pprof"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

/**
* golang-sample源代码，版权归锦翰科技（深圳）有限公司所有。
* <p>
* 文件名称 : server_test.go
* 文件路径 :
* 作者 : DavidLiu
× Email: david.liu@ginghan.com
*
* 创建日期 : 2022/6/2 16:10
* 修改历史 : 1. [2022/6/2 16:10] 创建文件 by LongYong
*/

func TestBase(t *testing.T) {

	l, err := net.Listen("tcp", ":9888")
	if err != nil {
		fmt.Println("Start server error " + err.Error())
		return
	}

	var readCount int64 = 0
	var sentCount int64 = 0

	//p := data.Pool{DefaultSize: 1024}
	handler := RequestHandler(func(ctx *RequestCtx) {
		for {
			//b := p.Get()
			b := make([]byte, 1024)
			nread, err := ctx.c.Read(b)

			//nread, err := ctx.c.Read(b.B)

			atomic.AddInt64(&readCount, 1)
			//fmt.Printf("IO read %v  %v %v \n", readCount, nread, err)

			if err == io.EOF {
				ctx.s.OnClose(ctx, nil)
				ctx.c.Close()
				//p.Put(b)
				break
			}

			if nread > 0 {
				cnt := atomic.AddInt64(&sentCount, 1)
				n, _ := ctx.c.Write(append([]byte{}, b[:nread]...))
				Logger.Debug("recv %v %v\n", cnt, n)
			}

			if err != nil {
				if netErr, ok := err.(*net.OpError); ok && netErr.Timeout() {
					Logger.Debug("recv timeout  %v \n", err)
				} else {
					Logger.Debug("recv error  %v \n", err)
				}
				//p.Put(b)
				break
			}

			//p.Put(b)
		}
	})

	go func() {
		http.ListenAndServe("0.0.0.0:8887", nil)
	}()

	s := server{Handler: handler, MaxIdleWorkerDuration: 10 * time.Second}
	err = s.Serve(l)

	if err != nil {
		panic(err)
	}
}

func TestOnDataBase(t *testing.T) {

	l, err := net.Listen("tcp", ":9888")
	if err != nil {
		fmt.Println("Start server error " + err.Error())
		return
	}

	var readCount int64 = 0
	var sentCount int64 = 0

	//p := data.Pool{DefaultSize: 1024}
	//
	//handler := ByteDataHandler(func(ctx *RequestCtx, b []byte) error {
	//	atomic.AddInt64(&readCount, 1)
	//	nread := len(b)
	//
	//	if nread > 0 {
	//		cnt := atomic.AddInt64(&sentCount, 1)
	//		n, _ := ctx.c.Write(b)
	//		Logger.Debug("recv %v %v\n", cnt, n)
	//	}
	//
	//	return nil
	//})

	handler := ByteDataHandler(func(ctx *RequestCtx, nread int) error {
		atomic.AddInt64(&readCount, 1)
		nread = len(ctx.Bytebuffer.Bytes())

		if nread > 0 {
			cnt := atomic.AddInt64(&sentCount, 1)
			n, _ := ctx.Bytebuffer.Compact(ctx.c, 0)

			if int64(nread) != n {
				fmt.Printf("#########\n")
			}

			Logger.Debug("recv %v %v\n", cnt, n)
		}

		return nil
	})

	OnClose := OnClose(func(ctx *RequestCtx, err error) {
		if err != nil {
			fmt.Printf("Close with %v\n", err)
		}
	})

	go func() {
		http.ListenAndServe("0.0.0.0:8887", nil)
	}()

	s := NewServer(WithMaxIdleWorkerDuration(10 * time.Second))
	s.ByteDataHandler = handler
	s.OnClose = OnClose
	//s.ReadTimeout = 10 * time.Second

	err = s.Serve(l)

	if err != nil {
		panic(err)
	}
}

func TestOnDataBaseWithWriteChannel(t *testing.T) {

	l, err := net.Listen("tcp", ":9888")
	if err != nil {
		fmt.Println("Start server error " + err.Error())
		return
	}

	var readCount int64 = 0
	var sentCount int64 = 0

	dataEventHandle := DataEventHandle(func(data any, job routine.EventChannel[any]) {
		if data != nil {
			one := data.(struct {
				c net.Conn
				b []byte
			})
			if one.c != nil && one.b != nil {
				one.c.Write(one.b)
			}
		}
	})

	handler := ByteDataHandler(func(ctx *RequestCtx, nread int) error {
		atomic.AddInt64(&readCount, 1)
		nread = len(ctx.Bytebuffer.Bytes())

		if nread > 0 {
			cnt := atomic.AddInt64(&sentCount, 1)
			b := ctx.Bytebuffer.Flip(nread)
			//ctx.Write(b)
			ctx.PushEventData(struct {
				c net.Conn
				b []byte
			}{
				c: ctx.GetConn(),
				b: b,
			})

			Logger.Debug("recv %v\n", cnt)
			//n, _ := ctx.Write(b)
			//
			//if nread != n {
			//	fmt.Printf("#########\n")
			//}

			//Logger.Debug("recv %v %v\n", cnt, n)
		}

		return nil
	})

	OnClose := OnClose(func(ctx *RequestCtx, err error) {
		if err != nil {
			fmt.Printf("Close with %v\n", err)
		}
	})

	go func() {
		http.ListenAndServe("0.0.0.0:8887", nil)
	}()

	s := NewServer(WithMaxIdleWorkerDuration(10 * time.Second))
	s.ByteDataHandler = handler
	s.DataEventHandle = dataEventHandle
	s.OnClose = OnClose
	//s.ReadTimeout = 10 * time.Second

	err = s.Serve(l)

	if err != nil {
		panic(err)
	}
}

func TestDelimiterDecoderBase(t *testing.T) {

	l, err := net.Listen("tcp", ":9888")
	if err != nil {
		fmt.Println("Start server error " + err.Error())
		return
	}

	decoder, err := codec.DelimiterBasedFrameDecoder([]byte("\n"))

	if err != nil {
		panic(err)
	}

	handler := ByteDataHandler(func(ctx *RequestCtx, nread int) error {
		bytesArray, err := decoder(ctx.Bytebuffer)

		if err != nil {
			return err
		}

		if bytesArray != nil {
			for _, one := range *bytesArray {
				ctx.c.Write(one)
			}
		}

		return nil
	})

	OnClose := OnClose(func(ctx *RequestCtx, err error) {
		if err != nil {
			fmt.Printf("%v\n", err)
		}
	})

	go func() {
		http.ListenAndServe("0.0.0.0:8887", nil)
	}()

	s := NewServer(WithMaxIdleWorkerDuration(10 * time.Second))
	s.ByteDataHandler = handler
	s.OnClose = OnClose
	err = s.Serve(l)

	if err != nil {
		panic(err)
	}
}

func TestStringLineDecoderBase(t *testing.T) {
	l, err := net.Listen("tcp", ":9888")
	if err != nil {
		fmt.Println("Start server error " + err.Error())
		return
	}

	decoder, err := codec.LineBasedFrameDecoder()

	if err != nil {
		panic(err)
	}

	handler := ByteDataHandler(func(ctx *RequestCtx, nread int) error {
		strs, err := decoder(ctx.Bytebuffer)

		if err != nil {
			return err
		}

		if strs != nil {
			for _, one := range *strs {
				fmt.Println(one)
				ctx.c.Write([]byte(one))
			}
		}

		return nil
	})

	OnClose := OnClose(func(ctx *RequestCtx, err error) {
		if err != nil {
			fmt.Printf("%v\n", err)
		}
	})

	go func() {
		http.ListenAndServe("0.0.0.0:8887", nil)
	}()

	s := NewServer(WithMaxIdleWorkerDuration(10 * time.Second))
	s.ByteDataHandler = handler
	s.OnClose = OnClose
	err = s.Serve(l)

	if err != nil {
		panic(err)
	}
}

func TestFixedLengthFrameDecoderBase(t *testing.T) {

	l, err := net.Listen("tcp", ":9888")
	if err != nil {
		fmt.Println("Start server error " + err.Error())
		return
	}

	decoder, err := codec.FixedLengthFrameDecoder(20)

	if err != nil {
		panic(err)
	}

	handler := ByteDataHandler(func(ctx *RequestCtx, nread int) error {
		bytesArray, err := decoder(ctx.Bytebuffer)

		if err != nil {
			return err
		}

		if bytesArray != nil {
			for _, one := range *bytesArray {
				ctx.c.Write(one)
			}
		}

		return nil
	})

	OnClose := OnClose(func(ctx *RequestCtx, err error) {
		if err != nil {
			fmt.Printf("%v\n", err)
		}
	})

	go func() {
		http.ListenAndServe("0.0.0.0:8887", nil)
	}()

	s := NewServer(WithMaxIdleWorkerDuration(10 * time.Second))
	s.ByteDataHandler = handler
	s.OnClose = OnClose
	err = s.Serve(l)

	if err != nil {
		panic(err)
	}
}

func TestFixLengthFieldFrameDecoderBase(t *testing.T) {

	l, err := net.Listen("tcp", ":9888")
	if err != nil {
		fmt.Println("Start server error " + err.Error())
		return
	}

	decoder, err := codec.FixLengthFieldFrameDecoder(codec.BIGENDIAN, codec.INT16)

	if err != nil {
		panic(err)
	}

	handler := ByteDataHandler(func(ctx *RequestCtx, nread int) error {
		bytesArray, err := decoder(ctx.Bytebuffer)

		if err != nil {
			return err
		}

		if bytesArray != nil {
			for _, one := range *bytesArray {
				ctx.c.Write(one)
			}
		}

		return nil
	})

	OnClose := OnClose(func(ctx *RequestCtx, err error) {
		if err != nil {
			fmt.Printf("%v\n", err)
		}
	})

	go func() {
		http.ListenAndServe("0.0.0.0:8887", nil)
	}()

	s := NewServer(WithMaxIdleWorkerDuration(10 * time.Second))
	s.ByteDataHandler = handler
	s.OnClose = OnClose
	err = s.Serve(l)

	if err != nil {
		panic(err)
	}
}

func TestLengthFieldBaseFrameDecoderBase(t *testing.T) {

	l, err := net.Listen("tcp", ":9888")
	if err != nil {
		fmt.Println("Start server error " + err.Error())
		return
	}

	// 矫正的值为：包长 - 长度域的值 – 长度域偏移 – 长度域长。
	//  0 =  包长(14) - 长度域的值(10) – 长度域偏移(2) – 长度域长(2)。
	decoder, err := codec.LengthFieldBasedFrameDecoder(codec.BIGENDIAN, codec.INT16,
		1024*1024, 2, 0, 4)

	if err != nil {
		panic(err)
	}

	handler := ByteDataHandler(func(ctx *RequestCtx, nread int) error {
		bytesArray, err := decoder(ctx.Bytebuffer)

		if err != nil {
			return err
		}

		if bytesArray != nil {
			for _, one := range *bytesArray {
				ctx.c.Write(one)
			}
		}

		return nil
	})

	OnClose := OnClose(func(ctx *RequestCtx, err error) {
		if err != nil {
			fmt.Printf("%v\n", err)
		}
	})

	go func() {
		http.ListenAndServe("0.0.0.0:8887", nil)
	}()

	s := NewServer(WithMaxIdleWorkerDuration(10 * time.Second))
	s.ByteDataHandler = handler
	s.OnClose = OnClose
	err = s.Serve(l)

	if err != nil {
		panic(err)
	}
}

func TestVariableLengthFieldFrameDecoderBase(t *testing.T) {

	l, err := net.Listen("tcp", ":9888")
	if err != nil {
		fmt.Println("Start server error " + err.Error())
		return
	}

	// 矫正的值为：包长 - 长度域的值 – 长度域偏移 – 长度域长。
	//  0 =  包长(13) - 长度域的值(10) – 长度域偏移(2) – 长度域长(1)。
	decoder, err := codec.VariableLengthFieldFrameDecoder(
		1024*1024, 2, 0, func(variableLength uint64) uint64 {
			return 2 + variableLength
		})

	if err != nil {
		panic(err)
	}

	d := ByteBufferDecoder(func(bytebuffer *data.ByteBuffer, nread int) (*[][]byte, error) {
		return decoder(bytebuffer)
	})

	//handler := ByteDataHandler(func(ctx *RequestCtx, nread int) error {
	//	bytesArray, err := decoder(ctx.Bytebuffer)
	//
	//	if err != nil {
	//		return err
	//	}
	//
	//	if bytesArray != nil {
	//		for _, one := range *bytesArray {
	//			ctx.Write(one)
	//		}
	//	}
	//
	//	return nil
	//})

	OnClose := OnClose(func(ctx *RequestCtx, err error) {
		if err != nil {
			fmt.Printf("%v\n", err)
		}
	})

	go func() {
		http.ListenAndServe("0.0.0.0:8887", nil)
	}()

	s := NewServer(WithMaxIdleWorkerDuration(10*time.Second),
		WithMaxPackageFrameSize(1024*10))
	//s.ByteDataHandler = handler
	s.OnClose = OnClose
	s.ByteBufferDecoder = d
	s.OnData = OnData(func(ctx *RequestCtx, b []byte) error {
		ctx.Write(b)
		// ctx.WriteToChannel(b)
		return nil
	})

	err = s.Serve(l)

	if err != nil {
		panic(err)
	}
}

func TestNetServer(t *testing.T) {

	l, err := net.Listen("tcp", ":9888")
	if err != nil {
		fmt.Println("Start server error " + err.Error())
		return
	}

	for {
		var c net.Conn
		var err error
		if c, err = l.Accept(); err != nil {
			panic(err)
		}

		go serverConn(c)
	}
}

func serverConn(c net.Conn) {
	b := make([]byte, 1024*4)
	for {
		n, err := c.Read(b)

		if err != nil {
			fmt.Printf("Read %v\n", err)
			c.Close()
			break
		}

		_, err2 := c.Write(b[0:n])

		if err2 != nil {
			fmt.Printf("Write error %v\n", err2)
			c.Close()
			break
		}
	}
}

func TestClient(t *testing.T) {
	var clientNum = 1000
	var msgSize = 1024 * 20

	var wg sync.WaitGroup
	wg.Add(clientNum)

	test := func(id int) {
		c, err := net.Dial("tcp", "127.0.0.1:9888")
		if err != nil {
			fmt.Println("Connect server error " + err.Error())
			return
		}

		var readCount int = 0

		go func() {
			n, err := c.Write(make([]byte, msgSize))
			if err != nil {
				fmt.Printf("send error  %v %v\n", n, err)
			}
		}()

		for {
			b := make([]byte, 1024)
			nread, err := c.Read(b)

			//fmt.Printf("IO read %v  %v %v \n", readCount, nread, err)

			if err == io.EOF {
				break
			}

			if err != nil {
				fmt.Printf("Connect error  %v \n", err)
				break
			}

			readCount = readCount + nread

			if readCount >= msgSize {
				//fmt.Printf("ID[%v] is receive all data %v\n", id, readCount)
				//time.Sleep(10 * time.Second)
				c.Close()
				break
			}

		}

		wg.Done()
	}

	for i := 0; i < clientNum; i++ {
		if runtime.GOOS != "windows" {
			go test(i)
		} else {
			go test(i)
		}
	}

	wg.Wait()
	fmt.Println("Exit")

	//time.Sleep(time.Second)

}

func TestPackageClient(t *testing.T) {
	var clientNum = 1

	var wg sync.WaitGroup
	wg.Add(clientNum)

	text := Test_BIG

	test := func(id int) {
		c, err := net.Dial("tcp", "127.0.0.1:9888")
		if err != nil {
			fmt.Println("Connect server error " + err.Error())
			return
		}

		var readCount int = 0
		var sentCount int = 0
		var loop = 20

		for idx := 0; idx < loop; idx++ {
			b := []byte(fmt.Sprintf(text, id, idx+1))
			sentCount += len(b)
		}

		go func() {

			for idx := 0; idx < loop; idx++ {

				b := []byte(fmt.Sprintf(text, id, idx+1))

				lenData := codec.PackFieldLength(codec.BIGENDIAN, codec.INT16, int64(len(b)))
				lenData = append(lenData, b...)

				n, err := c.Write(lenData)

				if err != nil {
					fmt.Printf("send error  %v %v\n", n, err)
				}
			}
		}()

		for {
			b := make([]byte, 1024)
			nread, err := c.Read(b)

			if err == io.EOF {
				break
			}

			if err != nil {
				fmt.Printf("Connect error  %v \n", err)
				break
			}

			readCount = readCount + nread
			fmt.Printf("ID[%v] is receive data %v\n", id, string(b[0:nread]))

			if readCount >= sentCount {
				c.Close()
				break
			}

		}

		wg.Done()
	}

	for i := 0; i < clientNum; i++ {
		if runtime.GOOS != "windows" {
			go test(i)
		} else {
			go test(i)
		}
	}

	wg.Wait()
	fmt.Println("Exit")

	//time.Sleep(time.Second)

}

func TestFixLengthFieldPackageClient(t *testing.T) {
	var clientNum = 1

	var wg sync.WaitGroup
	wg.Add(clientNum)

	test := func(id int) {
		c, err := net.Dial("tcp", "127.0.0.1:9888")
		if err != nil {
			fmt.Println("Connect server error " + err.Error())
			return
		}

		var readCount int = 0
		var sentCount int = 0
		var loop = 20

		for idx := 0; idx < loop; idx++ {
			b := []byte(fmt.Sprintf("Hello world index[%v] Loop=%v", id, idx+1))
			sentCount += len(b)
		}

		go func() {

			for idx := 0; idx < loop; idx++ {

				b := []byte(fmt.Sprintf("Hello world index[%v] Loop=%v", id, idx+1))

				lenData := codec.PackFieldLength(codec.BIGENDIAN, codec.INT16, int64(len(b)))
				lenData = append(lenData, b...)

				n, err := c.Write(lenData)

				if err != nil {
					fmt.Printf("send error  %v %v\n", n, err)
				}
			}
		}()

		for {
			b := make([]byte, 1024)
			nread, err := c.Read(b)

			if err == io.EOF {
				break
			}

			if err != nil {
				fmt.Printf("Connect error  %v \n", err)
				break
			}

			readCount = readCount + nread
			fmt.Printf("ID[%v] is receive data %v\n", id, string(b[0:nread]))

			if readCount >= sentCount {
				c.Close()
				break
			}

		}

		wg.Done()
	}

	for i := 0; i < clientNum; i++ {
		if runtime.GOOS != "windows" {
			go test(i)
		} else {
			go test(i)
		}
	}

	wg.Wait()
	fmt.Println("Exit")

	//time.Sleep(time.Second)

}

var (
	Test_SMALL  = "Hello world index[%v] Loop=%v"
	Test_MIDDLE = "Hello world index[%v] Loop=%v Hello world index  Hello world index " +
		"Hello world index Hello world index Hello world index Hello world index  Hello world index " +
		"Hello world index Hello world index Hello world index Hello world index  Hello world index " +
		"Hello world index Hello world index Hello world index Hello world index  Hello world index " +
		"Hello world index Hello world index Hello world index Hello world index  Hello world index " +
		"Hello world index Hello world index Hello world index Hello world index  Hello world index " +
		"Hello world index Hello world index Hello world index Hello world index  Hello world index " +
		"Hello world index Hello world index Hello world index Hello world index  Hello world index " +
		"Hello world index Hello world index Hello world index Hello world index  Hello world index " +
		"Hello world index Hello world index Hello world index"

	Test_BIG = "Hello world index[%v] Loop=%v Hello world index  Hello world index " +
		"Hello world index Hello world index Hello world index Hello world index  Hello world index " +
		"Hello world index Hello world index Hello world index Hello world index  Hello world index " +
		"Hello world index Hello world index Hello world index Hello world index  Hello world index " +
		"Hello world index Hello world index Hello world index Hello world index  Hello world index " +
		"Hello world index Hello world index Hello world index Hello world index  Hello world index " +
		"Hello world index Hello world index Hello world index Hello world index  Hello world index " +
		"Hello world index Hello world index Hello world index Hello world index  Hello world index " +
		"Hello world index Hello world index Hello world index Hello world index  Hello world index " +
		"Hello world index Hello world index Hello world index Hello world index  Hello world index " +
		"Hello world index Hello world index Hello world index Hello world index  Hello world index " +
		"Hello world index Hello world index Hello world index Hello world index  Hello world index " +
		"Hello world index Hello world index Hello world index Hello world index  Hello world index " +
		"Hello world index Hello world index Hello world index Hello world index  Hello world index " +
		"Hello world index Hello world index Hello world index Hello world index  Hello world index " +
		"Hello world index Hello world index Hello world index Hello world index  Hello world index " +
		"Hello world index Hello world index Hello world index Hello world index  Hello world index " +
		"Hello world index Hello world index Hello world index Hello world index  Hello world index " +
		"Hello world index Hello world index Hello world index Hello world index  Hello world index " +
		"Hello world index Hello world index Hello world index Hello world index  Hello world index " +
		"Hello world index Hello world index Hello world index Hello world index  Hello world index " +
		"Hello world index Hello world index Hello world index Hello world index  Hello world index " +
		"Hello world index Hello world index Hello world index Hello world index  Hello world index " +
		"Hello world index Hello world index Hello world index Hello world index  Hello world index " +
		"Hello world index Hello world index Hello world index Hello world index  Hello world index " +
		"Hello world index Hello world index Hello world index Hello world index  Hello world index " +
		"Hello world index Hello world index Hello world index Hello world index  Hello world index " +
		"Hello world index Hello world index Hello world index Hello world index  Hello world index " +
		"Hello world index Hello world index Hello world index Hello world index  Hello world index " +
		"Hello world index Hello world index Hello world index Hello world index  Hello world index " +
		"Hello world index Hello world index Hello world index Hello world index  Hello world index " +
		"Hello world index Hello world index Hello world index Hello world index  Hello world index " +
		"Hello world index Hello world index Hello world index Hello world index  Hello world index " +
		"Hello world index Hello world index Hello world index Hello world index  Hello world index " +
		"Hello world index Hello world index Hello world index Hello world index  Hello world index " +
		"Hello world index Hello world index Hello world index Hello world index  Hello world index " +
		"Hello world index Hello world index Hello world index Hello world index  Hello world index " +
		"Hello world index Hello world index Hello world index Hello world index  Hello world index " +
		"Hello world index Hello world index Hello world index Hello world index  Hello world index " +
		"Hello world index Hello world index Hello world index Hello world index  Hello world index " +
		"Hello world index Hello world index Hello world index Hello world index  Hello world index " +
		"Hello world index Hello world index Hello world index"
)

func TestVariableLengthFieldPackageClient(t *testing.T) {
	var clientNum = 100

	var wg sync.WaitGroup
	wg.Add(clientNum)

	//text := Test_SMALL
	//text := Test_MIDDLE
	text := Test_BIG

	test := func(id int) {
		c, err := net.Dial("tcp", "127.0.0.1:9888")
		if err != nil {
			fmt.Println("Connect server error " + err.Error())
			return
		}

		var readCount int = 0
		var sentCount int = 0
		var loop = 30

		for idx := 0; idx < loop; idx++ {
			b := []byte(fmt.Sprintf(text, id, idx+1))
			sentCount += len(b)
		}

		go func() {

			for idx := 0; idx < loop; idx++ {

				b := []byte(fmt.Sprintf(text, id, idx+1))

				lenData := codec.PackVariableLength(uint64(len(b)))
				lenData = append(lenData, b...)
				bs := append([]byte{}, []byte{0, 0}...)
				bs = append(bs, lenData...)

				n, err := c.Write(bs)

				if err != nil {
					fmt.Printf("send error  %v %v\n", n, err)
				}
			}
		}()

		for {
			b := make([]byte, 1024)
			nread, err := c.Read(b)

			if err == io.EOF {
				break
			}

			if err != nil {
				fmt.Printf("Connect error  %v \n", err)
				break
			}

			readCount = readCount + nread
			//fmt.Printf("ID[%v] is receive data %v\n", id, string(b[0:nread]))

			if readCount >= sentCount {
				c.Close()
				break
			}

		}

		wg.Done()
	}

	for i := 0; i < clientNum; i++ {
		if runtime.GOOS != "windows" {
			go test(i)
		} else {
			go test(i)
		}
	}

	wg.Wait()
	fmt.Println("Exit")

	//time.Sleep(time.Second)

}

func TestLengthFieldBasePackageClient(t *testing.T) {
	var clientNum = 1000

	var wg sync.WaitGroup
	wg.Add(clientNum)

	test := func(id int) {
		c, err := net.Dial("tcp", "127.0.0.1:9888")
		if err != nil {
			fmt.Println("Connect server error " + err.Error())
			return
		}

		var readCount int = 0
		var sentCount int = 0
		var loop = 20

		for idx := 0; idx < loop; idx++ {
			b := []byte(fmt.Sprintf("Hello world index[%v] Loop=%v", id, idx+1))
			sentCount += len(b)
		}

		go func() {

			for idx := 0; idx < loop; idx++ {

				b := []byte(fmt.Sprintf("Hello world index[%v] Loop=%v", id, idx+1))

				lenData := codec.PackFieldLength(codec.BIGENDIAN, codec.INT16, int64(len(b)))
				lenData = append(lenData, b...)
				bs := append([]byte{}, []byte{0, 0}...)
				bs = append(bs, lenData...)

				n, err := c.Write(bs)

				if err != nil {
					fmt.Printf("send error  %v %v\n", n, err)
				}
			}
		}()

		for {
			b := make([]byte, 1024)
			nread, err := c.Read(b)

			if err == io.EOF {
				break
			}

			if err != nil {
				fmt.Printf("Connect error  %v \n", err)
				break
			}

			readCount = readCount + nread
			fmt.Printf("ID[%v] is receive data %v\n", id, string(b[0:nread]))

			if readCount >= sentCount {
				c.Close()
				break
			}

		}

		wg.Done()
	}

	for i := 0; i < clientNum; i++ {
		if runtime.GOOS != "windows" {
			go test(i)
		} else {
			go test(i)
		}
	}

	wg.Wait()
	fmt.Println("Exit")

	//time.Sleep(time.Second)

}
