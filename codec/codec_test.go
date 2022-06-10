package codec

import (
	"fmt"
	"testing"
)

/**
* golang-sample源代码，版权归锦翰科技（深圳）有限公司所有。
* <p>
* 文件名称 : codec_test.go
* 文件路径 :
* 作者 : DavidLiu
× Email: david.liu@ginghan.com
*
* 创建日期 : 2022/6/8 10:17
* 修改历史 : 1. [2022/6/8 10:17] 创建文件 by LongYong
*/

func TestVariableLengthUnpack(t *testing.T) {
	var len uint64 = 281474976710527
	b := PackVariableLength(len)
	printBytes(b)

	b = PackVariableLength(127)
	printBytes(b)

	b = PackVariableLength(16383)
	printBytes(b)

	b = PackVariableLength(2097151)
	printBytes(b)

	b = PackVariableLength(1024)
	printBytes(b)

	b = PackVariableLength(4095)
	printBytes(b)

	b = PackVariableLength(268435455)
	printBytes(b)

	b = PackVariableLength(1982)
	printBytes(b)
}

func printBytes(b []byte) {
	fmt.Printf("Result : %v\n", b)
	printResult(b)
}

func printResult(b []byte) {
	v, n := UnpackVariableLength(b, 0)
	fmt.Printf("Length Result : %v[%v]\n", v, n)
}

func TestVariableLengthPack(t *testing.T) {
	printBytes([]byte{0})
	printBytes([]byte{127})
	printBytes([]byte{255, 127})
	printBytes([]byte{255, 255, 127})
	printBytes([]byte{255, 255, 255, 127})
	printBytes([]byte{255, 255, 255, 255, 127})
	printBytes([]byte{255, 255, 255, 255, 255, 127})
	printBytes([]byte{255, 255, 255, 255, 255, 255, 127})
	printBytes([]byte{255, 255, 255, 255, 255, 255, 255, 127})
	printBytes([]byte{255, 255, 255, 255, 255, 255, 255, 255, 127})
	printBytes([]byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 127})
}

func Test2(t *testing.T) {
	b := []byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 127}[2:]
	printBytes(b)
}
