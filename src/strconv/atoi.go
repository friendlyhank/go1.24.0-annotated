package strconv

import (
	"errors"
	"fmt"
	"internal/stringslite"
)

// lower(c) is a lower-case letter if and only if
// c is either that lower-case letter or the equivalent upper-case letter.
// Instead of writing c == 'x' || c == 'X' one can write lower(c) == 'x'.
// Note that lower of non-letters can produce other non-letters.
/*
 *这个函数使用了一个非常巧妙的位运算技巧来实现大小写转换：
1. 关键计算 : 'x' - 'X'

   - 'x' 的 ASCII 值是 120 (0x78)
   - 'X' 的 ASCII 值是 88 (0x58)
   - 差值是 32 (0x20)，二进制为 00100000
2. 位运算操作 : c | 0x20

   - 对输入字符 c 与 0x20 进行按位或运算
   - 这会将第 6 位（从右数第 6 位）设置为 1
## 为什么这样工作？
在 ASCII 编码中，大写字母和小写字母的差异就在第 6 位：

- 大写字母：第 6 位为 0
- 小写字母：第 6 位为 1
例如：

- 'A' = 65 = 01000001 → 'a' = 97 = 01100001
- 'B' = 66 = 01000010 → 'b' = 98 = 01100010
通过 | 0x20 操作，无论输入是大写还是小写字母，都会强制将第 6 位设为 1，从而得到小写字母。
*/
func lower(c byte) byte {
	return c | ('x' - 'X')
}

// ErrRange indicates that a value is out of range for the target type.
// 值超出范围限制
var ErrRange = errors.New("value out of range")

// ErrSyntax indicates that a value does not have the right syntax for the target type.
var ErrSyntax = errors.New("invalid syntax")

// A NumError records a failed conversion. 记录失败转换的错误
type NumError struct {
	Func string // 转换的方法 the failing function (ParseBool, ParseInt, ParseUint, ParseFloat, ParseComplex)
	Num  string // 输入的值 the input
	Err  error  // 错误原因 the reason the conversion failed (e.g. ErrRange, ErrSyntax, etc.)
}

// Error - 解析错误处理
func (e *NumError) Error() string {
	return "strconv." + e.Func + ": " + "parsing " + Quote(e.Num) + ": " + e.Err.Error()
}

// All ParseXXX functions allow the input string to escape to the error value.
// This hurts strconv.ParseXXX(string(b)) calls where b is []byte since
// the conversion from []byte must allocate a string on the heap.
// If we assume errors are infrequent, then we can avoid escaping the input
// back to the output by copying it first. This allows the compiler to call
// strconv.ParseXXX without a heap allocation for most []byte to string
// conversions, since it can now prove that the string cannot escape Parse.
func syntaxError(fn, str string) *NumError {
	return &NumError{fn, stringslite.Clone(str), ErrSyntax}
}

// rangeError - 解析转换超过数值
func rangeError(fn, str string) *NumError {
	return &NumError{fn, stringslite.Clone(str), ErrRange}
}

func baseError(fn, str string, base int) *NumError {
	return &NumError{fn, stringslite.Clone(str), errors.New("invalid base " + Itoa(base))}
}

func bitSizeError(fn, str string, bitSize int) *NumError {
	return &NumError{fn, stringslite.Clone(str), errors.New("invalid bit size " + Itoa(bitSize))}
}

/*
	位运算拆解

- ^uint(0) ：对 0 做按位取反，得到“全 1”的无符号数。
  - 在 32 位平台： ^uint(0) 是 0xFFFFFFFF （32 个 1）
  - 在 64 位平台： ^uint(0) 是 0xFFFFFFFFFFFFFFFF （64 个 1）

- >> 63 ：将上述无符号数右移 63 位。
  - 在 64 位平台：右移 63 位后得到 1 （最高位的 1 被移到最低位）
  - 在 32 位平台：因为值只有 32 位，右移 63 位后结果为 0 （移完为 0）

- 32 << (...) ：
  - 在 64 位平台： 32 << 1 = 64
  - 在 32 位平台： 32 << 0 = 32

因此：

- 32 位平台： intSize = 32
- 64 位平台： intSize = 64
*/
const intSize = 32 << (^uint(0) >> 63)

// IntSize is the size in bits of an int or uint value.
// 这行代码用于在编译期计算当前平台上 int 类型的位数，结果只会是 32 或 64
const IntSize = intSize

const maxUint64 = 1<<64 - 1

