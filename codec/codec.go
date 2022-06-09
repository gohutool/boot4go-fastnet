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
	DelimiterIsEmpty       = errors.New("delimiter is Empty")
	FixLengthInvalid       = errors.New("fix length is invalid")
	VariableLengthOverflow = errors.New("variable length is over flow")
)

type Decoder[S, T any] func(s *S) (*T, error)
type Encoder[S, T any] func(s *S, t *T) error

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

var DelimiterBasedFrameEncoder = func(delimiter []byte) (Encoder[data.ByteBuffer, []byte], error) {

	delimiterLen := len(delimiter)
	if delimiterLen == 0 {
		return nil, nil
	}

	return Encoder[data.ByteBuffer, []byte](func(bytebuffer *data.ByteBuffer, bb *[]byte) error {
		if bb != nil && len(*bb) > 0 {
			bytebuffer.Write(*bb)
			bytebuffer.Write(delimiter)
		}

		return nil

	}), nil
}

var LineBasedFrameDecoder = func() (Decoder[data.ByteBuffer, []string], error) {
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

var LineBasedFrameEncoder = func() (Encoder[data.ByteBuffer, string], error) {
	encoder, err := DelimiterBasedFrameEncoder([]byte("\n"))
	if err != nil {
		return nil, err
	}

	return Encoder[data.ByteBuffer, string](func(bytebuffer *data.ByteBuffer, ss *string) error {
		if ss != nil && len(*ss) > 0 {
			o := []byte(*ss)
			return encoder(bytebuffer, &o)
		}

		return nil
	}), nil
}

var FixedLengthFrameDecoder = func(fixLength int) (Decoder[data.ByteBuffer, [][]byte], error) {

	if fixLength <= 0 {
		return nil, FixLengthInvalid
	}

	return Decoder[data.ByteBuffer, [][]byte](func(bb *data.ByteBuffer) (*[][]byte, error) {
		len := len(bb.Bytes())

		if len <= fixLength {
			return nil, nil
		}

		n := len / fixLength

		bytes := bb.Flip(int(n * fixLength))

		rtn := make([][]byte, 0)

		for idx := 0; idx < n; idx++ {
			rtn = append(rtn, bytes[idx*fixLength:(idx+1)*fixLength])
		}

		return &rtn, nil

	}), nil
}

var FixedLengthFrameEncoder = func(fixLength int) (Encoder[data.ByteBuffer, []byte], error) {

	if fixLength <= 0 {
		return nil, FixLengthInvalid
	}

	return Encoder[data.ByteBuffer, []byte](func(bytebuffer *data.ByteBuffer, bs *[]byte) error {
		if bs != nil && len(*bs) > 0 {
			return nil
		}

		len := len(*bs)

		if len == 0 {
			return nil
		}

		n := (len+fixLength-1/fixLength)*fixLength - len

		bytebuffer.Write(*bs)

		if n > 0 {
			bytebuffer.Write(make([]byte, n, n))
		}

		return nil

	}), nil
}

var FixLengthFieldFrameDecoder = func(em EndianMode, lengthFiledByte LengthFiledByte) (Decoder[data.ByteBuffer, [][]byte], error) {

	len := uint64(8)

	switch lengthFiledByte {
	case INT8:
		len = 1
	case INT16:
		len = 2
	case INT32:
		len = 4
	case INT64:
		len = 8
	}

	// 矫正的值为：包长 - 长度域的值 – 长度域偏移 – 长度域长。
	//  0 =  包长(12) - 长度域的值(10) – 长度域偏移(0) – 长度域长(2)。
	decoder, err := LengthFieldBasedFrameDecoder(em, lengthFiledByte, 1024*1024*100, 0, 0, len)

	if err != nil {
		return nil, err
	}

	return Decoder[data.ByteBuffer, [][]byte](func(bb *data.ByteBuffer) (*[][]byte, error) {
		return decoder(bb)
	}), nil
}

//
//var FixLengthFieldFrameDecoder = func(em EndianMode, lengthFiledByte LengthFiledByte) (Decoder[data.ByteBuffer, [][]byte], error) {
//
//	len := 8
//
//	switch lengthFiledByte {
//	case INT8:
//		len = 1
//	case INT16:
//		len = 2
//	case INT32:
//		len = 4
//	case INT64:
//		len = 8
//	}
//
//	return Decoder[data.ByteBuffer, [][]byte](func(bb *data.ByteBuffer) (*[][]byte, error) {
//		b := bb.Bytes()
//		allLen := bb.Len()
//
//		idx := 0
//
//		var dataLen int64
//		var lenData []byte
//
//		rtn := make([][]byte, 0)
//
//		if allLen <= len {
//			return nil, nil
//		}
//
//		for {
//			lenData = b[idx : idx+len]
//			dataLen = UnpackFieldLength(em, lengthFiledByte, lenData)
//
//			if dataLen > int64(allLen-len-idx) {
//				break
//			} else {
//				rtn = append(rtn, b[idx+len:int64(idx+len)+dataLen])
//
//				idx = idx + len + int(dataLen)
//
//				if idx >= allLen-len {
//					break
//				}
//			}
//		}
//
//		if idx > 0 {
//			bb.Flip(idx)
//			return &rtn, nil
//		}
//
//		return nil, nil
//
//	}), nil
//}

var FixLengthFieldFrameEncoder = func(em EndianMode, lengthFiledByte LengthFiledByte) (Encoder[data.ByteBuffer, []byte], error) {
	return Encoder[data.ByteBuffer, []byte](func(byteBuffer *data.ByteBuffer, bs *[]byte) error {
		if bs != nil && len(*bs) > 0 {
			return nil
		}

		byteBuffer.Write(PackFieldLength(em, lengthFiledByte, int64(len(*bs))))
		byteBuffer.Write(*bs)

		return nil

	}), nil
}

var (
	MaxFrameLengthInvalid      = errors.New("maxFrameLength must be a positive integer")
	LengthFieldOffsetInvalid   = errors.New("lengthFieldOffset must be a non-negative integer")
	InitialBytesToStripInvalid = errors.New("InitialBytesToStrip must be a non-negative integer")
	lengthFieldLengthInvalid   = errors.New("lengthFieldLength must be either 1, 2, 3, 4, or 8")
	MaxFrameLengthSmallInvalid = errors.New("maxFrameLength must be equal to or greater than lengthFieldOffset + lengthFieldLength")
)

/**

Copy from netty's document  io.netty.handler.codec.LengthFieldBasedFrameDecoder

A decoder that splits the received ByteBufs dynamically by the value of the length field in the message. It is particularly useful when you decode a binary message which has an integer header field that represents the length of the message body or the whole message.
LengthFieldBasedFrameDecoder has many configuration parameters so that it can decode any message with a length field, which is often seen in proprietary client-server protocols. Here are some example that will give you the basic idea on which option does what.
2 bytes length field at offset 0, do not strip header
The value of the length field in this example is 12 (0x0C) which represents the length of "HELLO, WORLD". By default, the decoder assumes that the length field represents the number of the bytes that follows the length field. Therefore, it can be decoded with the simplistic parameter combination.
   lengthFieldOffset   = 0
   lengthFieldLength   = 2
   lengthAdjustment    = 0
   initialBytesToStrip = 0 (= do not strip header)

   BEFORE DECODE (14 bytes)         AFTER DECODE (14 bytes)
   +--------+----------------+      +--------+----------------+
   | Length | Actual Content |----->| Length | Actual Content |
   | 0x000C | "HELLO, WORLD" |      | 0x000C | "HELLO, WORLD" |
   +--------+----------------+      +--------+----------------+

2 bytes length field at offset 0, strip header
Because we can get the length of the content by calling ByteBuf.readableBytes(), you might want to strip the length field by specifying initialBytesToStrip. In this example, we specified 2, that is same with the length of the length field, to strip the first two bytes.
   lengthFieldOffset   = 0
   lengthFieldLength   = 2
   lengthAdjustment    = 0
   initialBytesToStrip = 2 (= the length of the Length field)

   BEFORE DECODE (14 bytes)         AFTER DECODE (12 bytes)
   +--------+----------------+      +----------------+
   | Length | Actual Content |----->| Actual Content |
   | 0x000C | "HELLO, WORLD" |      | "HELLO, WORLD" |
   +--------+----------------+      +----------------+

2 bytes length field at offset 0, do not strip header, the length field represents the length of the whole message
In most cases, the length field represents the length of the message body only, as shown in the previous examples. However, in some protocols, the length field represents the length of the whole message, including the message header. In such a case, we specify a non-zero lengthAdjustment. Because the length value in this example message is always greater than the body length by 2, we specify -2 as lengthAdjustment for compensation.
   lengthFieldOffset   =  0
   lengthFieldLength   =  2
   lengthAdjustment    = -2 (= the length of the Length field)
   initialBytesToStrip =  0

   BEFORE DECODE (14 bytes)         AFTER DECODE (14 bytes)
   +--------+----------------+      +--------+----------------+
   | Length | Actual Content |----->| Length | Actual Content |
   | 0x000E | "HELLO, WORLD" |      | 0x000E | "HELLO, WORLD" |
   +--------+----------------+      +--------+----------------+

3 bytes length field at the end of 5 bytes header, do not strip header
The following message is a simple variation of the first example. An extra header value is prepended to the message. lengthAdjustment is zero again because the decoder always takes the length of the prepended data into account during frame length calculation.
   lengthFieldOffset   = 2 (= the length of Header 1)
   lengthFieldLength   = 3
   lengthAdjustment    = 0
   initialBytesToStrip = 0

   BEFORE DECODE (17 bytes)                      AFTER DECODE (17 bytes)
   +----------+----------+----------------+      +----------+----------+----------------+
   | Header 1 |  Length  | Actual Content |----->| Header 1 |  Length  | Actual Content |
   |  0xCAFE  | 0x00000C | "HELLO, WORLD" |      |  0xCAFE  | 0x00000C | "HELLO, WORLD" |
   +----------+----------+----------------+      +----------+----------+----------------+

3 bytes length field at the beginning of 5 bytes header, do not strip header
This is an advanced example that shows the case where there is an extra header between the length field and the message body. You have to specify a positive lengthAdjustment so that the decoder counts the extra header into the frame length calculation.
   lengthFieldOffset   = 0
   lengthFieldLength   = 3
   lengthAdjustment    = 2 (= the length of Header 1)
   initialBytesToStrip = 0

   BEFORE DECODE (17 bytes)                      AFTER DECODE (17 bytes)
   +----------+----------+----------------+      +----------+----------+----------------+
   |  Length  | Header 1 | Actual Content |----->|  Length  | Header 1 | Actual Content |
   | 0x00000C |  0xCAFE  | "HELLO, WORLD" |      | 0x00000C |  0xCAFE  | "HELLO, WORLD" |
   +----------+----------+----------------+      +----------+----------+----------------+

2 bytes length field at offset 1 in the middle of 4 bytes header, strip the first header field and the length field
This is a combination of all the examples above. There are the prepended header before the length field and the extra header after the length field. The prepended header affects the lengthFieldOffset and the extra header affects the lengthAdjustment. We also specified a non-zero initialBytesToStrip to strip the length field and the prepended header from the frame. If you don't want to strip the prepended header, you could specify 0 for initialBytesToSkip.
   lengthFieldOffset   = 1 (= the length of HDR1)
   lengthFieldLength   = 2
   lengthAdjustment    = 1 (= the length of HDR2)
   initialBytesToStrip = 3 (= the length of HDR1 + LEN)

   BEFORE DECODE (16 bytes)                       AFTER DECODE (13 bytes)
   +------+--------+------+----------------+      +------+----------------+
   | HDR1 | Length | HDR2 | Actual Content |----->| HDR2 | Actual Content |
   | 0xCA | 0x000C | 0xFE | "HELLO, WORLD" |      | 0xFE | "HELLO, WORLD" |
   +------+--------+------+----------------+      +------+----------------+

2 bytes length field at offset 1 in the middle of 4 bytes header, strip the first header field and the length field, the length field represents the length of the whole message
Let's give another twist to the previous example. The only difference from the previous example is that the length field represents the length of the whole message instead of the message body, just like the third example. We have to count the length of HDR1 and Length into lengthAdjustment. Please note that we don't need to take the length of HDR2 into account because the length field already includes the whole header length.
   lengthFieldOffset   =  1
   lengthFieldLength   =  2
   lengthAdjustment    = -3 (= the length of HDR1 + LEN, negative)
   initialBytesToStrip =  3

   BEFORE DECODE (16 bytes)                       AFTER DECODE (13 bytes)
   +------+--------+------+----------------+      +------+----------------+
   | HDR1 | Length | HDR2 | Actual Content |----->| HDR2 | Actual Content |
   | 0xCA | 0x0010 | 0xFE | "HELLO, WORLD" |      | 0xFE | "HELLO, WORLD" |
   +------+--------+------+----------------+      +------+----------------+

*/

/**
Params:
em – the ByteOrder of the length field
lengthFiledByte – the length of the length field
maxFrameLength – the maximum length of the frame. If the length of the frame is greater than this value, TooLongFrameException will be thrown.
lengthFieldOffset – the offset of the length field
lengthAdjustment – the compensation value to add to the value of the length field
initialBytesToStrip – the number of first bytes to strip out from the decoded frame
*/

var LengthFieldBasedFrameDecoder = func(em EndianMode,
	lengthFiledByte LengthFiledByte,
	maxFrameLength uint64,
	lengthFieldOffset uint64,
	lengthAdjustment uint64,
	initialBytesToStrip uint64) (Decoder[data.ByteBuffer, [][]byte], error) {

	var lengthFieldLength uint64 = 8

	switch lengthFiledByte {
	case INT8:
		lengthFieldLength = 1
	case INT16:
		lengthFieldLength = 2
	case INT32:
		lengthFieldLength = 4
	case INT64:
		lengthFieldLength = 8
	}

	if maxFrameLength <= 0 {
		return nil, MaxFrameLengthInvalid
	}

	if lengthFieldOffset < 0 {
		return nil, LengthFieldOffsetInvalid
	}

	if initialBytesToStrip < 0 {
		return nil, InitialBytesToStripInvalid
	}

	if lengthFieldOffset > maxFrameLength-lengthFieldLength {
		return nil, MaxFrameLengthSmallInvalid
	}

	// lengthAdjustment – 长度域的偏移量矫正。
	// 如果长度域的值，除了包含有效数据域的长度外，还包含了其他域（如长度域自身）长度，那么，就需要进行矫正。
	// 矫正的值为：包长 - 长度域的值 – 长度域偏移 – 长度域长。
	//  0 =  包长(12) - 长度域的值(10) – 长度域偏移(0) – 长度域长(2)。

	// 包的总长度看扣除数据域的长度
	lengthFrameRemainLen := lengthAdjustment + lengthFieldOffset + lengthFieldLength

	return Decoder[data.ByteBuffer, [][]byte](func(bb *data.ByteBuffer) (*[][]byte, error) {
		b := bb.Bytes()
		allLen := uint64(bb.Len())
		var idx uint64 = 0
		rtn := make([][]byte, 0)

		for {
			// get the length data area's end offset
			lenDataEndOffset := idx + lengthFieldOffset + lengthFieldLength

			// the frame length is less than lengthDataEnd
			if allLen < lenDataEndOffset {
				break
			}
			// get the value of length data area
			lenValue := uint64(UnpackFieldLength(em, lengthFiledByte, b[idx+lengthFieldOffset:lenDataEndOffset]))

			// 包的总长度
			frameLen := lengthFrameRemainLen + lenValue

			// get the frame end offset
			frameEndOffset := idx + frameLen

			if allLen < frameEndOffset {
				break
			}

			one := b[idx+initialBytesToStrip : frameEndOffset]
			rtn = append(rtn, one)

			idx = frameEndOffset
		}

		if idx > 0 {
			bb.Flip(int(idx))
			return &rtn, nil
		}

		return nil, nil

	}), nil
}

// VariableLengthFieldFrameDecoder
/**
VariableLengthFieldFrameDecoder

Params:
maxFrameLength – the maximum length of the frame. If the length of the frame is greater than this value, TooLongFrameException will be thrown.
lengthFieldOffset – the offset of the length field
lengthAdjustment – the compensation value to add to the value of the length field
initialBytesToStripFn – the number of first bytes to strip out from the decoded frame
*/

var VariableLengthFieldFrameDecoder = func(
	maxFrameLength uint64,
	lengthFieldOffset uint64,
	lengthAdjustment uint64,
	initialBytesToStripFn func(variableLength uint64) uint64) (Decoder[data.ByteBuffer, [][]byte], error) {

	if maxFrameLength <= 0 {
		return nil, MaxFrameLengthInvalid
	}

	if lengthFieldOffset < 0 {
		return nil, LengthFieldOffsetInvalid
	}

	var lengthFieldLength uint64 = 1
	if lengthFieldOffset > maxFrameLength-lengthFieldLength {
		return nil, MaxFrameLengthSmallInvalid
	}

	// lengthAdjustment – 长度域的偏移量矫正。
	// 如果长度域的值，除了包含有效数据域的长度外，还包含了其他域（如长度域自身）长度，那么，就需要进行矫正。
	// 矫正的值为：包长 - 长度域的值 – 长度域偏移 – 长度域长。
	//  0 =  包长(12) - 长度域的值(10) – 长度域偏移(0) – 长度域长(2)。

	// 包的总长度看扣除数据域的长度和长度域的长度
	//lengthFrameRemainLen := lengthAdjustment + lengthFieldOffset + lengthFieldLength
	lengthFrameRemainLen := lengthAdjustment + lengthFieldOffset

	ii := 0

	return Decoder[data.ByteBuffer, [][]byte](func(bb *data.ByteBuffer) (*[][]byte, error) {
		b := bb.Bytes()
		allLen := uint64(bb.Len())
		var idx uint64 = 0
		rtn := make([][]byte, 0)

		for {
			// get the length data area's end offset
			lenDataStartOffset := idx + lengthFieldOffset

			// the frame length is less than lengthDataEnd
			if allLen < lenDataStartOffset+lengthFieldLength {
				break
			}
			// get the value of length data area
			//lenValue := uint64(UnpackFieldLength(em, lengthFiledByte, b[idx+lengthFieldOffset:lenDataEndOffset]))
			//fmt.Println(lengthFrameRemainLen + lenDataStartOffset + uint64(len(b)))
			// get the value of length data area

			lenDataEndOffset := lenDataStartOffset + 10
			if allLen < lenDataEndOffset {
				lenDataEndOffset = allLen
			}

			//lenValue, n := UnpackVariableLength(b[lenDataStartOffset:lenDataEndOffset])
			lenValue, n := UnpackVariableLength(b, int(lenDataStartOffset))

			if n == 0 {
				break
			}

			if n < 0 {
				return nil, VariableLengthOverflow
			}

			lengthFieldLength = uint64(n)

			// 包的总长度
			frameLen := lengthFrameRemainLen + lenValue + lengthFieldLength

			// get the frame end offset
			frameEndOffset := idx + frameLen

			if allLen < frameEndOffset {
				break
			}

			var initialBytesToStrip uint64 = 0

			if initialBytesToStripFn != nil {
				initialBytesToStrip = initialBytesToStripFn(lengthFieldLength)
			}

			if initialBytesToStrip < 0 {
				return nil, InitialBytesToStripInvalid
			}

			one := b[idx+initialBytesToStrip : frameEndOffset]

			//fmt.Printf("[%v] VariableLen[%v] FiledLen[%v] FrameLen[%v] [%v]\n", ii, lengthFieldLength, lenValue, frameLen, string(one))
			ii++

			rtn = append(rtn, one)

			idx = frameEndOffset
		}

		if idx > 0 {
			bb.Flip(int(idx))
			return &rtn, nil
		}

		return nil, nil

	}), nil
}
