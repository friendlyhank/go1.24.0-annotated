package stringslite

import "unsafe"

// Clone - 拷贝字符串(分配新内存)
func Clone(s string) string {
	if len(s) == 0 {
		return ""
	}
	b := make([]byte, len(s))
	copy(b, s)
	// &b[0] 取得的是这个切片第一个元素的地址（即整个底层数组的起始地址）。
	// 使用该地址和长度 len(b)，unsafe.String 能够将字节切片内容映射为字符串(内存地址是连续的)，而无需额外分配内存。
	return unsafe.String(&b[0], len(b))
}
