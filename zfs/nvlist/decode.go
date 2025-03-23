// Package nvlist implements encoding and decoding of ZFS-style nvlists with an interface similar to
// that of encoding/json. It supports "native" encoding and parts of XDR in both big and little endian.
package nvlist

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"reflect"
	"slices"
	"strings"
	"unsafe"
)

var (
	ErrInvalidEncoding  = errors.New("this nvlist is not in native encoding")
	ErrInvalidEndianess = errors.New("this nvlist is neither in big nor in little endian")
	ErrInvalidData      = errors.New("this nvlist contains invalid data")
	ErrInvalidValue     = errors.New("the value provided to unmarshal contains invalid types")
	ErrUnsupportedType  = errors.New("this nvlist contains an unsupported type (hrtime)")
)

// Encoding represents the encoding used for serialization/deserialization
type Encoding uint8

const (
	// EncodingNative is used in syscalls and cache files
	EncodingNative Encoding = 0x00
	// EncodingXDR is used on-disk (and is not actually XDR)
	EncodingXDR  Encoding = 0x01
	bigEndian             = 0x00
	littleEndian          = 0x01
)

// Unmarshal parses a ZFS-style nvlist in native encoding and with any endianness
func Unmarshal(data []byte, val interface{}) error {
	r := NVListReader{Data: data}
	return r.Unmarshal(reflect.ValueOf(val))
}

type NVListReader struct {
	Data []byte
	pos  int

	// endianness  binary.ByteOrder
	encoding  Encoding
	alignment int
	flags     uint32
	version   int32

	nameBytes    []byte
	numElements  int
	dataPos      int
	dataLen      int
	currentToken NVType
}

func (r *NVListReader) readByte() (byte, error) {
	if r.pos < len(r.Data) {
		val := r.Data[r.pos]
		r.pos++
		return val, nil
	}
	return 0x00, ErrInvalidData
}

func (r *NVListReader) skipN(n int) {
	r.pos += n
}

func (r *NVListReader) readNvHeader() error {
	encoding, err := r.readByte()
	if err != nil {
		return err
	}
	switch Encoding(encoding) {
	case EncodingNative:
		r.encoding = EncodingNative
		r.alignment = 8
	case EncodingXDR:
		r.encoding = EncodingXDR
		r.alignment = 4
	default:
		return ErrInvalidEncoding
	}

	endiness, err := r.readByte()
	if err != nil {
		return err
	}

	var e binary.ByteOrder
	switch endiness {
	case bigEndian:
		e = binary.BigEndian
	case littleEndian:
		e = binary.LittleEndian
	default:
		return ErrInvalidEndianess
	}

	if e == binary.BigEndian {
		return fmt.Errorf("conversion from big endian not yet supported")
	}

	r.skipN(2) // reserved

	r.version, err = r.readInt32()
	if err != nil {
		return err
	}

	r.flags, err = r.readUint32()
	if err != nil {
		return err
	}

	return nil
}

func (r *NVListReader) readInt32() (i int32, err error) {
	if r.pos+4 >= len(r.Data) {
		err = ErrInvalidData
		return
	}
	i = int32(binary.NativeEndian.Uint32(r.Data[r.pos:]))
	r.pos += 4
	return
}

func (r *NVListReader) readInt16() (i int16, err error) {
	if r.pos+2 >= len(r.Data) {
		err = ErrInvalidData
		return
	}
	i = int16(binary.NativeEndian.Uint16(r.Data[r.pos:]))
	r.pos += 2
	return
}

func (r *NVListReader) readUint32() (i uint32, err error) {
	if r.pos+4 >= len(r.Data) {
		err = ErrInvalidData
		return
	}
	i = uint32(binary.NativeEndian.Uint32(r.Data[r.pos:]))
	r.pos += 4
	return
}

func (r *NVListReader) readBytes(n int) (b []byte, err error) {
	if r.pos+n >= len(r.Data) {
		err = ErrInvalidData
		return
	}
	b = r.Data[r.pos : r.pos+n]
	r.pos += n
	return
}

