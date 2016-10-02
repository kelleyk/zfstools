package zfs

//#include <libnvpair.h>
import "C"
import (
	"strings"
	"syscall"
)

// NVList corresponds to nvlist_t.
//
// /- nvlist header -/
// typedef struct nvlist {
// int32_t		nvl_version;
// uint32_t	nvl_nvflag;	/- persistent flags -/
// uint64_t	nvl_priv;	/- ptr to private data if not packed -/
// uint32_t	nvl_flag;
// int32_t		nvl_pad;	/- currently not used, for alignment -/
// } nvlist_t;
type NVList C.nvlist_t

// N.B.: This implementation is borrowed from nathan7/go-nvpair.

func NewNVList(flags NVFlags) *NVList {
	var l *C.nvlist_t
	if errno := C.nvlist_alloc(&l, C.uint_t(flags), 0); errno != 0 {
		panic(syscall.Errno(errno))
	}
	return (*NVList)(l)
}

func NVListFromPointer(ptr *C.nvlist_t) *NVList {
	return (*NVList)(ptr)
}

func (l *NVList) Free() {
	C.nvlist_free((*C.nvlist_t)(l))
}

func (l *NVList) Flags() NVFlags {
	return NVFlags(C.nvlist_nvflag((*C.nvlist_t)(l)))
}

func (l *NVList) Dup() *NVList {
	// xxx: cp
	nc := (*C.nvlist_t)(l)
	var nvl *C.nvlist_t
	if ret := C.nvlist_dup(nc, &nvl, 0); ret != 0 {
		panic(syscall.Errno(ret))
	}
	return (*NVList)(nvl)
}

func (l *NVList) Next(p *NVPair) *NVPair {
	// xxx: cp
	nc := (*C.nvlist_t)(l)
	nvpC := (*C.nvpair_t)(p)
	return (*NVPair)(C.nvlist_next_nvpair(nc, nvpC))
}

func (l *NVList) String() string {
	var parts []string

	var p *NVPair
	for {
		p = l.Next(p)
		if p == nil {
			break
		}

		parts = append(parts, p.ValueString())
	}

	return strings.Join(parts, ", ")
}
