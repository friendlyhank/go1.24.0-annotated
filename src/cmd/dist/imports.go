package main

import (
	"bufio"
	"errors"
	"io"
	"strings"
)

// importReader - 读取导入的包
type importReader struct {
	b    *bufio.Reader
	buf  []byte // 临时存放读取的数据
	peek byte   // 下一个待读取的数据
	err  error  // 读取过程发生的错误
	eof  bool   // 表示是否已经到达输入流的末尾（EOF）
	nerr int    // 记录错误的额数量
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
}

// readimports returns the imports found in the named file.
// 读取文件导入的包信息
func readimports(file string) []string {
	var imports []string
	r := &importReader{b: bufio.NewReader(strings.NewReader(readfile(file)))}
	r.readKeyword("package")
	return imports
}
