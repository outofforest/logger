package logger

import (
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type array []int

func (a array) MarshalLogArray(marshaler zapcore.ArrayEncoder) error {
	for _, v := range a {
		marshaler.AppendInt(v)
	}
	return nil
}

type object struct {
	Field1 string
	Field2 int
}

type object2 struct {
	Field1 string
	Field2 int
}

func (o object2) MarshalLogObject(marshaler zapcore.ObjectEncoder) error {
	marshaler.AddString("field1", o.Field1)
	marshaler.AddInt("field2", o.Field2)
	return nil
}

func TestPrint(t *testing.T) {
	config := ToolDefaultConfig
	New(config).
		With(zap.String("withField1", "value1"),
			zap.Int("withField2", 2)).
		Error("This is message",
			zap.Array("array", array{0, 1, 2, 3, 4}),
			zap.Any("any", object{Field1: "stringValue", Field2: 3}),
			zap.Any("nil", nil),
			zap.Bool("bool", false),
			zap.Bools("bools", []bool{true, false, true}),
			zap.Binary("binary", []byte("this is string")),
			zap.Complex64s("complex64s", []complex64{complex(13, 12), complex(10, -8)}),
			zap.Complex64("complex64", complex(1, 2)),
			zap.Duration("duration", 2*time.Hour),
			zap.Error(errors.New("this is the error")),
			zap.Errors("errors", []error{errors.New("error1"), errors.New("error2")}),
			zap.Float32("float32", 12.34),
			zap.Float64("float64", 12.34),
			zap.Int("int", 8),
			zap.Object("object", object2{Field1: "stringValue", Field2: 56}),
			zap.Stack("stack"),
			zap.String("string", "value"),
			zap.Time("time", time.Now()),
		)

}
