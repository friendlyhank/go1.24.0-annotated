package main

import (
	"bufio"
	"errors"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"
)

/*
 * 通过读取文件，获取需要导入的包
 */

// importReader - 读取导入的包
type importReader struct {
	b    *bufio.Reader
	buf  []byte // 临时存放读取的数据
	peek byte   // 下一个待读取的数据
	err  error  // 读取过程发生的错误
	eof  bool   // 表示是否已经到达输入流的末尾（EOF）
	nerr int    // 记录错误的额数量
}

func isIdent(c byte) bool {
	return 'A' <= c && c <= 'Z' || 'a' <= c && c <= 'z' || '0' <= c && c <= '9' || c == '_' || c >= utf8.RuneSelf
}

var (
	errSyntax = errors.New("syntax error")
	errNUL    = errors.New("unexpected NUL in input") // 读取的是NULL
)

// syntaxError records a syntax error, but only if an I/O error has not already been recorded.
func (r *importReader) syntaxError() {
	if r.err == nil {
		r.err = errSyntax
	}
}

// readByte reads the next byte from the input, saves it in buf, and returns it.
// If an error occurs, readByte records the error in r.err and returns 0.
// 从缓冲读取器中读取一个字节
func (r *importReader) readByte() byte {
	c, err := r.b.ReadByte()
	if err == nil {
		r.buf = append(r.buf, c)
		if c == 0 {
			err = errNUL
		}
	}
	if err != nil {
		if err == io.EOF {
			r.eof = true
		} else if r.err == nil {
			r.err = err
		}
		c = 0
	}
	return c
}

func (r *importReader) peekByte(skipSpace bool) byte {
	if r.err != nil {
		if r.nerr++; r.nerr > 10000 {
			panic("go/build: import reader looping")
		}
		return 0
	}
	// Use r.peek as first input byte.
	// Don't just return r.peek here: it might have been left by peekByte(false)
	// and this might be peekByte(true).
	c := r.peek
	// 每次读取下一个字节数据,r.peek会被设置为0以保证读取最新的字节数据
	if c == 0 {
		c = r.readByte()
	}
	// 如果没有错误并且文件未读取完成
	for r.err == nil && !r.eof {
		// 需要跳过空格`
		if skipSpace {
			// For the purposes of this reader, semicolons are never necessary to
			// understand the input and are treated as spaces.
			switch c {
			// 跳过空格字符、换页符、制表符、回车符、换行符、分号字符
			case ' ', '\f', '\t', '\r', '\n', ';':
				c = r.readByte()
				continue
			case '/':
				c = r.readByte()
				if c == '/' { // 说明是代码注释 //
					for c != '\n' && r.err == nil && !r.eof {
						c = r.readByte()
					}
				} else if c == '*' {
					// /* ... */ 的代码注释
					var c1 byte
					for (c != '*' || c1 != '/') && r.err == nil {
						if r.eof {
							r.syntaxError()
						}
						c, c1 = c1, r.readByte()
					}
				} else {
					r.syntaxError() // 发生错误
				}
				c = r.readByte()
				continue
			}
		}
		break
	}
	r.peek = c
	return r.peek
}

// nextByte is like peekByte but advances beyond the returned byte.
// 读取下一字节的数据
func (r *importReader) nextByte(skipSpace bool) byte {
	c := r.peekByte(skipSpace)
	r.peek = 0 // 设置peek为0,这样下次调用peekByte可以读取到最新的字节数据
	return c
}

// readKeyword reads the given keyword from the input.
// If the keyword is not present, readKeyword records a syntax error.
// 读取指定的关键字
func (r *importReader) readKeyword(kw string) {
	r.peekByte(true)
	for i := 0; i < len(kw); i++ {
		if r.nextByte(false) != kw[i] {
			r.syntaxError()
			return
		}
	}
	if isIdent(r.peekByte(false)) {
		r.syntaxError()
	}
}

// readIdent reads an identifier from the input.
// If an identifier is not present, readIdent records a syntax error.
func (r *importReader) readIdent() {
	c := r.peekByte(true)
	if !isIdent(c) {
		r.syntaxError()
		return
	}
	for isIdent(r.peekByte(false)) {
		r.peek = 0
	}
}

// readString reads a quoted string literal from the input.
// If an identifier is not present, readString records a syntax error.
func (r *importReader) readString(save *[]string) {
	switch r.nextByte(true) {
	case '`':
		start := len(r.buf) - 1
		for r.err == nil {
			if r.nextByte(false) == '`' {
				if save != nil {
					*save = append(*save, string(r.buf[start:]))
				}
				break
			}
			if r.eof {
				r.syntaxError()
			}
		}
	case '"':
		start := len(r.buf) - 1
		for r.err == nil {
			c := r.nextByte(false)
			if c == '"' {
				if save != nil {
					*save = append(*save, string(r.buf[start:]))
				}
				break
			}
			if r.eof || c == '\n' {
				r.syntaxError()
			}
			if c == '\\' {
				r.nextByte(false)
			}
		}
	default:
		r.syntaxError()
	}
}

// readImport reads an import clause - optional identifier followed by quoted string -
// from the input.
func (r *importReader) readImport(imports *[]string) {
	c := r.peekByte(true)
	if c == '.' {
		r.peek = 0
	} else if isIdent(c) {
		r.readIdent()
	}
	r.readString(imports)
}

// readimports returns the imports found in the named file.
// 读取文件导入的包信息
func readimports(file string) []string {
	var imports []string
	r := &importReader{b: bufio.NewReader(strings.NewReader(readfile(file)))}
	r.readKeyword("package")
	r.readIdent()
	for r.peekByte(true) == 'i' {
		r.readKeyword("import")
		if r.peekByte(true) == '(' {
			r.nextByte(false)
			for r.peekByte(true) != ')' && r.err == nil {
				r.readImport(&imports)
			}
			r.nextByte(false)
		} else {
			r.readImport(&imports)
		}
	}

	for i := range imports {
		unquoted, err := strconv.Unquote(imports[i])
		if err != nil {
			fatalf("reading imports from %s: %v", file, err)
		}
		imports[i] = unquoted
	}

	return imports
}

// resolveVendor returns a unique package path imported with the given import
// path from srcDir.
//
// resolveVendor assumes that a package is vendored if and only if its first
// path component contains a dot. If a package is vendored, its import path
// is returned with a "vendor" or "cmd/vendor" prefix, depending on srcDir.
// Otherwise, the import path is returned verbatim.
//
//	// 功能：处理 Go 模块的 vendor 机制，返回经过 vendor 路径修正的导入路径
func resolveVendor(imp, srcDir string) string {
	var first string
	// 判断导入包的路径又没/
	if i := strings.Index(imp, "/"); i < 0 {
		first = imp
	} else {
		first = imp[:i]
	}
	isStandard := !strings.Contains(first, ".")
	if isStandard {
		return imp
	}
	return ""
}
