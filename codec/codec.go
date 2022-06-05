package codec

import (
	"errors"
	"github.com/gohutool/boot4go-util/data"
)

/**
* golang-sample源代码，版权归锦翰科技（深圳）有限公司所有。
* <p>
* 文件名称 : codec.go
* 文件路径 :
* 作者 : DavidLiu
× Email: david.liu@ginghan.com
*
* 创建日期 : 2022/6/5 16:13
* 修改历史 : 1. [2022/6/5 16:13] 创建文件 by LongYong
*/

var (
	DelimiterIsEmpty = errors.New("delimiter is Empty")
)

type Decoder[S, T any] func(s *S) (*T, error)

var DelimiterBasedFrameDecoder = func(delimiter []byte) (Decoder[data.ByteBuffer, []byte], error) {
	delimiterLen := len(delimiter)
	if delimiterLen == 0 {
		return nil, nil
	}

	return Decoder[data.ByteBuffer, []byte](func(bytebuffer *data.ByteBuffer) (*[]byte, error) {
		if bytebuffer.Len() <= delimiterLen {
			return nil, nil
		} else {

		}
		return nil, nil
	}), nil
}
