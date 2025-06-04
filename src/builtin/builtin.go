package builtin

// bool is the set of boolean values, true and false.
type bool bool

// string is the set of all strings of 8-bit bytes, conventionally but not
// necessarily representing UTF-8-encoded text. A string may be empty, but
// not nil. Values of string type are immutable.
// 定义字符串类型，由8字节组成的序列
type string string
