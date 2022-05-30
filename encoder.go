package logger

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/ridge/must"
	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

const encoderName = "better-console"

var bufPool = buffer.NewPool()
var homeDir = must.String(os.UserHomeDir()) + "/"

func init() {
	must.OK(zap.RegisterEncoder(encoderName, func(config zapcore.EncoderConfig) (zapcore.Encoder, error) {
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
	nested  int
	element int
	buffer  *buffer.Buffer
}

func (c *console) AddArray(key string, marshaler zapcore.ArrayMarshaler) error {
	c.addKey(key)
	return c.AppendArray(marshaler)
}

func (c *console) AddObject(key string, marshaler zapcore.ObjectMarshaler) error {
	c.addKey(key)
	return c.AppendObject(marshaler)
}

func (c *console) AddBinary(key string, value []byte) {
	c.addKey(key)
	c.buffer.AppendString(fmt.Sprintf("[% x]", value))
}

func (c *console) AddByteString(key string, value []byte) {
	c.addKey(key)
	c.buffer.AppendString(string(value))
}

func (c *console) AddBool(key string, value bool) {
	c.addKey(key)
	c.buffer.AppendBool(value)
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
}

func (c *console) AddFloat64(key string, value float64) {
	c.addKey(key)
	c.buffer.AppendFloat(value, 64)
}

func (c *console) AddFloat32(key string, value float32) {
	c.addKey(key)
	c.buffer.AppendFloat(float64(value), 32)
}

func (c *console) AddInt(key string, value int) {
	c.addKey(key)
	c.AppendInt(value)
	c.buffer.AppendInt(int64(value))
}

func (c *console) AddInt64(key string, value int64) {
	c.addKey(key)
	c.buffer.AppendInt(value)
}

func (c *console) AddInt32(key string, value int32) {
	c.addKey(key)
	c.buffer.AppendInt(int64(value))
}

func (c *console) AddInt16(key string, value int16) {
	c.addKey(key)
	c.buffer.AppendInt(int64(value))
}

func (c *console) AddInt8(key string, value int8) {
	c.addKey(key)
	c.buffer.AppendInt(int64(value))
}

func (c *console) AddString(key, value string) {
	c.addKey(key)
	c.appendString(value)
}

func (c *console) AddTime(key string, value time.Time) {
	c.addKey(key)
	c.buffer.AppendTime(value.UTC(), "2006-01-02 15:04:05.000")
}

func (c *console) AddUint(key string, value uint) {
	c.addKey(key)
	c.buffer.AppendUint(uint64(value))
}

func (c *console) AddUint64(key string, value uint64) {
	c.addKey(key)
	c.buffer.AppendUint(value)
}

func (c *console) AddUint32(key string, value uint32) {
	c.addKey(key)
	c.buffer.AppendUint(uint64(value))
}

func (c *console) AddUint16(key string, value uint16) {
	c.addKey(key)
	c.buffer.AppendUint(uint64(value))
}

func (c *console) AddUint8(key string, value uint8) {
	c.addKey(key)
	c.buffer.AppendUint(uint64(value))
}

func (c *console) AddUintptr(key string, value uintptr) {
	c.addKey(key)
	c.buffer.AppendUint(uint64(value))
}

func (c *console) AddReflected(key string, value interface{}) error {
	c.addKey(key)
	return c.AppendReflected(value)
}

func (c *console) OpenNamespace(key string) {
	c.buffer.AppendString("\n % ")
	c.buffer.AppendString(key)
	c.buffer.AppendByte(':')
}

func (c *console) Clone() zapcore.Encoder {
	buf := bufPool.Get()
	must.Any(buf.Write(c.buffer.Bytes()))
	return &console{
		nested: c.nested,
		buffer: buf,
	}
}

func (c *console) EncodeEntry(entry zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	buf := bufPool.Get()
	buf.AppendTime(entry.Time.UTC(), "2006-01-02 15:04:05.000 ")
	buf.AppendString(strings.ToUpper(entry.Level.CapitalString()))
	buf.AppendByte(' ')
	if entry.Message != "" {
		buf.AppendString(entry.Message)
		buf.AppendByte(' ')
	}

	if c.buffer.Len() > 0 {
		must.Any(buf.Write(c.buffer.Bytes()))
		buf.AppendString(", ")
	}

	subEncoder := newConsoleEncoder(0)
	defer subEncoder.buffer.Free()
	for _, field := range fields {
		if !subEncoder.appendError(field) {
			field.AddTo(subEncoder)
		}
	}

	must.Any(buf.Write(subEncoder.buffer.Bytes()))

	if entry.LoggerName != "" {
		buf.AppendString("\n # ")
		buf.AppendString(entry.LoggerName)
	}

	buf.AppendString("\n @ ")
	file := entry.Caller.File
	if strings.HasPrefix(file, homeDir) {
		file = "~/" + file[len(homeDir):]
	}
	buf.AppendString(file)
	buf.AppendByte(':')
	buf.AppendInt(int64(entry.Caller.Line))
	buf.AppendByte('\n')

	if entry.Stack != "" {
		buf.AppendString(entry.Stack)
		buf.AppendByte('\n')
	}
	return buf, nil
}

func (c *console) AppendBool(value bool) {
	c.addComma()
	c.buffer.AppendBool(value)
}

func (c *console) AppendByteString(value []byte) {
	c.addComma()
	c.appendString(string(value))
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
}

func (c *console) AppendFloat32(value float32) {
	c.addComma()
	c.buffer.AppendFloat(float64(value), 32)
}

func (c *console) AppendInt(value int) {
	c.addComma()
	c.buffer.AppendInt(int64(value))
}

func (c *console) AppendInt64(value int64) {
	c.addComma()
	c.buffer.AppendInt(value)
}

func (c *console) AppendInt32(value int32) {
	c.addComma()
	c.buffer.AppendInt(int64(value))
}

func (c *console) AppendInt16(value int16) {
	c.addComma()
	c.buffer.AppendInt(int64(value))
}

func (c *console) AppendInt8(value int8) {
	c.addComma()
	c.buffer.AppendInt(int64(value))
}

func (c *console) AppendString(value string) {
	c.addComma()
	c.appendString(value)
}

func (c *console) AppendUint(value uint) {
	c.addComma()
	c.buffer.AppendUint(uint64(value))
}

func (c *console) AppendUint64(value uint64) {
	c.addComma()
	c.buffer.AppendUint(value)
}

func (c *console) AppendUint32(value uint32) {
	c.addComma()
	c.buffer.AppendUint(uint64(value))
}

func (c *console) AppendUint16(value uint16) {
	c.addComma()
	c.buffer.AppendUint(uint64(value))
}

func (c *console) AppendUint8(value uint8) {
	c.addComma()
	c.buffer.AppendUint(uint64(value))
}

func (c *console) AppendUintptr(value uintptr) {
	c.addComma()
	c.buffer.AppendUint(uint64(value))
}

func (c *console) AppendDuration(value time.Duration) {
	c.addComma()
	c.buffer.AppendString(value.String())
}

func (c *console) AppendTime(value time.Time) {
	c.addComma()
	c.buffer.AppendTime(value.UTC(), "2006-01-02 15:04:05.000")
}

func (c *console) AppendArray(marshaler zapcore.ArrayMarshaler) error {
	subEncoder := newConsoleEncoder(0)
	defer subEncoder.buffer.Free()

	if err := marshaler.MarshalLogArray(subEncoder); err != nil {
		return err
	}

	c.addComma()
	c.buffer.AppendByte('[')
	must.Any(c.buffer.Write(subEncoder.buffer.Bytes()))
	c.buffer.AppendByte(']')
	return nil
}

func (c *console) AppendObject(marshaler zapcore.ObjectMarshaler) error {
	subEncoder := newConsoleEncoder(c.nested + 1)
	defer subEncoder.buffer.Free()

	if err := marshaler.MarshalLogObject(subEncoder); err != nil {
		return err
	}

	c.addComma()
	c.buffer.AppendByte('{')
	must.Any(c.buffer.Write(subEncoder.buffer.Bytes()))
	c.buffer.AppendString("\n ")
	c.buffer.AppendString(c.indentation())
	c.buffer.AppendByte('}')
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
		return c.appendReflectedSequence(v)
	case reflect.Slice:
		if v.IsNil() {
			c.appendNil()
		} else {
			return c.appendReflectedSequence(v)
		}
	case reflect.Map:
		if v.IsNil() {
			c.appendNil()
		} else {
			return c.appendReflectedMapping(v)
		}
	case reflect.Ptr, reflect.Interface:
		if v.IsNil() {
			c.appendNil()
		} else {
			return c.AppendReflected(v.Elem().Interface())
		}
	case reflect.Struct:
		return c.appendReflectedStruct(v)
	case reflect.String:
		c.AppendString(v.String())
	default:
		return errors.Errorf("unable to serialize %s", v.Kind())
	}
	return nil
}

func (c *console) indentation() string {
	var res string
	for i := 0; i < c.nested; i++ {
		res += "  "
	}
	return res
}

func (c *console) addComma() {
	if c.element > 0 {
		c.buffer.AppendString(", ")
	}
	c.element++
}

func (c *console) addKey(key string) {
	c.element = 0
	c.buffer.AppendByte('\n')
	c.buffer.AppendString(c.indentation())
	c.buffer.AppendString(" - ")
	c.buffer.AppendString(key)
	c.buffer.AppendString(": ")
}

func (c *console) appendNil() {
	c.buffer.AppendString("<nil>")
}

func (c *console) appendError(field zapcore.Field) bool {
	if field.Type == zapcore.ErrorType {
		err := field.Interface.(error)
		errStack, ok := err.(stackTracer)
		if ok {
			stack := errStack.StackTrace()
			if len(stack) > 0 {
				c.addKey(field.Key)
				c.buffer.AppendString(err.Error())
				ind := "\n     " + c.indentation()
				for _, frame := range stack {
					c.buffer.AppendString(ind)
					c.buffer.AppendString(strings.ReplaceAll(string(must.Bytes(frame.MarshalText())), homeDir, "~/"))
				}
				return true
			}
		}
	}
	return false
}

func (c *console) appendString(value string) {
	if strings.Contains(value, "\n") {
		ind := c.indentation()
		c.buffer.AppendByte('\n')
		c.buffer.AppendString(ind)
		c.buffer.AppendString("     \"")
		c.buffer.AppendString(strings.ReplaceAll(value, "\n", "\n      "+ind))
	} else {
		c.buffer.AppendByte('"')
		c.buffer.AppendString(value)
	}
	c.buffer.AppendByte('"')
}

func (c *console) appendComplex128(value complex128) {
	re, im := real(value), imag(value)
	c.buffer.AppendString(strconv.FormatFloat(re, 'g', -1, 64))
	if im >= 0 {
		c.buffer.AppendString("+")
	}
	c.buffer.AppendString(strconv.FormatFloat(im, 'g', -1, 64))
}

func (c *console) appendReflectedSequence(v reflect.Value) error {
	return c.AppendArray(zapcore.ArrayMarshalerFunc(func(enc zapcore.ArrayEncoder) error {
		n := v.Len()
		for i := 0; i < n; i++ {
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
		n := t.NumField()
		for i := 0; i < n; i++ {
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

type stackTracer interface {
	StackTrace() errors.StackTrace
}