func (r *NVListReader) Next() (NVType, error) {
	if r.pos == 0 {
		err := r.readNvHeader()
		if err != nil {
			return TypeUnknown, err
		}
	}

	startPos := r.pos

	size, err := r.readInt32()
	if err != nil {
		return TypeUnknown, err
	}
	if size < 0 {
		return TypeUnknown, ErrInvalidData
	}
	if size == 0 { // End indicated by zero size
		return TypeUnknown, io.EOF
	}
	if r.pos+int(size) >= len(r.Data) {
		return TypeUnknown, ErrInvalidData
	}

	nextNVPairPos := startPos + int(size)

	if r.encoding == EncodingXDR {
		r.skipN(4) // Skip decoded size, it's irrelevant for us
	}

	nameSize, err := r.readInt16()
	if err != nil {
		return TypeUnknown, err
	}
	if nameSize <= 0 { // Null terminated, so at least size 1 is required
		return TypeUnknown, ErrInvalidData
	}

	// read 'reserve' and discard it.
	_, err = r.readInt16()
	if err != nil {
		return TypeUnknown, err
	}

	numElements, err := r.readInt32()
	if err != nil {
		return TypeUnknown, err
	}

	if numElements < 0 || numElements > 65535 { // 64K entries are enough
		return TypeUnknown, ErrInvalidData
	}
	r.numElements = int(numElements)

	nvTypeUInt32, err := r.readUint32()
	if err != nil {
		return TypeUnknown, err
	}
	nvType := NVType(nvTypeUInt32)

	nameBytes, err := r.readBytes(int(nameSize))
	if err != nil {
		return TypeUnknown, err
	}
	// Remove the null terminator
	r.nameBytes = nameBytes[:len(nameBytes)-1]

	if (r.pos-startPos)%r.alignment != 0 {
		r.pos += r.alignment - ((r.pos - startPos) % r.alignment)
	}

	r.dataPos = r.pos
	r.dataLen = nextNVPairPos - r.pos
	r.pos = nextNVPairPos
	r.currentToken = nvType

	if nvType == TypeStringArray {
		// ignore the space for the pointers
		r.dataPos += 8 * r.NumElements()
	}

	return nvType, nil
}
func (r *NVListReader) Token() NVType {
	return r.currentToken
}

func (r *NVListReader) NameBytes() []byte {
	return r.nameBytes
}

func (r *NVListReader) Name() string {
	return unsafe.String(unsafe.SliceData(r.nameBytes), len(r.nameBytes))
}

func (r *NVListReader) NumElements() int {
	return r.numElements
}

func (r *NVListReader) UInt8() uint8 {
	return r.Data[r.dataPos]
}

func (r *NVListReader) UInt8Array() []uint8 {
	return unsafe.Slice((*uint8)(unsafe.Pointer(unsafe.SliceData(r.Data[r.dataPos:r.dataPos+r.dataLen]))), r.NumElements())
}

func (r *NVListReader) Int8() int8 {
	return int8(r.Data[r.dataPos])
}

func (r *NVListReader) Int8Array() []int8 {
	return unsafe.Slice((*int8)(unsafe.Pointer(unsafe.SliceData(r.Data[r.dataPos:r.dataPos+r.dataLen]))), r.NumElements())
}

func (r *NVListReader) UInt16() uint16 {
	return binary.NativeEndian.Uint16(r.Data[r.dataPos : r.dataPos+r.dataLen])
}

func (r *NVListReader) UInt16Array() []uint16 {
	return unsafe.Slice((*uint16)(unsafe.Pointer(unsafe.SliceData(r.Data[r.dataPos:r.dataPos+r.dataLen]))), r.NumElements())
}

func (r *NVListReader) Int16() int16 {
	return int16(binary.NativeEndian.Uint16(r.Data[r.dataPos : r.dataPos+r.dataLen]))
}

func (r *NVListReader) Int16Array() []int16 {
	return unsafe.Slice((*int16)(unsafe.Pointer(unsafe.SliceData(r.Data[r.dataPos:r.dataPos+r.dataLen]))), r.NumElements())
}

func (r *NVListReader) UInt32() uint32 {
	return binary.NativeEndian.Uint32(r.Data[r.dataPos : r.dataPos+r.dataLen])
}

func (r *NVListReader) UInt32Array() []uint32 {
	return unsafe.Slice((*uint32)(unsafe.Pointer(unsafe.SliceData(r.Data[r.dataPos:r.dataPos+r.dataLen]))), r.NumElements())
}

func (r *NVListReader) Int32() int32 {
	return int32(binary.NativeEndian.Uint32(r.Data[r.dataPos : r.dataPos+r.dataLen]))
}

