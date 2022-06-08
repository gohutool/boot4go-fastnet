package fastnet

import (
	"errors"
	"github.com/gohutool/log4go"
	"net"
	"time"
)

/**
* golang-sample源代码，版权归锦翰科技（深圳）有限公司所有。
* <p>
* 文件名称 : const
* 文件路径 :
* 作者 : DavidLiu
× Email: david.liu@ginghan.com
*
* 创建日期 : 2022/6/2 14:14
* 修改历史 : 1. [2022/6/2 14:14] 创建文件 by LongYong
*/

var zeroTCPAddr = &net.TCPAddr{
	IP: net.IPv4zero,
}

const (
	defaultReadBufferSize         = 4 * 1024
	defaultWriteBufferSize        = 4 * 1024
	defaultMaxPackageFrameSize    = 4 * 1024
	defaultWriterEventChannelSize = 1000
)

var zeroTime time.Time

var (
	WorkPoolInitError              = errors.New("encounter error while init worker pool")
	ManyRequests                   = errors.New("The number of connections from your ip exceeds MaxConnsPerIP")
	ManyConcurrency                = errors.New("The connection cannot be served because Server.Concurrency limit exceeded")
	ManyBytesPerConnection         = errors.New("The connection cannot be served because Max bytes limit exceeded")
	ByteBufferExceedMaxPackageSize = errors.New("ByteBuffer is exceed maxPackageSize")
)

var (
	Logger = log4go.LoggerManager.GetLogger("gohutool.boot4go.fastnet")
)

var (
	DummyOnClose = func(ctx *RequestCtx, err error) {

	}

	DummyOnReadError = func(ctx *RequestCtx, err error) bool {
		Logger.Warning("OnReadError : %v ", err)
		return true
	}

	DummyOnWriteError = func(ctx *RequestCtx, err error) bool {
		Logger.Warning("OnWriteError : %v ", err)
		return false
	}

	DummyOnConnect = func(ctx *RequestCtx) bool {
		return true
	}

	DummyOnWrite = func(ctx *RequestCtx, nwrite int) {
	}

	DummyOnData = func(ctx *RequestCtx, b []byte) error {
		return nil
	}
)
