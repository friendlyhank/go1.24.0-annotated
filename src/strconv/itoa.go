package strconv

import "fmt"

/*
 * int,int64转string
 */

const fastSmalls = true // enable fast path for small integers 小整数是否启用快速路径

// FormatInt returns the string representation of i in the given base,
// for 2 <= base <= 36. The result uses the lower-case letters 'a' to 'z'
// for digit values >= 10.
// FormatInt 返回整数 i 在指定进制下的字符串表示，支持的进制范围是 2 <= base <= 36。当数字值 >= 10 时，结果使用小写字母 'a' 到 'z' 来表示。
func FormatInt(i int64, base int) string {
	fmt.Println(i, base)
	return ""
}

// Itoa is equivalent to [FormatInt](int64(i), 10).
// 将int类型转换成字符串
func Itoa(i int) string {
	return FormatInt(int64(i), 10)
}

const nSmalls = 100 // 使用快速路径的数字范围
