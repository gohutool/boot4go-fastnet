package codec

import (
	"encoding/binary"
)

/**
* golang-sample源代码，版权归锦翰科技（深圳）有限公司所有。
* <p>
* 文件名称 : util.go
* 文件路径 :
* 作者 : DavidLiu
× Email: david.liu@ginghan.com
*
* 创建日期 : 2022/6/6 12:06
* 修改历史 : 1. [2022/6/6 12:06] 创建文件 by LongYong
*/

type LengthFiledByte byte

const (
	INT8 LengthFiledByte = iota
	INT16
	INT32
	INT64
)

type EndianMode byte

const (
	BIGENDIAN EndianMode = iota
	LITTLEENDIAN
)

var (
	bigEndian    binary.ByteOrder = binary.BigEndian
	littleEndian binary.ByteOrder = binary.LittleEndian
)

func UnpackFieldLength(endianMode EndianMode, len LengthFiledByte, byte []byte) int64 {
	var rtn int64

	byteOrder := bigEndian
	if endianMode == LITTLEENDIAN {
		byteOrder = littleEndian
	}

	switch len {
	case INT8:
		rtn = int64(byte[0])
	case INT16:
		rtn = int64(byteOrder.Uint16(byte))
	case INT32:
		rtn = int64(byteOrder.Uint32(byte))
	case INT64:
		rtn = int64(byteOrder.Uint64(byte))
	default:
	}

	return rtn
}

func PackFieldLength(endianMode EndianMode, len LengthFiledByte, dataLen int64) []byte {

	byteOrder := bigEndian
	if endianMode == LITTLEENDIAN {
		byteOrder = littleEndian
	}

	var lengthBuff []byte

	switch len {
	case INT8:
		lengthBuff = make([]byte, 1)
		lengthBuff[0] = byte(dataLen)
	case INT16:
		lengthBuff = make([]byte, 2)
		byteOrder.PutUint16(lengthBuff, uint16(dataLen))
	case INT32:
		lengthBuff = make([]byte, 4)
		byteOrder.PutUint32(lengthBuff, uint32(dataLen))
	case INT64:
		lengthBuff = make([]byte, 8)
		byteOrder.PutUint64(lengthBuff, uint64(dataLen))
	default:
		lengthBuff = make([]byte, 8)
		byteOrder.PutUint64(lengthBuff, uint64(dataLen))
	}

	return lengthBuff
}

func UnpackVariableLength(b []byte) (uint64, int) {

	var vbi uint64
	var multiplier uint64
	n := 0
	for {
		if n >= len(b) {
			return 0, 0
		}

		digit := b[n]
		n++
		vbi |= uint64(digit&127) << multiplier

		if vbi > 18446744073709551615 {
			return 0, -1
		}

		if (digit & 128) == 0 {
			break
		}

		multiplier += 7
	}

	return vbi, n
}

func PackVariableLength(length uint64) []byte {
	var result []byte
	if length < 128 {
		result = make([]byte, 1)
	} else if length < 16384 {
		result = make([]byte, 2)
	} else if length < 2097152 {
		result = make([]byte, 3)
	} else if length < 268435456 {
		result = make([]byte, 4)
	} else if length < 34359738368 {
		result = make([]byte, 5)
	} else if length < 4398046511104 {
		result = make([]byte, 6)
	} else if length < 562949953421312 {
		result = make([]byte, 7)
	} else if length < 72057594037927936 {
		result = make([]byte, 8)
	} else if length < 9223372036854775808 {
		result = make([]byte, 9)
	} else {
		result = make([]byte, 10)
	}
	var i int
	for {
		encodedByte := length % 128
		length = length / 128
		// if there are more data to encode, set the top bit of this byte
		if length > 0 {
			encodedByte = encodedByte | 128
		}
		result[i] = byte(encodedByte)
		i++
		if length <= 0 {
			break
		}
	}
	return result
}
