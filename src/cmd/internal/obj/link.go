package obj

type Link struct {
	Pkgpath string // the current package's import path 当前导入包的路径
	Std     bool   // is standard library package  是否标准的库包
}

type LinkArch struct{}
