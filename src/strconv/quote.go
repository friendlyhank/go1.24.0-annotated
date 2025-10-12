package strconv

import "unicode/utf8"

func quoteWith(s string, quote byte, ASCIIonly, graphicOnly bool) string {
	// 分配足够的1.5倍的空间
	return string(appendQuotedWith(make([]byte, 0, 3*len(s)/2), s, quote, ASCIIonly, graphicOnly))
}

/*
* buf：目标缓冲区，用于存储结果。
* s：输入字符串
* quote：引号字符（如 '"' 或 '\”）
* ASCIIonly：若为 true，仅保留 ASCII 字符，其他字符转义。
*graphicOnly：若为 true，保留 Unicode 图形字符，其他转义。
 */
func appendQuotedWith(buf []byte, s string, quote byte, ASCIIonly, graphicOnly bool) []byte {
	// Often called with big strings, so preallocate. If there's quoting,
	// this is conservative but still helps a lot.
	// 若剩余缓存区容量不足以容纳s，则进行扩容(buf的容量-buf实际存储的长度 小于 s的长度)
	if cap(buf)-len(buf) < len(s) {
		nBuf := make([]byte, len(buf), len(buf)+1+len(s)+1)
		copy(nBuf, buf)
		buf = nBuf
	}
	// 设置起始符号
	buf = append(buf, quote)
	for width := 0; len(s) > 0; s = s[width:] {
		r := rune(s[0])
		width = 1
		// 表示多字节utf-8,小于的表示ASCII（单字节）
		if r >= utf8.RuneSelf {
			r, width = utf8.DecodeRuneInString(s)
		}
		// 常见的如单字节非法字符\x80、\x90、\xC0、\xE0、\xF5、\xFE、\xFF
		// 如果是非法的UTF-8 字节，将每个每个非法字节转为 \xNN 形式，不进行转义，保证输出字符合法、可见、可逆。
		if width == 1 && r == utf8.RuneError {
			continue
		}
	}
	return buf
}

// Quote returns a double-quoted Go string literal representing s. The
// returned string uses Go escape sequences (\t, \n, \xFF, \u0100) for
// control characters and non-printable characters as defined by
// [IsPrint].
// Quote 返回一个用双引号括起的 Go 字符串字面量，用于表示字符串 s。
// 返回的字符串使用 Go 的转义序列（如 \t、\n、\xFF、\u0100）表示控制字符
// 和不可打印字符（具体定义见 [IsPrint] 函数）
func Quote(s string) string {
	return quoteWith(s, '"', false, false)
}