// ParseUint is like [ParseInt] but for unsigned numbers.
//
// A sign prefix is not permitted.
func ParseUint(s string, base int, bitSize int) (uint64, error) {
	const fnParseUint = "ParseUint"

	if s == "" {
		return 0, syntaxError(fnParseUint, s)
	}

	base0 := base == 0

	s0 := s
	switch {
	case 2 <= base && base <= 36:
		// valid base; nothing to do

	case base == 0:
		// Look for octal, hex prefix.判断前缀
		base = 10
		if s[0] == '0' {
			switch {
			case len(s) >= 3 && lower(s[1]) == 'b':
				base = 2
				s = s[2:]
			case len(s) >= 3 && lower(s[1]) == 'o':
				base = 8
				s = s[2:]
			case len(s) >= 3 && lower(s[1]) == 'x':
				base = 16
				s = s[2:]
			default:
				base = 8
				s = s[1:]
			}
		}
	default:
		return 0, baseError(fnParseUint, s0, base)
	}

	if bitSize == 0 {
		bitSize = IntSize
	} else if bitSize < 0 || bitSize > 64 {
		return 0, bitSizeError(fnParseUint, s0, bitSize)
	}

	// Cutoff is the smallest number such that cutoff*base > maxUint64.
	// Use compile-time constants for common cases.
	var cutoff uint64
	switch base {
	case 10:
		cutoff = maxUint64/10 + 1
	case 16:
		cutoff = maxUint64/16 + 1
	default:
		cutoff = maxUint64/uint64(base) + 1
	}

	fmt.Println("cutoff", cutoff)

	maxVal := uint64(1)<<uint(bitSize) - 1

	// 核心字符串如何转化为数字
	underscores := false
	var n uint64
	for _, c := range []byte(s) {
		var d byte
		switch {
		case c == '_' && base0: //自动跳过数字分隔符下划线（如 1_000_000 ）
			underscores = true
			continue
		case '0' <= c && c <= '9': // 处理 0-9 的数字字符(这种判断可以借鉴)
			d = c - '0' // 通过 ASCII 值相减得到实际数字值
		case 'a' <= lower(c) && lower(c) <= 'z': // 处理十六进制等高进制中的字母字符 (这种判断可以借鉴)
			d = lower(c) - 'a' + 10
		default:
			return 0, syntaxError(fnParseUint, s0)
		}

		if d >= byte(base) {
			return 0, syntaxError(fnParseUint, s0)
		}

		if n >= cutoff {
			// n*base overflows
			return maxVal, rangeError(fnParseUint, s0)
		}
		n *= uint64(base)
		n1 := n + uint64(d)
		if n1 < n || n1 > maxVal {
			// n+d overflows
			return maxVal, rangeError(fnParseUint, s0)
		}
		n = n1
	}

	fmt.Println(underscores)

	return n, nil
}

// ParseInt 解析字符串 s 为指定进制 (0, 2-36) 和位大小 (0-64) 的整数值
// ParseInt interprets a string s in the given base (0, 2 to 36) and
// bit size (0 to 64) and returns the corresponding value i.
//
// 字符串可以以符号开头："+" 或 "-"
// The string may begin with a leading sign: "+" or "-".
//
// 如果 base 参数为 0，真正的进制由字符串的前缀决定（符号之后）：
// "0b" 表示二进制，"0" 或 "0o" 表示八进制，"0x" 表示十六进制，其他情况为十进制
// 另外，仅当 base 为 0 时，允许使用下划线字符，遵循 Go 语法中的整数字面量规则
// If the base argument is 0, the true base is implied by the string's
// prefix following the sign (if present): 2 for "0b", 8 for "0" or "0o",
// 16 for "0x", and 10 otherwise. Also, for argument base 0 only,
// underscore characters are permitted as defined by the Go syntax for
// [integer literals].
//
// bitSize 参数指定结果必须适合的整数类型
// 位大小 0, 8, 16, 32, 64 分别对应 int, int8, int16, int32, int64
// 如果 bitSize 小于 0 或大于 64，将返回错误
// The bitSize argument specifies the integer type
// that the result must fit into. Bit sizes 0, 8, 16, 32, and 64
// correspond to int, int8, int16, int32, and int64.
// If bitSize is below 0 or above 64, an error is returned.
//
// ParseInt 返回的错误具体类型为 [*NumError]，包含 err.Num = s
// 如果 s 为空或包含无效数字，err.Err = [ErrSyntax]，返回值为 0
// 如果 s 对应的值无法用给定大小的有符号整数表示，err.Err = [ErrRange]
// 返回值为对应 bitSize 和符号的最大绝对值整数
// The errors that ParseInt returns have concrete type [*NumError]
// and include err.Num = s. If s is empty or contains invalid
// digits, err.Err = [ErrSyntax] and the returned value is 0;
// if the value corresponding to s cannot be represented by a
// signed integer of the given size, err.Err = [ErrRange] and the
// returned value is the maximum magnitude integer of the
// appropriate bitSize and sign.
//
// [integer literals]: https://go.dev/ref/spec#Integer_literals
func ParseInt(s string, base int, bitSize int) (i int64, err error) {
	const fnParseInt = "ParseInt"

	if s == "" {
		return 0, syntaxError(fnParseInt, s)
	}

	// Pick off leading sign.提取+，-符号
	s0 := s
	neg := false
	if s[0] == '+' {
		s = s[1:]
	} else if s[0] == '-' {
		neg = true
		s = s[1:]
	}

	// Convert unsigned and check range.转化为无符合的整数并检查范围
	var un uint64
	un, err = ParseUint(s, base, bitSize)
	if err != nil && err.(*NumError).Err != ErrRange {
		err.(*NumError).Func = fnParseInt
		err.(*NumError).Num = stringslite.Clone(s0)
		return 0, err
	}

	if bitSize == 0 {
		bitSize = IntSize
	}

	/*
		- bitSize = 8 ： cutoff = 1 << 7 = 128 （int8 范围：-128 到 127）
		- bitSize = 32 ： cutoff = 1 << 31 = 2147483648 （int32 范围：-2147483648 到 2147483647）
		- bitSize = 64 ： cutoff = 1 << 63 = 9223372036854775808 （int64 范围：-9223372036854775808 到 9223372036854775807）
	*/
	cutoff := uint64(1 << uint(bitSize-1))
	if !neg && un >= cutoff {
		return int64(cutoff - 1), rangeError(fnParseInt, s0)
	}
	if neg && un > cutoff {
		return -int64(cutoff), rangeError(fnParseInt, s0)
	}

	n := int64(un)
	if neg {
		n = -n
	}
	return n, nil
}