func (r *NVListReader) Int32Array() []int32 {
	return unsafe.Slice((*int32)(unsafe.Pointer(unsafe.SliceData(r.Data[r.dataPos:r.dataPos+r.dataLen]))), r.NumElements())
}

func (r *NVListReader) UInt64() uint64 {
	return binary.NativeEndian.Uint64(r.Data[r.dataPos : r.dataPos+r.dataLen])
}

func (r *NVListReader) UInt64Array() []uint64 {
	return unsafe.Slice((*uint64)(unsafe.Pointer(unsafe.SliceData(r.Data[r.dataPos:r.dataPos+r.dataLen]))), r.NumElements())
}

func (r *NVListReader) Int64() int64 {
	return int64(binary.NativeEndian.Uint64(r.Data[r.dataPos : r.dataPos+r.dataLen]))
}

func (r *NVListReader) Int64Array() []int64 {
	return unsafe.Slice((*int64)(unsafe.Pointer(unsafe.SliceData(r.Data[r.dataPos:r.dataPos+r.dataLen]))), r.NumElements())
}

func (r *NVListReader) Byte() byte {
	return r.Data[r.dataPos]
}

func (r *NVListReader) ByteArray() []byte {
	return r.Data[r.dataPos : r.dataPos+r.dataLen]
}

func (r *NVListReader) Boolean() (bool, error) {
	switch r.Int32() {
	case 0:
		return false, nil
	case 1:
		return true, nil
	default:
		return false, ErrInvalidData
	}
}

func (r *NVListReader) BooleanArray(dst []bool) ([]bool, error) {
	dst = slices.Grow(dst, r.NumElements())
	dst = dst[:0]
	for range r.NumElements() {
		s, err := r.Boolean()
		if err != nil {
			return nil, err
		}
		dst = append(dst, s)
	}
	return dst, nil
}

func (r *NVListReader) BytesUntilDelimiter(delim byte) ([]byte, error) {
	for i := range r.dataLen {
		if r.Data[r.dataPos+i] == delim {
			return r.Data[r.dataPos : r.dataPos+i], nil
		}
	}
	return nil, ErrInvalidData
}

func (r *NVListReader) String() (string, error) {
	b, err := r.BytesUntilDelimiter(0x00)
	if err != nil {
		return "", err
	}
	return unsafe.String(unsafe.SliceData(b), len(b)), nil
}

func (r *NVListReader) StringArray(dst []string) ([]string, error) {
	dst = slices.Grow(dst, r.NumElements())
	dst = dst[:0]
	for range r.NumElements() {
		s, err := r.String()
		if err != nil {
			return nil, err
		}
		dst = append(dst, s)
	}
	return dst, nil
}

func (r *NVListReader) StringArraySafe(dst []string) ([]string, error) {
	dst = slices.Grow(dst, r.NumElements())
	dst = dst[:0]
	for range r.NumElements() {
		s, err := r.String()
		if err != nil {
			return nil, err
		}
		dst = append(dst, strings.Clone(s))
	}
	return dst, nil
}

func (r *NVListReader) Value(val any) error {
	_, err := binary.Decode(r.Data[r.dataPos:r.dataPos+r.dataLen], binary.NativeEndian, val)
	if err != nil {
		return err
	}
	return nil
}

