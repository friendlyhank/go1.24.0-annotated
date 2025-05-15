#!/usr/bin/env bash
# Copyright 2009 The Go Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

# See golang.org/s/go15bootstrap for an overview of the build process.

# Environment variables that control make.bash:
#
# GOHOSTARCH: The architecture for host tools (compilers and
# binaries).  Binaries of this type must be executable on the current
# system, so the only common reason to set this is to set
# GOHOSTARCH=386 on an amd64 machine.
# GOHOSTARCH: 主机工具（编译器和二进制文件）的架构。
# 这类二进制文件必须能在当前系统上执行，
# 所以设置它的常见原因是在 amd64 机器上设置 GOHOSTARCH=386。
#
# GOARCH: The target architecture for installed packages and tools.安装包和工具的目标架构。
#
# GOOS: The target operating system for installed packages and tools.安装包和工具的目标操作系统。
#
# GO_GCFLAGS: Additional go tool compile arguments to use when
# building the packages and commands.
#
# GO_LDFLAGS: Additional go tool link arguments to use when
# building the commands.
#
# CGO_ENABLED: Controls cgo usage during the build. Set it to 1
# to include all cgo related files, .c and .go file with "cgo"
# build directive, in the build. Set it to 0 to ignore them.
#
# GO_EXTLINK_ENABLED: Set to 1 to invoke the host linker when building
# packages that use cgo.  Set to 0 to do all linking internally.  This
# controls the default behavior of the linker's -linkmode option.  The
# default value depends on the system.
#
# GO_LDSO: Sets the default dynamic linker/loader (ld.so) to be used
# by the internal linker.
#
# CC: Command line to run to compile C code for GOHOSTARCH.
# Default is "gcc". Also supported: "clang".
#
# CC_FOR_TARGET: Command line to run to compile C code for GOARCH.
# This is used by cgo. Default is CC.
#
# CC_FOR_${GOOS}_${GOARCH}: Command line to run to compile C code for specified ${GOOS} and ${GOARCH}.
# (for example, CC_FOR_linux_arm)
# If this is not set, the build will use CC_FOR_TARGET if appropriate, or CC.
#
# CXX_FOR_TARGET: Command line to run to compile C++ code for GOARCH.
# This is used by cgo. Default is CXX, or, if that is not set,
# "g++" or "clang++".
#
# CXX_FOR_${GOOS}_${GOARCH}: Command line to run to compile C++ code for specified ${GOOS} and ${GOARCH}.
# (for example, CXX_FOR_linux_arm)
# If this is not set, the build will use CXX_FOR_TARGET if appropriate, or CXX.
#
# FC: Command line to run to compile Fortran code for GOARCH.
# This is used by cgo. Default is "gfortran".
#
# PKG_CONFIG: Path to pkg-config tool. Default is "pkg-config".
#
# GO_DISTFLAGS: extra flags to provide to "dist bootstrap".
# (Or just pass them to the make.bash command line.)
#
# GOBUILDTIMELOGFILE: If set, make.bash and all.bash write
# timing information to this file. Useful for profiling where the
# time goes when these scripts run.
#GOBUILDTIMELOGFILE: 如果设置，make.bash 和 all.bash 会将时间信息写入此文件。用于分析脚本运行时间的分布。

# GOROOT_BOOTSTRAP: A working Go tree >= Go 1.22.6 for bootstrap.
# If $GOROOT_BOOTSTRAP/bin/go is missing, $(go env GOROOT) is
# tried for all "go" in $PATH. By default, one of $HOME/go1.22.6,
# $HOME/sdk/go1.22.6, or $HOME/go1.4, whichever exists, in that order.
# We still check $HOME/go1.4 to allow for build scripts that still hard-code
# that name even though they put newer Go toolchains there.
# GOROOT_BOOTSTRAP: 用于引导的 Go 目录树，版本 >= Go 1.22.6。
# 如果 $GOROOT_BOOTSTRAP/bin/go 不存在，会尝试 $PATH 中所有 "go" 的 $(go env GOROOT)。
# 默认会按顺序查找 $HOME/go1.22.6、$HOME/sdk/go1.22.6 或 $HOME/go1.4 中存在的任意一个。
# 我们仍然检查 $HOME/go1.4 是为了兼容那些仍然硬编码该名称的构建脚本，
# 即使它们在那里放置了更新的 Go 工具链。

#构建的最低版本号
bootgo=1.22.6

# 当发生错误时退出
set -e

