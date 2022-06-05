package codec

import (
	"bytes"
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

var DelimiterBasedFrameDecoder = func(delimiter []byte) (Decoder[data.ByteBuffer, [][]byte], error) {
	delimiterLen := len(delimiter)
	if delimiterLen == 0 {
		return nil, nil
	}

	return Decoder[data.ByteBuffer, [][]byte](func(bytebuffer *data.ByteBuffer) (*[][]byte, error) {
		if bytebuffer.Len() <= delimiterLen {
			return nil, nil
		} else {
			idx := -1

			idx = bytes.LastIndex(bytebuffer.Bytes(), delimiter)

			if idx >= 0 {
				if idx == 0 {
					bytebuffer.Flip(delimiterLen)
					return nil, nil
				} else {
					segments := bytebuffer.Flip(idx + delimiterLen)

					rtn := make([][]byte, 0)

					for {
						idx := bytes.Index(segments, delimiter)

						if idx < 0 {
							break
						}

						b1 := segments[0:idx]
						rtn = append(rtn, b1)

						segments = segments[idx+delimiterLen:]

						if len(segments) == 0 {
							break
						}
					}

					return &rtn, nil
				}
			} else {
				return nil, nil
			}
		}
		return nil, nil
	}), nil
}

var StringLineDecoder = func() (Decoder[data.ByteBuffer, []string], error) {
	codec, err := DelimiterBasedFrameDecoder([]byte("\n"))
	if err != nil {
		return nil, err
	}

	return Decoder[data.ByteBuffer, []string](func(bb *data.ByteBuffer) (*[]string, error) {
		bs, err := codec(bb)
		if err != nil {
			return nil, err
		}

		if len(*bs) > 0 {
			rtn := make([]string, 0)

			for _, one := range *bs {
				rtn = append(rtn, string(one))
			}

			return &rtn, nil

		} else {
			return nil, nil
		}

	}), nil
}
