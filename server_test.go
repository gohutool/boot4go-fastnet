package fastnet

import (
	"fmt"
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
	//handler := OnData(func(ctx *RequestCtx, b []byte) error {
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

	handler := OnData(func(ctx *RequestCtx, nread int) error {
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
			fmt.Printf("%v\n", err)
		}
	})

	go func() {
		http.ListenAndServe("0.0.0.0:8887", nil)
	}()

	s := NewServer(WithMaxIdleWorkerDuration(10 * time.Second))
	s.OnData = handler
	s.OnClose = OnClose
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
	var clientNum = 1
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