# 判断run.bash是否存在 -f 测试文件是否存在且是普通文件
if [[ ! -f run.bash ]]; then
    echo 'make.bash must be run from $GOROOT/src' 1>&2
    exit 1
fi

# Test for Windows. 判断是否windows环境
case "$(uname)" in
*MINGW* | *WIN32* | *CYGWIN*)
	echo 'ERROR: Do not use make.bash to build on Windows.'
	echo 'Use make.bat instead.'
	echo
	exit 1
	;;
esac

verbose=false #设置默认的详细输出模式为关闭
vflag="" #初始化一个空的标志变量
if [[ "$1" == "-v" ]]; then
	verbose=true
	vflag=-v
	shift
fi

#检查 GOROOT_BOOTSTRAP 是否已设置，如果已设置则 goroot_bootstrap_set 为 "true"
# 如果未设置，则按照优先级查找可用的Go安装
# $HOME/sdk/go1.22.6
# $HOME/go1.22.6
# $HOME/go1.4
goroot_bootstrap_set=${GOROOT_BOOTSTRAP+"true"}
if [[ -z "$GOROOT_BOOTSTRAP" ]]; then
    GOROOT_BOOTSTRAP="$HOME/go1.4"

    # 遍历可能的 Go 安装目录
   for d in sdk/go$bootgo go$bootgo; do
      if [[ -d "$HOME/$d" ]]; then
      			GOROOT_BOOTSTRAP="$HOME/$d"
      fi
   done
fi
# 设置 GOROOT_BOOTSTRAP 环境变量为找到的 Go 安装路径 (todo hank 先特殊处理)
export GOROOT_BOOTSTRAP

# 它会设置一个干净的环境（清除所有可能影响 Go 命令执行的环境变量）
bootstrapenv() {
	GOROOT="$GOROOT_BOOTSTRAP" GO111MODULE=off GOENV=off GOOS= GOARCH= GOEXPERIMENT= GOFLAGS= "$@"
}

# GOROOT_BOOTSTRAP 找到新版本可编译的工具链路
# 设置go的根目录并设置为环境变量
export GOROOT="$(cd .. && pwd)"
# - type -ap go 查找系统中所有名为 "go" 的可执行文件
IFS=$'\n'; for go_exe in $(type -ap go); do
	if [[ ! -x "$GOROOT_BOOTSTRAP/bin/go" ]]; then
		goroot_bootstrap=$GOROOT_BOOTSTRAP
		GOROOT_BOOTSTRAP=""
		goroot=$(bootstrapenv "$go_exe" env GOROOT)
		GOROOT_BOOTSTRAP=$goroot_bootstrap
		if [[ "$goroot" != "$GOROOT" ]]; then
			if [[ "$goroot_bootstrap_set" == "true" ]]; then
				printf 'WARNING: %s does not exist, found %s from env\n' "$GOROOT_BOOTSTRAP/bin/go" "$go_exe" >&2
				printf 'WARNING: set %s as GOROOT_BOOTSTRAP\n' "$goroot" >&2
			fi
			GOROOT_BOOTSTRAP="$goroot"
		fi
	fi
done; unset IFS
if [[ ! -x "$GOROOT_BOOTSTRAP/bin/go" ]]; then
	echo "ERROR: Cannot find $GOROOT_BOOTSTRAP/bin/go." >&2
	echo "Set \$GOROOT_BOOTSTRAP to a working Go tree >= Go $bootgo." >&2
	exit 1
fi

# Get the exact bootstrap toolchain version to help with debugging.
# We clear GOOS and GOARCH to avoid an ominous but harmless warning if
# the bootstrap doesn't support them.
GOROOT_BOOTSTRAP_VERSION=$(bootstrapenv "$GOROOT_BOOTSTRAP/bin/go" version | sed 's/go version //')
echo "Building Go cmd/dist using $GOROOT_BOOTSTRAP. ($GOROOT_BOOTSTRAP_VERSION)"
if $verbose; then
	echo cmd/dist
fi
rm -f cmd/dist/dist
bootstrapenv "$GOROOT_BOOTSTRAP/bin/go" build -o cmd/dist/dist ./cmd/dist

# -e doesn't propagate out of eval, so check success by hand. 捕获执行结果，报错则退出
# 这个捕获不知道为啥会报错

if $verbose; then
	echo
fi

#  dist bootstrap -a构建工具链
./cmd/dist/dist bootstrap -a $vflag "$@"
rm -f ./cmd/dist/dist