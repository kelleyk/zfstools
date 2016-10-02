package zfs

/*
#cgo CFLAGS: -I /usr/include/libzfs -I /usr/include/libspl -DHAVE_IOCTL_IN_SYS_IOCTL_H
#cgo LDFLAGS: -lnvpair

#include <libnvpair.h>
*/
import "C"

const (
	// nvp implementation version
	NVVersion = C.NV_VERSION

	// nvlist pack encoding
	NVEncodeNative = 0
	NVEncodeXDR    = 1

	// nvlist lookup pairs related flags
	NVFlagNOENTOK = 0x1 // XXX: "NoEntOK"?
)

// NVListFlags is a bitfield consisting of zero or more of the values defined below.
type NVFlags C.uint_t

const (
	// nvlist persistent unique name flags, stored in nvl_nvflags
	NVUniqueName     NVFlags = 0x1
	NVUniqueNameType         = 0x2
)

type DataType C.data_type_t

// Ref.: https://github.com/zfsonlinux/zfs/blob/master/include/sys/nvpair.h#L42
const (
	TypeUnknown      DataType = C.DATA_TYPE_UNKNOWN
	TypeBoolean               = C.DATA_TYPE_BOOLEAN
	TypeByte                  = C.DATA_TYPE_BYTE
	TypeInt16                 = C.DATA_TYPE_INT16
	TypeUint16                = C.DATA_TYPE_UINT16
	TypeInt32                 = C.DATA_TYPE_INT32
	TypeUint32                = C.DATA_TYPE_UINT32
	TypeInt64                 = C.DATA_TYPE_INT64
	TypeUint64                = C.DATA_TYPE_UINT64
	TypeString                = C.DATA_TYPE_STRING
	TypeByteArray             = C.DATA_TYPE_BYTE_ARRAY
	TypeInt16Array            = C.DATA_TYPE_INT16_ARRAY
	TypeUint16Array           = C.DATA_TYPE_UINT16_ARRAY
	TypeInt32Array            = C.DATA_TYPE_INT32_ARRAY
	TypeUint32Array           = C.DATA_TYPE_UINT32_ARRAY
	TypeInt64Array            = C.DATA_TYPE_INT64_ARRAY
	TypeUint64Array           = C.DATA_TYPE_UINT64_ARRAY
	TypeStringArray           = C.DATA_TYPE_STRING_ARRAY
	TypeTime                  = C.DATA_TYPE_HRTIME
	TypeNVList                = C.DATA_TYPE_NVLIST
	TypeNVListArray           = C.DATA_TYPE_NVLIST_ARRAY
	TypeBooleanValue          = C.DATA_TYPE_BOOLEAN_VALUE
	TypeInt8                  = C.DATA_TYPE_INT8
	TypeUint8                 = C.DATA_TYPE_UINT8
	TypeBooleanArray          = C.DATA_TYPE_BOOLEAN_ARRAY
	TypeInt8Array             = C.DATA_TYPE_INT8_ARRAY
	TypeUint8Array            = C.DATA_TYPE_UINT8_ARRAY
	TypeDouble                = C.DATA_TYPE_DOUBLE
)

func (t DataType) String() string {
	switch t {
	case TypeUnknown:
		return "UNKNOWN"
	case TypeBoolean:
		return "bool" // a flag; always true if present?
	case TypeByte:
		return "byte"
	case TypeInt16:
		return "int16"
	case TypeUint16:
		return "uint16"
	case TypeInt32:
		return "int32"
	case TypeUint32:
		return "uint32"
	case TypeInt64:
		return "int64"
	case TypeUint64:
		return "uint64"
	case TypeString:
		return "string"
	case TypeByteArray:
		return "[]byte"
	case TypeInt16Array:
		return "[]int16"
	case TypeUint16Array:
		return "[]uint16"
	case TypeInt32Array:
		return "[]int32"
	case TypeUint32Array:
		return "[]uint32"
	case TypeInt64Array:
		return "[]int64"
	case TypeUint64Array:
		return "[]uint64"
	case TypeStringArray:
		return "[]string"
	case TypeTime:
		return "time"
	case TypeNVList:
		return "*NVList"
	case TypeNVListArray:
		return "[]*NVList"
	case TypeBooleanValue:
		return "boolValue"
	case TypeInt8:
		return "int8"
	case TypeUint8:
		return "uint8"
	case TypeBooleanArray:
		return "[]bool"
	case TypeInt8Array:
		return "[]int8"
	case TypeUint8Array:
		return "[]uint8"
	case TypeDouble:
		return "double"
	default:
		return "UNEXPECTED_VALUE"
	}
}
