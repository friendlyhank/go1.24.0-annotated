package builtin

// bool is the set of boolean values, true and false.
type bool bool

// string is the set of all strings of 8-bit bytes, conventionally but not
// necessarily representing UTF-8-encoded text. A string may be empty, but
// not nil. Values of string type are immutable.
// 定义字符串类型，由8字节组成的序列
type string string

// The len built-in function returns the length of v, according to its type:
//
//   - Array: the number of elements in v.
//   - Pointer to array: the number of elements in *v (even if v is nil).
//   - Slice, or map: the number of elements in v; if v is nil, len(v) is zero.
//   - String: the number of bytes in v.
//   - Channel: the number of elements queued (unread) in the channel buffer;
//     if v is nil, len(v) is zero.
//
// For some arguments, such as a string literal or a simple array expression, the
// result can be a constant. See the Go language specification's "Length and
// capacity" section for details.
func len(v Type) int