func (r *NVListReader) Skip() error {
	for {
		token, err := r.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		if token == TypeNvlist {
			err = r.Skip()
			if err != nil {
				return err
			}
		} else if token == TypeNvlistArray {
			numElements := r.NumElements()
			for range numElements {
				err = r.Skip()
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (r *NVListReader) Unmarshal(v reflect.Value) error {
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() == reflect.Interface && v.NumMethod() == 0 {
		val := make(map[string]interface{})
		v.Set(reflect.ValueOf(val))
		v = v.Elem()
	}
	structFieldByName := make(map[string]reflect.Value)
	if v.Kind() == reflect.Struct {
		t := v.Type()
		for i := range t.NumField() {
			field := t.Field(i)
			tags := strings.Split(field.Tag.Get("nvlist"), ",")
			name := field.Name
			if tags[0] != "" {
				name = tags[0]
			}
			structFieldByName[name] = v.Field(i)
		}
	} else if v.Kind() == reflect.Map {
		// Noop, but valid
	} else {
		return ErrInvalidData
	}

	for {
		token, err := r.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		name := strings.Clone(r.Name())

		setPrimitive := func(value interface{}) {
			rValue := reflect.ValueOf(value)
			if rValue.Kind() == reflect.Ptr {
				rValue = rValue.Elem()
			}
			if v.Kind() == reflect.Struct {
				field := structFieldByName[name]
				if field.CanSet() {
					field.Set(rValue)
				}
			} else if v.Kind() == reflect.Map {
				v.SetMapIndex(reflect.ValueOf(name), rValue)
			}
		}

		switch token {
		case TypeUnknown:
			return ErrInvalidData
		case TypeBoolean:
			setPrimitive(true)
		case TypeInt16:
			setPrimitive(r.Int16())
		case TypeUint16:
			setPrimitive(r.UInt16())
		case TypeInt32:
			setPrimitive(r.Int32())
		case TypeUint32:
			setPrimitive(r.UInt32())
		case TypeInt64:
			setPrimitive(r.Int64())
		case TypeUint64:
			setPrimitive(r.UInt64())
		case TypeInt8:
			setPrimitive(r.Int8())
		case TypeUint8:
			setPrimitive(r.UInt8())
		case TypeByte:
			setPrimitive(r.Byte())
		case TypeString:
			s, err := r.String()
			if err != nil {
				return err
			}
			setPrimitive(strings.Clone(s))
		case TypeBooleanValue:
			b, err := r.Boolean()
			if err != nil {
				return err
			}
			setPrimitive(b)
		case TypeInt8Array:
			setPrimitive(slices.Clone(r.Int8Array()))
		case TypeUint8Array:
			setPrimitive(slices.Clone(r.UInt8Array()))
		case TypeInt16Array:
			setPrimitive(slices.Clone(r.Int16Array()))
		case TypeUint16Array:
			setPrimitive(slices.Clone(r.UInt16Array()))
		case TypeInt32Array:
			setPrimitive(slices.Clone(r.Int32Array()))
		case TypeUint32Array:
			setPrimitive(slices.Clone(r.UInt32Array()))
		case TypeInt64Array:
			setPrimitive(slices.Clone(r.Int64Array()))
		case TypeUint64Array:
			setPrimitive(slices.Clone(r.UInt64Array()))
		case TypeByteArray:
			setPrimitive(slices.Clone(r.ByteArray()))
		case TypeStringArray:
			val, err := r.StringArraySafe(nil)
			if err != nil {
				return err
			}
			setPrimitive(val)
		case TypeBooleanArray:
			val, err := r.BooleanArray(nil)
			if err != nil {
				return err
			}
			setPrimitive(val)
		case TypeNvlist:
			if v.Kind() == reflect.Struct {
				field := structFieldByName[name]
				if field.CanSet() {
					if err := r.Unmarshal(field); err != nil {
						return err
					}
				}
			} else if v.Kind() == reflect.Map {
				valueType := v.Type().Elem()
				var val reflect.Value
				if valueType.Kind() == reflect.Interface {
					val = reflect.ValueOf(make(map[string]interface{}))
				} else if valueType.Kind() == reflect.Struct {
					val = reflect.New(valueType)
				} else if valueType.Kind() == reflect.Map {
					val = reflect.MakeMap(reflect.MapOf(reflect.TypeOf(""), valueType.Elem()))
				} else {
					return fmt.Errorf("complex hybrid types not supported")
				}
				if err := r.Unmarshal(val); err != nil {
					return err
				}
				if val.Kind() == reflect.Ptr {
					v.SetMapIndex(reflect.ValueOf(name), val.Elem())
				} else {
					v.SetMapIndex(reflect.ValueOf(name), val)
				}
			} else {
				return fmt.Errorf("invalid pair type (not map or struct)")
			}
		case TypeNvlistArray:
			var val reflect.Value
			if v.Kind() == reflect.Struct {
				return fmt.Errorf("deserializing NVListArrays into structs currently unsupported")
			} else if v.Kind() == reflect.Map {
				numElements := r.NumElements()
				val = reflect.MakeSlice(reflect.SliceOf(reflect.TypeOf(map[string]any{})), numElements, numElements)
				for i := range numElements { // arraySize is <2^16
					val.Index(i).Set(reflect.MakeMap(val.Type().Elem()))
					err := r.Unmarshal(val.Index(i))
					if err != nil {
						return err
					}
				}
				v.SetMapIndex(reflect.ValueOf(name), val)
			} else {
				return fmt.Errorf("invalid pair type (not map or struct)")
			}
		}
	}
	return nil
}
