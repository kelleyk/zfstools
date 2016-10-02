package zfs

/*
#cgo CFLAGS: -I /usr/include/libzfs -I /usr/include/libspl -DHAVE_IOCTL_IN_SYS_IOCTL_H
#cgo LDFLAGS: -lnvpair

#include <libnvpair.h>
*/
import "C"
import (
	"time"
	"unsafe"
)

func copyinBool(value C.boolean_t) bool {
	if value == C.B_TRUE {
		return true
	}
	return false
}
func copyinTime(value C.hrtime_t) time.Time {
	return time.Unix(0, int64(value))
}
func copyinBools(p *C.boolean_t, n C.uint_t) []bool {
	dst := make([]bool, n)
	src := (*[1 << 30]C.boolean_t)(unsafe.Pointer(p))
	for i := 0; i < int(n); i += 1 {
		if src[i] == C.B_TRUE {
			dst[i] = true
		} else {
			dst[i] = false
		}
	}
	return dst
}
func copyinBytes(p *C.uchar_t, n C.uint_t) []byte {
	return copyinUint8s((*C.uint8_t)(p), n)
}
func copyinUint8s(p *C.uint8_t, n C.uint_t) []uint8 {
	dst := make([]uint8, n)
	src := (*[1 << 30]uint8)(unsafe.Pointer(p))
	copy(dst, src[:])
	return dst
}
func copyinInt8s(p *C.int8_t, n C.uint_t) []int8 {
	dst := make([]int8, n)
	src := (*[1 << 30]int8)(unsafe.Pointer(p))
	copy(dst, src[:])
	return dst
}
func copyinUint16s(p *C.uint16_t, n C.uint_t) []uint16 {
	dst := make([]uint16, n)
	src := (*[1 << 30]uint16)(unsafe.Pointer(p))
	copy(dst, src[:])
	return dst
}
func copyinInt16s(p *C.int16_t, n C.uint_t) []int16 {
	dst := make([]int16, n)
	src := (*[1 << 30]int16)(unsafe.Pointer(p))
	copy(dst, src[:])
	return dst
}
func copyinUint32s(p *C.uint32_t, n C.uint_t) []uint32 {
	dst := make([]uint32, n)
	src := (*[1 << 30]uint32)(unsafe.Pointer(p))
	copy(dst, src[:])
	return dst
}
func copyinInt32s(p *C.int32_t, n C.uint_t) []int32 {
	dst := make([]int32, n)
	src := (*[1 << 30]int32)(unsafe.Pointer(p))
	copy(dst, src[:])
	return dst
}
func copyinUint64s(p *C.uint64_t, n C.uint_t) []uint64 {
	dst := make([]uint64, n)
	src := (*[1 << 30]uint64)(unsafe.Pointer(p))
	copy(dst, src[:])
	return dst
}
func copyinInt64s(p *C.int64_t, n C.uint_t) []int64 {
	dst := make([]int64, n)
	src := (*[1 << 30]int64)(unsafe.Pointer(p))
	copy(dst, src[:])
	return dst
}
func copyinStrings(p **C.char, n C.uint_t) []string {
	dst := make([]string, n)
	src := (*[1 << 30]*C.char)(unsafe.Pointer(p))
	for i := 0; i < int(n); i += 1 {
		dst[i] = C.GoString(src[i])
	}
	return dst
}
func copyinNVLists(p **C.nvlist_t, n C.uint_t) []*NVList {
	dst := make([]*NVList, n)
	src := (*[1 << 30]*NVList)(unsafe.Pointer(p))
	copy(dst, src[:])
	return dst
}
