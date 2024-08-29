package logger

import (
	"encoding/hex"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

var bufPool = buffer.NewPool()

func init() {
	lo.Must0(zap.RegisterEncoder(string(FormatYAML), func(config zapcore.EncoderConfig) (zapcore.Encoder, error) {
		return newConsoleEncoder(0), nil
	}))
}

func newConsoleEncoder(nested int) *console {
	return &console{
		nested: nested,
		buffer: bufPool.Get(),
	}
}

type console struct {
	nested                 int
	element                int
	ignoreFirstIndentation bool
	array                  bool
	skipErrorStackTrace    bool
	containsStackTrace     bool
	buffer                 *buffer.Buffer
}

func (c *console) AddArray(key string, marshaler zapcore.ArrayMarshaler) error {
	c.addKey(key)
	c.buffer.AppendByte('\n')
	return c.AppendArray(marshaler)
}

func (c *console) AddObject(key string, marshaler zapcore.ObjectMarshaler) error {
	c.addKey(key)
	c.buffer.AppendByte('\n')
	return c.AppendObject(marshaler)
}

func (c *console) AddBinary(key string, value []byte) {
	c.addKey(key)
	c.buffer.AppendByte('"')
	c.buffer.AppendString(hex.EncodeToString(value))
	c.buffer.AppendByte('"')
	c.buffer.AppendByte('\n')
}

func (c *console) AddByteString(key string, value []byte) {
	c.addKey(key)
	c.buffer.AppendString(string(value))
	c.buffer.AppendByte('\n')
}

func (c *console) AddBool(key string, value bool) {
	c.addKey(key)
	c.buffer.AppendBool(value)
	c.buffer.AppendByte('\n')
}

func (c *console) AddComplex128(key string, value complex128) {
	c.addKey(key)
	c.appendComplex128(value)
}

func (c *console) AddComplex64(key string, value complex64) {
	c.addKey(key)
	c.appendComplex128(complex128(value))
}

func (c *console) AddDuration(key string, value time.Duration) {
	c.addKey(key)
	c.buffer.AppendString(value.String())
	c.buffer.AppendByte('\n')
}

func (c *console) AddFloat64(key string, value float64) {
	c.addKey(key)
	c.buffer.AppendFloat(value, 64)
	c.buffer.AppendByte('\n')
}

func (c *console) AddFloat32(key string, value float32) {
	c.addKey(key)
	c.buffer.AppendFloat(float64(value), 32)
	c.buffer.AppendByte('\n')
}

func (c *console) AddInt(key string, value int) {
	c.addKey(key)
	c.buffer.AppendInt(int64(value))
	c.buffer.AppendByte('\n')
}

func (c *console) AddInt64(key string, value int64) {
	c.addKey(key)
	c.buffer.AppendInt(value)
	c.buffer.AppendByte('\n')
}

func (c *console) AddInt32(key string, value int32) {
	c.addKey(key)
	c.buffer.AppendInt(int64(value))
	c.buffer.AppendByte('\n')
}

func (c *console) AddInt16(key string, value int16) {
	c.addKey(key)
	c.buffer.AppendInt(int64(value))
	c.buffer.AppendByte('\n')
}

func (c *console) AddInt8(key string, value int8) {
	c.addKey(key)
	c.buffer.AppendInt(int64(value))
	c.buffer.AppendByte('\n')
}

func (c *console) AddString(key, value string) {
	c.addKey(key)
	appendString(c.buffer, value, c.indentation())
}

func (c *console) AddTime(key string, value time.Time) {
	c.addKey(key)
	c.buffer.AppendTime(value.UTC(), "2006-01-02 15:04:05.000")
	c.buffer.AppendByte('\n')
}

func (c *console) AddUint(key string, value uint) {
	c.addKey(key)
	c.buffer.AppendUint(uint64(value))
	c.buffer.AppendByte('\n')
}

func (c *console) AddUint64(key string, value uint64) {
	c.addKey(key)
	c.buffer.AppendUint(value)
	c.buffer.AppendByte('\n')
}

func (c *console) AddUint32(key string, value uint32) {
	c.addKey(key)
	c.buffer.AppendUint(uint64(value))
	c.buffer.AppendByte('\n')
}

func (c *console) AddUint16(key string, value uint16) {
	c.addKey(key)
	c.buffer.AppendUint(uint64(value))
	c.buffer.AppendByte('\n')
}

func (c *console) AddUint8(key string, value uint8) {
	c.addKey(key)
	c.buffer.AppendUint(uint64(value))
	c.buffer.AppendByte('\n')
}

func (c *console) AddUintptr(key string, value uintptr) {
	c.addKey(key)
	c.buffer.AppendUint(uint64(value))
	c.buffer.AppendByte('\n')
}

func (c *console) AddReflected(key string, value interface{}) error {
	c.addKey(key)
	return c.AppendReflected(value)
}

func (c *console) OpenNamespace(key string) {
	c.nested = 0
	c.addKey(key)
	c.buffer.AppendByte('\n')
	c.nested = 1
}

func (c *console) Clone() zapcore.Encoder {
	buf := bufPool.Get()
	lo.Must(buf.Write(c.buffer.Bytes()))
	return &console{
		nested:              c.nested,
		array:               c.array,
		skipErrorStackTrace: c.skipErrorStackTrace,
		containsStackTrace:  c.containsStackTrace,
		buffer:              buf,
	}
}

func (c *console) EncodeEntry(entry zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	buf := bufPool.Get()
	buf.AppendString(`- log: "`)
	buf.AppendTime(entry.Time.UTC(), "2006-01-02 15:04:05.000 ")
	buf.AppendString(strings.ToUpper(entry.Level.CapitalString()))
	buf.AppendByte(' ')
	if entry.Message != "" {
		buf.AppendString(entry.Message)
	}
	buf.AppendString("\"\n  details:\n")

	if c.buffer.Len() > 0 {
		lo.Must(buf.Write(c.buffer.Bytes()))
	}

	subEncoder := newConsoleEncoder(0)
	if entry.Level == zap.InfoLevel {
		subEncoder.skipErrorStackTrace = true
	}
	defer subEncoder.buffer.Free()
	for _, field := range fields {
		if !subEncoder.appendError(field) {
			field.AddTo(subEncoder)
		}
	}

	lo.Must(buf.Write(subEncoder.buffer.Bytes()))

	buf.AppendString("    logged:\n")
	if entry.LoggerName != "" {
		buf.AppendString(`      by: "`)
		buf.AppendString(entry.LoggerName)
		buf.AppendString("\"\n")
	}

	buf.AppendString(`      at: "`)
	buf.AppendString(entry.Caller.File)
	buf.AppendByte(':')
	buf.AppendInt(int64(entry.Caller.Line))
	buf.AppendString("\"\n")

	if !c.containsStackTrace && !subEncoder.containsStackTrace && entry.Stack != "" {
		buf.AppendString(`      stack: `)
		appendString(buf, entry.Stack, "  ")
	}
	return buf, nil
}

func (c *console) AppendBool(value bool) {
	c.addComma()
	c.buffer.AppendBool(value)
	c.buffer.AppendByte('\n')
}

func (c *console) AppendByteString(value []byte) {
	c.addComma()
	appendString(c.buffer, string(value), c.indentation())
}

func (c *console) AppendComplex128(value complex128) {
	c.addComma()
	c.appendComplex128(value)
}

func (c *console) AppendComplex64(value complex64) {
	c.addComma()
	c.appendComplex128(complex128(value))
}

func (c *console) AppendFloat64(value float64) {
	c.addComma()
	c.buffer.AppendFloat(value, 64)
	c.buffer.AppendByte('\n')
}

func (c *console) AppendFloat32(value float32) {
	c.addComma()
	c.buffer.AppendFloat(float64(value), 32)
	c.buffer.AppendByte('\n')
}

func (c *console) AppendInt(value int) {
	c.addComma()
	c.buffer.AppendInt(int64(value))
	c.buffer.AppendByte('\n')
}

func (c *console) AppendInt64(value int64) {
	c.addComma()
	c.buffer.AppendInt(value)
	c.buffer.AppendByte('\n')
}

func (c *console) AppendInt32(value int32) {
	c.addComma()
	c.buffer.AppendInt(int64(value))
	c.buffer.AppendByte('\n')
}

func (c *console) AppendInt16(value int16) {
	c.addComma()
	c.buffer.AppendInt(int64(value))
	c.buffer.AppendByte('\n')
}

func (c *console) AppendInt8(value int8) {
	c.addComma()
	c.buffer.AppendInt(int64(value))
	c.buffer.AppendByte('\n')
}

func (c *console) AppendString(value string) {
	c.addComma()
	appendString(c.buffer, value, c.indentation())
}

func (c *console) AppendUint(value uint) {
	c.addComma()
	c.buffer.AppendUint(uint64(value))
	c.buffer.AppendByte('\n')
}

func (c *console) AppendUint64(value uint64) {
	c.addComma()
	c.buffer.AppendUint(value)
	c.buffer.AppendByte('\n')
}

func (c *console) AppendUint32(value uint32) {
	c.addComma()
	c.buffer.AppendUint(uint64(value))
	c.buffer.AppendByte('\n')
}

func (c *console) AppendUint16(value uint16) {
	c.addComma()
	c.buffer.AppendUint(uint64(value))
	c.buffer.AppendByte('\n')
}

func (c *console) AppendUint8(value uint8) {
	c.addComma()
	c.buffer.AppendUint(uint64(value))
	c.buffer.AppendByte('\n')
}

func (c *console) AppendUintptr(value uintptr) {
	c.addComma()
	c.buffer.AppendUint(uint64(value))
	c.buffer.AppendByte('\n')
}

func (c *console) AppendDuration(value time.Duration) {
	c.addComma()
	c.buffer.AppendString(value.String())
	c.buffer.AppendByte('\n')
}

func (c *console) AppendTime(value time.Time) {
	c.addComma()
	c.buffer.AppendTime(value.UTC(), "2006-01-02 15:04:05.000")
	c.buffer.AppendByte('\n')
}

func (c *console) AppendArray(marshaler zapcore.ArrayMarshaler) error {
	subEncoder := newConsoleEncoder(c.nested + 1)
	subEncoder.array = true
	defer subEncoder.buffer.Free()

	if err := marshaler.MarshalLogArray(subEncoder); err != nil {
		return errors.WithStack(err)
	}

	c.addComma()
	lo.Must(c.buffer.Write(subEncoder.buffer.Bytes()))
	return nil
}

func (c *console) AppendObject(marshaler zapcore.ObjectMarshaler) error {
	subEncoder := newConsoleEncoder(c.nested + 1)
	if c.array {
		subEncoder.nested--
		subEncoder.ignoreFirstIndentation = true
	}
	defer subEncoder.buffer.Free()

	if err := marshaler.MarshalLogObject(subEncoder); err != nil {
		return errors.WithStack(err)
	}

	c.addComma()
	lo.Must(c.buffer.Write(subEncoder.buffer.Bytes()))
	return nil
}

func (c *console) AppendReflected(value interface{}) error {
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Invalid:
		c.appendNil()
	case reflect.Bool:
		c.AppendBool(v.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		c.AppendInt64(v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		c.AppendUint64(v.Uint())
	case reflect.Float32, reflect.Float64:
		c.AppendFloat64(v.Float())
	case reflect.Complex64, reflect.Complex128:
		c.AppendComplex128(v.Complex())
	case reflect.Array:
		c.buffer.AppendByte('\n')
		return c.appendReflectedSequence(v)
	case reflect.Slice:
		if v.IsNil() {
			c.appendNil()
		} else {
			c.buffer.AppendByte('\n')
			return c.appendReflectedSequence(v)
		}
	case reflect.Map:
		if v.IsNil() {
			c.appendNil()
		} else {
			c.buffer.AppendByte('\n')
			return c.appendReflectedMapping(v)
		}
	case reflect.Ptr, reflect.Interface:
		if v.IsNil() {
			c.appendNil()
		} else {
			return c.AppendReflected(v.Elem().Interface())
		}
	case reflect.Struct:
		switch v.Type() {
		case reflect.TypeOf(time.Time{}):
			c.AppendTime(v.Interface().(time.Time))
		default:
			c.buffer.AppendByte('\n')
			return c.appendReflectedStruct(v)
		}
	case reflect.String:
		c.AppendString(v.String())
	default:
		return errors.Errorf("unable to serialize %s", v.Kind())
	}
	return nil
}

func (c *console) indentation() string {
	var res string
	for range c.nested {
		res += "  "
	}
	return res
}

func (c *console) addComma() {
	if c.array {
		c.buffer.AppendString(c.indentation())
		c.buffer.AppendString("  - ")
	}
}

func (c *console) addKey(key string) {
	if !c.ignoreFirstIndentation || c.element > 0 {
		c.buffer.AppendString(c.indentation())
		c.buffer.AppendString("    ")
	}
	c.buffer.AppendString(key)
	c.buffer.AppendString(": ")
	c.element++
}

func (c *console) appendNil() {
	c.buffer.AppendString("null\n")
}

func (c *console) appendError(field zapcore.Field) bool {
	//nolint:nestif
	if field.Type == zapcore.ErrorType {
		c.addKey(field.Key)

		ind := "\n" + c.indentation()
		err := field.Interface.(error)
		c.buffer.AppendString(ind)
		c.buffer.AppendString("      msg: ")
		c.buffer.AppendByte('"')
		c.buffer.AppendString(err.Error())
		c.buffer.AppendString("\"\n")

		if !c.skipErrorStackTrace {
			errStack, ok := err.(stackTracer)
			if ok {
				stack := errStack.StackTrace()
				if len(stack) > 0 {
					c.buffer.AppendString("      stack:")
					for _, frame := range stack {
						c.buffer.AppendString(ind)
						c.buffer.AppendString("      - \"")
						c.buffer.AppendString(string(lo.Must(frame.MarshalText())))
						c.buffer.AppendByte('"')
					}
					c.buffer.AppendByte('\n')
					c.containsStackTrace = true
					return true
				}
			}
		}
		return true
	}
	return false
}

func (c *console) appendComplex128(value complex128) {
	re, im := real(value), imag(value)
	c.buffer.AppendString(strconv.FormatFloat(re, 'g', -1, 64))
	if im >= 0 {
		c.buffer.AppendString("+")
	}
	c.buffer.AppendString(strconv.FormatFloat(im, 'g', -1, 64))
	c.buffer.AppendByte('\n')
}

func (c *console) appendReflectedSequence(v reflect.Value) error {
	return c.AppendArray(zapcore.ArrayMarshalerFunc(func(enc zapcore.ArrayEncoder) error {
		for i := range v.Len() {
			if err := enc.AppendReflected(v.Index(i).Interface()); err != nil {
				return err
			}
		}
		return nil
	}))
}

func (c *console) appendReflectedMapping(v reflect.Value) error {
	return c.AppendObject(zapcore.ObjectMarshalerFunc(func(enc zapcore.ObjectEncoder) error {
		iter := v.MapRange()
		for iter.Next() {
			if err := enc.AddReflected(iter.Key().String(), iter.Value().Interface()); err != nil {
				return err
			}
		}
		return nil
	}))
}

func (c *console) appendReflectedStruct(v reflect.Value) error {
	return c.AppendObject(zapcore.ObjectMarshalerFunc(func(enc zapcore.ObjectEncoder) error {
		t := v.Type()
		for i := range t.NumField() {
			f := t.Field(i)
			if !f.IsExported() {
				continue
			}
			if err := enc.AddReflected(f.Name, v.FieldByIndex(f.Index).Interface()); err != nil {
				return err
			}
		}
		return nil
	}))
}

func appendString(buffer *buffer.Buffer, value string, indentation string) {
	if strings.Contains(value, "\n") {
		buffer.AppendString("\n")
		buffer.AppendString(indentation)
		buffer.AppendString("      \"")
		buffer.AppendString(strings.ReplaceAll(value, "\n", "\n       "+indentation))
	} else {
		buffer.AppendByte('"')
		buffer.AppendString(value)
	}
	buffer.AppendByte('"')
	buffer.AppendByte('\n')
}

type stackTracer interface {
	StackTrace() errors.StackTrace
}
