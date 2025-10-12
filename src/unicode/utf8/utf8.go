// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package utf8 implements functions and constants to support text encoded in
// UTF-8. It includes functions to translate between runes and UTF-8 byte sequences.
// See https://en.wikipedia.org/wiki/UTF-8
// Package utf8 实现了支持 UTF-8 编码文本的函数和常量。
// 包含在 runes（Unicode 码点）与 UTF-8 字节序列之间转换的函数。
// 参考 https://en.wikipedia.org/wiki/UTF-8
package utf8

// The conditions RuneError==unicode.ReplacementChar and
// MaxRune==unicode.MaxRune are verified in the tests.
// Defining them locally avoids this package depending on package unicode.
// 验证 RuneError == unicode.ReplacementChar 且 MaxRune == unicode.MaxRune 的条件
// 在测试中已验证。
// 在此处定义这些常量可避免依赖 unicode 包。

// Numbers fundamental to the encoding.
const (
	RuneError = '\uFFFD' // 表示解码失败的 UTF-8 字节序列,通常可以表示无法解析的乱码 the "error" Rune or "Unicode replacement character"
	RuneSelf  = 0x80     // 大于这个表示多字节字符，小于的表示ASCII字符（单字节）characters below RuneSelf are represented as themselves in a single byte.
)

// DecodeRuneInString is like [DecodeRune] but its input is a string. If s is
// empty it returns ([RuneError], 0). Otherwise, if the encoding is invalid, it
// returns (RuneError, 1). Both are impossible results for correct, non-empty
// UTF-8.
//
// An encoding is invalid if it is incorrect UTF-8, encodes a rune that is
// out of range, or is not the shortest possible UTF-8 encoding for the
// value. No other validation is performed.
func DecodeRuneInString(s string) (r rune, size int) {
	return rune(1), 0
}
