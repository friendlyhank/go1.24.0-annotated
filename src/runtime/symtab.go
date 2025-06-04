package runtime

import (
	"internal/abi"
	"internal/runtime/sys"
)

// pcHeader holds data used by the pclntab lookups.
// pcHeader 是 Go 运行时符号表（pclntab）的核心元数据结构
type pcHeader struct {
}

type modulehash struct {
}

type functab struct {
}

type textsect struct {
}

type itab = abi.ITab

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
// todo hank 这里有空可以重点研究下 很多字段还不懂是啥
type moduledata struct {
	sys.NotInHeap // Only in static data

	// 2.符号表与调试信息 作用：支持运行时反射、调试和错误追踪（如 panic 堆栈打印）。
	pcHeader     *pcHeader // pcHeader 是 Go 运行时符号表（pclntab）的核心元数据结构
	funcnametab  []byte    // 函数名字符串表
	cutab        []uint32
	filetab      []byte
	pctab        []byte    // PC 到源码位置的映射表
	pclntable    []byte    // 行号信息表
	ftab         []functab //  函数入口地址表
	findfunctab  uintptr
	minpc, maxpc uintptr

	// 1.模块布局描述 作用：记录模块（可执行文件/共享库/插件）在内存中的布局，支持运行时地址转换（如 textAddr 和 textOff 方法）。
	text, etext           uintptr // 文本段（代码段）的起始和结束地
	noptrdata, enoptrdata uintptr
	data, edata           uintptr // 可读写数据段
	bss, ebss             uintptr // 未初始化全局变量段
	noptrbss, enoptrbss   uintptr
	covctrs, ecovctrs     uintptr
	end, gcdata, gcbss    uintptr
	types, etypes         uintptr
	rodata                uintptr // 只读数据段
	gofunc                uintptr // go.func.*

	textsectmap []textsect
	typelinks   []int32 // offsets from types
	itablinks   []*itab

	ptab []ptabEntry

	pluginpath string
	pkghashes  []modulehash

	// This slice records the initializing tasks that need to be
	// done to start up the program. It is built by the linker.
	// 4.运行时管理与初始化 初始化任务列表
	inittasks []*initTask

	// 3.模块依赖与版本验证 作用：验证模块间的兼容性，防止不同版本库的 ABI 冲突。
	modulename   string
	modulehashes []modulehash

	// 4.运行时管理与初始化  是否包含 main 函数
	hasmain uint8 // 1 if module contains the main function, 0 otherwise
	bad     bool  // module failed to load and should be ignored

	gcdatamask, gcbssmask bitvector

	typemap map[typeOff]*_type // offset to *_rtype in previous module

	next *moduledata
}

type initTask struct{}
