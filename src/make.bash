#!/usr/bin/env bash
# Copyright 2009 The Go Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

# See golang.org/s/go15bootstrap for an overview of the build process.

# Environment variables that control make.bash:
#

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
goroot_bootstrap_set=${GOROOT_BOOTSTRAP+"true"}

# 如果未设置，则按照优先级查找可用的Go安装
# $HOME/sdk/go1.22.6
# $HOME/go1.22.6
# $HOME/go1.4
if [[ -z "$GOROOT_BOOTSTRAP" ]]; then
    GOROOT_BOOTSTRAP="$HOME/go1.4"

    # 遍历可能的 Go 安装目录
   for d in sdk/go$bootgo go$bootgo; do
      if [[ -d "$HOME/$d" ]]; then
      			GOROOT_BOOTSTRAP="$HOME/$d"
      fi
   done
fi
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