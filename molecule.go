// Package molecule is a Go library for parsing protobufs in an efficient and zero-allocation manner.
package molecule

import (
	"fmt"
	"io"

	"github.com/richardartoul/molecule/src/codec"
)

// MessageEachFn is a function that will be called for each top-level field in a
// message passed to MessageEach.
type MessageEachFn func(fieldNum int32, value Value) bool

// MessageEach iterates over each top-level field in the message stored in buffer
// and calls fn on each one.
func MessageEach(buffer *codec.Buffer, fn MessageEachFn) error {
	for !buffer.EOF() {
		fieldNum, wireType, err := buffer.DecodeTagAndWireType()
		if err == io.EOF {
			return nil
		}

		value, err := readValueFromBuffer(wireType, buffer)
		if err != nil {
			return fmt.Errorf("MessageEach: error reading value from buffer: %v", err)
		}

		if shouldContinue := fn(fieldNum, value); !shouldContinue {
			return nil
		}
	}
	return nil
}

// PackedRepeatedEachFn is a function that is called for each value in a repeated field.
type PackedRepeatedEachFn func(value Value) bool

// PackedArrayEach iterates over each value in the packed repeated field stored in buffer
// and calls fn on each one.
//
// The fieldType argument should match the type of the value stored in the repeated field.
//
// PackedArrayEach only supports repeated fields encoded using packed encoding.
func PackedArrayEach(buffer *codec.Buffer, fieldType codec.FieldDescriptorProto_Type, fn PackedRepeatedEachFn) error {
	var wireType int8
	switch fieldType {
	case codec.FieldDescriptorProto_TYPE_INT32,
		codec.FieldDescriptorProto_TYPE_INT64,
		codec.FieldDescriptorProto_TYPE_UINT32,
		codec.FieldDescriptorProto_TYPE_UINT64,
		codec.FieldDescriptorProto_TYPE_SINT32,
		codec.FieldDescriptorProto_TYPE_SINT64,
		codec.FieldDescriptorProto_TYPE_BOOL,
		codec.FieldDescriptorProto_TYPE_ENUM:
		wireType = codec.WireVarint
	case codec.FieldDescriptorProto_TYPE_FIXED64,
		codec.FieldDescriptorProto_TYPE_SFIXED64,
		codec.FieldDescriptorProto_TYPE_DOUBLE:
		wireType = codec.WireFixed64
	case codec.FieldDescriptorProto_TYPE_FIXED32,
		codec.FieldDescriptorProto_TYPE_SFIXED32,
		codec.FieldDescriptorProto_TYPE_FLOAT:
		wireType = codec.WireFixed32
	case codec.FieldDescriptorProto_TYPE_STRING,
		codec.FieldDescriptorProto_TYPE_MESSAGE,
		codec.FieldDescriptorProto_TYPE_BYTES:
		wireType = codec.WireBytes
	default:
		return fmt.Errorf(
			"PackedArrayEach: unknown field type: %v", fieldType)
	}

	for !buffer.EOF() {
		value, err := readValueFromBuffer(wireType, buffer)
		if err != nil {
			return fmt.Errorf("ArrayEach: error reading value from buffer: %v", err)
		}
		if shouldContinue := fn(value); !shouldContinue {
			return nil
		}
	}

	return nil
}

func readValueFromBuffer(wireType int8, buffer *codec.Buffer) (Value, error) {
	value := Value{
		WireType: wireType,
	}

	switch wireType {
	case codec.WireVarint:
		varint, err := buffer.DecodeVarint()
		if err != nil {
			return Value{}, fmt.Errorf(
				"MessageEach: error decoding varint: %v", err)
		}
		value.Number = varint
	case codec.WireFixed32:
		fixed32, err := buffer.DecodeFixed32()
		if err != nil {
			return Value{}, fmt.Errorf(
				"MessageEach: error decoding fixed32: %v", err)
		}
		value.Number = fixed32
	case codec.WireFixed64:
		fixed64, err := buffer.DecodeFixed64()
		if err != nil {
			return Value{}, fmt.Errorf(
				"MessageEach: error decoding fixed64: %v", err)
		}
		value.Number = fixed64
	case codec.WireBytes:
		b, err := buffer.DecodeRawBytes(false)
		if err != nil {
			return Value{}, fmt.Errorf(
				"MessageEach: error decoding raw bytes: %v", err)
		}
		value.Bytes = b
	case codec.WireStartGroup, codec.WireEndGroup:
		return Value{}, fmt.Errorf(
			"MessageEach: encountered group wire type: %d. Groups not supported",
			wireType)
	default:
		return Value{}, fmt.Errorf(
			"MessageEach: unknown wireType: %d", wireType)
	}

	return value, nil
}
