package ssagen

import "cmd/internal/obj"

var Arch ArchInfo

// interface to back end
type ArchInfo struct {
	LinkArch *obj.LinkArch
}
