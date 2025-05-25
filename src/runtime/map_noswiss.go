package runtime

// mapinitnoop is a no-op function known the Go linker; if a given global
// map (of the right size) is determined to be dead, the linker will
// rewrite the relocation (from the package init func) from the outlined
// map init function to this symbol. Defined in assembly so as to avoid
// complications with instrumentation (coverage, etc).
/*
 *主要被用于链接器link,用于链接器优化和死代码消除
 1. 替代无用的全局 map 初始化，如var _ = map[string]int{"a": 1} // 未命名的全局 map
	场景： 当编译器/链接器分析发现某个全局 map 变量 从未被访问 或 初始化后未被使用，会将其标记为 死代码（dead code）。
	优化行为： 链接器会 移除该 map 的初始化函数调用（如 makemap），并将原本指向初始化函数的 重定位（relocation） 指向 mapinitnoop。
	目的： 减少二进制体积，提升性能，同时避免因移除初始化函数导致的符号未定义错误。
  2. 避免插桩（Instrumentation）干扰
	插桩问题： 如果用 Go 语言实现 mapinitnoop，在启用覆盖率分析（-cover）或竞态检测（-race）等插桩工具时，编译器可能会为其注入额外代码（如 __count_0 变量），破坏其“空函数”的特性。
	解决方案： 用汇编语言定义该函数（通常在 runtime/mapinit_noop.s 中），确保其 不包含任何指令，也不会被插桩工具修改。
*/
func mapinitnoop()
