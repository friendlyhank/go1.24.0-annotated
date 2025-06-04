package runtime

import "std/runtime/sys"

// moduledata records information about the layout of the executable
// image. It is written by the linker. Any changes here must be
// matched changes to the code in cmd/link/internal/ld/symtab.go:symtab.
// moduledata is stored in statically allocated non-pointer memory;
// none of the pointers here are visible to the garbage collector.
// moduledata 记录了可执行文件映像的布局信息。
// 它由链接器写入。此处的任何更改都必须
// 与 cmd/link/internal/ld/symtab.go:symtab 中的代码更改相匹配。
// moduledata 存储在静态分配的非指针内存中；
// 此处的所有指针对垃圾回收器均不可见。
// 这个东西由连接器link使用
type moduledata struct {
	sys.NotInHeap // Only in static data

	next *moduledata
}
