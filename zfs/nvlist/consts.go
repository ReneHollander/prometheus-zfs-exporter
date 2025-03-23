package nvlist

type NVType uint32

const (
	TypeUnknown NVType = iota
	TypeBoolean
	TypeByte
	TypeInt16
	TypeUint16
	TypeInt32
	TypeUint32
	TypeInt64
	TypeUint64
	TypeString
	TypeByteArray
	TypeInt16Array
	TypeUint16Array
	TypeInt32Array
	TypeUint32Array
	TypeInt64Array
	TypeUint64Array
	TypeStringArray
	TypeHrtime
	TypeNvlist
	TypeNvlistArray
	TypeBooleanValue
	TypeInt8
	TypeUint8
	TypeBooleanArray
	TypeInt8Array
	TypeUint8Array
	TypeDouble
)

func (t NVType) String() string {
	switch t {
	case TypeBoolean:
		return "boolean"
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
		return "bytearray"
	case TypeInt16Array:
		return "int16array"
	case TypeUint16Array:
		return "uint16array"
	case TypeInt32Array:
		return "int32array"
	case TypeUint32Array:
		return "uint32array"
	case TypeInt64Array:
		return "int64array"
	case TypeUint64Array:
		return "uint64array"
	case TypeStringArray:
		return "stringarray"
	case TypeHrtime:
		return "hrtime"
	case TypeNvlist:
		return "nvlist"
	case TypeNvlistArray:
		return "nvlistarray"
	case TypeBooleanValue:
		return "booleanvalue"
	case TypeInt8:
		return "int8"
	case TypeUint8:
		return "uint8"
	case TypeBooleanArray:
		return "booleanarray"
	case TypeInt8Array:
		return "int8array"
	case TypeUint8Array:
		return "uint8array"
	case TypeDouble:
		return "double"
	case TypeUnknown:
		fallthrough
	default:
		return "unknown"
	}
}
