// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sys

// NOTE: keep in sync with cmd/compile/internal/types.CalcSize
// to make the compiler recognize this as an intrinsic type.
是 Go 运行时中用于 支持 NotInHeap 类型机制 的底层编译器协同结构，其核心目的是 确保编译器将特定类型识别为内建类型，从而正确应用内存分配限制规则。
type nih struct{}

// NotInHeap is a type must never be allocated from the GC'd heap or on the stack,
// and is called not-in-heap.
//
// Other types can embed NotInHeap to make it not-in-heap. Specifically, pointers
// to these types must always fail the `runtime.inheap` check. The type may be used
// for global variables, or for objects in unmanaged memory (e.g., allocated with
// `sysAlloc`, `persistentalloc`, `fixalloc`, or from a manually-managed span).
//
// Specifically:
//
// 1. `new(T)`, `make([]T)`, `append([]T, ...)` and implicit heap
// allocation of T are disallowed. (Though implicit allocations are
// disallowed in the runtime anyway.)
//
// 2. A pointer to a regular type (other than `unsafe.Pointer`) cannot be
// converted to a pointer to a not-in-heap type, even if they have the
// same underlying type.
//
// 3. Any type that containing a not-in-heap type is itself considered as not-in-heap.
//
// - Structs and arrays are not-in-heap if their elements are not-in-heap.
// - Maps and channels contains no-in-heap types are disallowed.
//
// 4. Write barriers on pointers to not-in-heap types can be omitted.
//
// The last point is the real benefit of NotInHeap. The runtime uses
// it for low-level internal structures to avoid memory barriers in the
// scheduler and the memory allocator where they are illegal or simply
// inefficient. This mechanism is reasonably safe and does not compromise
// the readability of the runtime.

/*
 NotInHeap 是 Go 运行时（runtime）中的一种内存分配控制机制，用于标记某些类型必须 禁止从 GC 堆或栈上分配内存。
 其本质是通过编译器和运行时的协同检查，确保某些关键数据结构始终位于 非托管内存区域（如全局变量、手动管理的内存池），从而避免触发垃圾回收器的写屏障（Write Barrier）或内存管理逻辑

 (1) 禁止堆/栈分配
  任何包含 NotInHeap 的类型都无法通过以下方式分配内存：
  new(T)           // 禁止
  make([]T)        // 禁止
  append([]T, ...) // 禁止
  原因： 这些操作会隐式分配 GC 托管的内存。运行时底层结构（如调度器、内存分配器）需要避免依赖 GC 机制，否则可能导致死锁或性能问题。

  (2) 类型转换限制
  普通类型的指针（除 unsafe.Pointer 外）不能直接转换为 *NotInHeap 类型的指针，即使底层类型相同
  type MyStruct struct { sys.NotInHeap; ... }
  var s struct{ ... }
  p := (*MyStruct)(unsafe.Pointer(&s)) // 合法
  p := (*MyStruct)(&s) // 非法（编译错误）
  原因：防止意外将栈或堆上的对象伪装成 NotInHeap 类型，破坏内存安全
  (3) 继承规则
  规则：
	结构体或数组：若所有字段/元素均为 NotInHeap，则整体视为 NotInHeap。
	Map/Channel：禁止包含 NotInHeap 类型的元素
  type A struct { sys.NotInHeap }
  type B struct { A }        // 合法：B 也是 NotInHeap
  type C [2]A                // 合法：C 也是 NotInHeap
  var m map[string]A         // 非法（编译错误）
*/

type NotInHeap struct{ _ nih }
