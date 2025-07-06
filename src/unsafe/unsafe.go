package unsafe

// IntegerType is here for the purposes of documentation only and is not actually
// part of the unsafe package. It represents any arbitrary integer type.
type IntegerType int

// String returns a string value whose underlying bytes
// start at ptr and whose length is len.
//
// The len argument must be of integer type or an untyped constant.
// A constant len argument must be non-negative and representable by a value of type int;
// if it is an untyped constant it is given type int.
// At run time, if len is negative, or if ptr is nil and len is not zero,
// a run-time panic occurs.
//
// Since Go strings are immutable, the bytes passed to String
// must not be modified as long as the returned string value exists.
// 以ptr为起始地址，长度为len的字节数组，返回一个字符串(内存不安全的)
func String(ptr *byte, len IntegerType) string
