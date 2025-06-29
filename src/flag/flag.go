// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package flag implements command-line flag parsing.

# Usage

Define flags using [flag.String], [Bool], [Int], etc.

This declares an integer flag, -n, stored in the pointer nFlag, with type *int:

	import "flag"
	var nFlag = flag.Int("n", 1234, "help message for flag n")

If you like, you can bind the flag to a variable using the Var() functions.

	var flagvar int
	func init() {
		flag.IntVar(&flagvar, "flagname", 1234, "help message for flagname")
	}

Or you can create custom flags that satisfy the Value interface (with
pointer receivers) and couple them to flag parsing by

	flag.Var(&flagVal, "name", "help message for flagname")

For such flags, the default value is just the initial value of the variable.

After all flags are defined, call

	flag.Parse()

to parse the command line into the defined flags.

Flags may then be used directly. If you're using the flags themselves,
they are all pointers; if you bind to variables, they're values.

	fmt.Println("ip has value ", *ip)
	fmt.Println("flagvar has value ", flagvar)

After parsing, the arguments following the flags are available as the
slice [flag.Args] or individually as [flag.Arg](i).
The arguments are indexed from 0 through [flag.NArg]-1.

# Command line flag syntax

The following forms are permitted:

	-flag
	--flag   // double dashes are also permitted
	-flag=x
	-flag x  // non-boolean flags only

One or two dashes may be used; they are equivalent.
The last form is not permitted for boolean flags because the
meaning of the command

	cmd -x *

where * is a Unix shell wildcard, will change if there is a file
called 0, false, etc. You must use the -flag=false form to turn
off a boolean flag.

Flag parsing stops just before the first non-flag argument
("-" is a non-flag argument) or after the terminator "--".

Integer flags accept 1234, 0664, 0x1234 and may be negative.
Boolean flags may be:

	1, 0, t, f, T, F, true, false, TRUE, FALSE, True, False

Duration flags accept any input valid for time.ParseDuration.

The default set of command-line flags is controlled by
top-level functions.  The [FlagSet] type allows one to define
independent sets of flags, such as to implement subcommands
in a command-line interface. The methods of [FlagSet] are
analogous to the top-level functions for the command-line
flag set.
*/

/*
 * flag命令解析处理
 */

package flag

import (
	"fmt"
	"io"
	"os"
)

// Value is the interface to the dynamic value stored in a flag.
// (The default value is represented as a string.)
//
// If a Value has an IsBoolFlag() bool method returning true,
// the command-line parser makes -name equivalent to -name=true
// rather than using the next command-line argument.
//
// Set is called once, in command line order, for each flag present.
// The flag package may call the [String] method with a zero-valued receiver,
// such as a nil pointer.
type Value interface {
	String() string
	Set(string) error
}

// ErrorHandling defines how [FlagSet.Parse] behaves if the parse fails.
type ErrorHandling int

// These constants cause [FlagSet.Parse] to behave as described if the parse fails.
const (
	ContinueOnError ErrorHandling = iota // 返回错误但继续执行 Return a descriptive error.
	ExitOnError                          // 错误然后终止程序 Call os.Exit(2) or for -h/-help Exit(0).
	PanicOnError                         // 错误会抛出异常 Call panic with a descriptive error.
)

// A FlagSet represents a set of defined flags. The zero value of a FlagSet
// has no name and has [ContinueOnError] error handling.
//
// [Flag] names must be unique within a FlagSet. An attempt to define a flag whose
// name is already in use will cause a panic.
type FlagSet struct {
	// Usage is the function called when an error occurs while parsing flags.
	// The field is a function (not a method) that may be changed to point to
	// a custom error handler. What happens after Usage is called depends
	// on the ErrorHandling setting; for the command line, this defaults
	// to ExitOnError, which exits the program after calling Usage.
	Usage func() // 用于当参数解析错误的自定义错误处理方式(标记使用方法)

	name          string
	errorHandling ErrorHandling // 错误处理方式
	output        io.Writer     // nil means stderr; use Output() accessor
}

// Output returns the destination for usage and error messages. [os.Stderr] is returned if
// output was not set or was set to nil.
func (f *FlagSet) Output() io.Writer {
	if f.output == nil {
		return os.Stderr
	}
	return f.output
}

// PrintDefaults prints, to standard error unless configured otherwise, the
// default values of all defined command-line flags in the set. See the
// documentation for the global function PrintDefaults for more information.
// todo hank
func (f *FlagSet) PrintDefaults() {

}

func (f *FlagSet) defaultUsage() {
	if f.name == "" {
		fmt.Fprintf(f.Output(), "Usage:\n")
	} else {
		fmt.Fprintf(f.Output(), "Usage of %s:\n", f.name)
	}
	f.PrintDefaults()
}

// todo hank 实现
func (f *FlagSet) Var(value Value, name string, usage string) {

}

// Var defines a flag with the specified name and usage string. The type and
// value of the flag are represented by the first argument, of type [Value], which
// typically holds a user-defined implementation of [Value]. For instance, the
// caller could create a flag that turns a comma-separated string into a slice
// of strings by giving the slice the methods of [Value]; in particular, [Set] would
// decompose the comma-separated string into the slice.
func Var(value Value, name string, usage string) {
	CommandLine.Var(value, name, usage)
}

// CommandLine is the default set of command-line flags, parsed from [os.Args].
// The top-level functions such as [BoolVar], [Arg], and so on are wrappers for the
// methods of CommandLine.
var CommandLine *FlagSet

func init() {
	// It's possible for execl to hand us an empty os.Args.
	if len(os.Args) == 0 {

	}
}

func NewFlagSet(name string, errorHandling ErrorHandling) *FlagSet {
	f := &FlagSet{
		name:          name,
		errorHandling: errorHandling,
	}
	f.Usage = f.defaultUsage
	return f
}
