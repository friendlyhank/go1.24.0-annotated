package strconv

import (
	"fmt"
	"reflect"
	"testing"
)

type parseInt64Test struct {
	in  string
	out int64
	err error
}

var parseInt64Tests = []parseInt64Test{
	{"", 0, ErrSyntax},
	{"0", 0, nil},
	{"-0", 0, nil},
	{"+0", 0, nil},
	{"1", 1, nil},
	{"-1", -1, nil},
	{"+1", 1, nil},
	{"12345", 12345, nil},
	{"-12345", -12345, nil},
	{"012345", 12345, nil},
	{"-012345", -12345, nil},
	{"98765432100", 98765432100, nil},
	{"-98765432100", -98765432100, nil},
	{"9223372036854775807", 1<<63 - 1, nil},
	{"-9223372036854775807", -(1<<63 - 1), nil},
	{"9223372036854775808", 1<<63 - 1, ErrRange},
	{"-9223372036854775808", -1 << 63, nil},
	{"9223372036854775809", 1<<63 - 1, ErrRange},
	{"-9223372036854775809", -1 << 63, ErrRange},
	{"-1_2_3_4_5", 0, ErrSyntax}, // base=10 so no underscores allowed
	{"-_12345", 0, ErrSyntax},
	{"_12345", 0, ErrSyntax},
	{"1__2345", 0, ErrSyntax},
	{"12345_", 0, ErrSyntax},
	{"123%45", 0, ErrSyntax},
}

type parseInt32Test struct {
	in  string
	out int32
	err error
}

var parseInt32Tests = []parseInt32Test{
	{"", 0, ErrSyntax},
	{"0", 0, nil},
	{"-0", 0, nil},
	{"1", 1, nil},
	{"-1", -1, nil},
	{"12345", 12345, nil},
	{"-12345", -12345, nil},
	{"012345", 12345, nil},
	{"-012345", -12345, nil},
	{"12345x", 0, ErrSyntax},
	{"-12345x", 0, ErrSyntax},
	{"987654321", 987654321, nil},
	{"-987654321", -987654321, nil},
	{"2147483647", 1<<31 - 1, nil},
	{"-2147483647", -(1<<31 - 1), nil},
	{"2147483648", 1<<31 - 1, ErrRange},
	{"-2147483648", -1 << 31, nil},
	{"2147483649", 1<<31 - 1, ErrRange},
	{"-2147483649", -1 << 31, ErrRange},
	{"-1_2_3_4_5", 0, ErrSyntax}, // base=10 so no underscores allowed
	{"-_12345", 0, ErrSyntax},
	{"_12345", 0, ErrSyntax},
	{"1__2345", 0, ErrSyntax},
	{"12345_", 0, ErrSyntax},
	{"123%45", 0, ErrSyntax},
}

// string转int32的测试用例子
func TestParseInt32(t *testing.T) {
	for i := range parseInt32Tests {
		test := &parseInt32Tests[i]
		out, err := ParseInt(test.in, 10, 32)
		fmt.Println("out", out, "err", err)
		if int64(test.out) != out || !reflect.DeepEqual(test.err, err) {
			t.Errorf("ParseInt(%q, 10 ,32) = %v, %v want %v, %v",
				test.in, out, err, test.out, test.err)
		}
	}
}

// TestParseInt64 - string转int64的测试用例
func TestParseInt64(t *testing.T) {
	for i := range parseInt64Tests {
		test := &parseInt64Tests[i]
		out, err := ParseInt(test.in, 10, 64)
		fmt.Println("out", out, "err", err)
		if test.out != out || !reflect.DeepEqual(test.err, err) {
			t.Errorf("ParseInt(%q, 10, 64) = %v, %v want %v, %v",
				test.in, out, err, test.out, test.err)
		}
	}
}
