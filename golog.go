// Package golog provides a standardized way to initialize
// github.com/rs/zerolog based loggers.
package golog

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"runtime"
	"time"

	"github.com/KernelPryanic/ctxerr"

	"github.com/rs/zerolog"
)

// Default is the default global logger.
var Default zerolog.Logger

func init() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	zerolog.TimeFieldFormat = time.RFC3339Nano
	Default = New(false, os.Stdout)
	zerolog.DefaultContextLogger = &Default
}

// New creates a new logger instance.
// If out is nil, os.Stdout will be used.
func New(consoleWriter bool, out io.Writer) zerolog.Logger {
	if out == nil {
		out = os.Stdout
	}
	if consoleWriter {
		out = zerolog.ConsoleWriter{
			Out:        out,
			TimeFormat: zerolog.TimeFieldFormat,
		}
	}
	return zerolog.New(out).With().Timestamp().Logger().Hook(callerHook{})
}

var logLevels = map[string]zerolog.Level{
	"trace":    zerolog.TraceLevel,
	"debug":    zerolog.DebugLevel,
	"info":     zerolog.InfoLevel,
	"warn":     zerolog.WarnLevel,
	"error":    zerolog.ErrorLevel,
	"fatal":    zerolog.FatalLevel,
	"panic":    zerolog.PanicLevel,
	"disabled": zerolog.Disabled,
}

// SetLogLevel sets the log level for all logger instances.
// NOTE: Can be called at any time to change the log level.
func SetLogLevel(l string) {
	zerolog.SetGlobalLevel(logLevels[l])
}

// SetTimeFormat sets the time format for all logger instances.
// NOTE: Must be called before `New` to take effect.
func SetTimeFormat(format string) {
	zerolog.TimeFieldFormat = format
}

// CensoredFieldPlaceholder is the default value used to replace censored fields.
const CensoredFieldPlaceholder = "***"

// callerHook is a zerolog hook for post-processing log events.
type callerHook struct{}

var _ zerolog.Hook = callerHook{}

// Run implements the zerolog.Hook interface.
func (h callerHook) Run(e *zerolog.Event, level zerolog.Level, msg string) {
	errCtx := ctxerr.From(e.GetCtx())
	if errCtx.Err != nil {
		e.Fields(errCtx.Ctx)
		e.Err(errCtx.Err)
	}
	switch level {
	case zerolog.ErrorLevel, zerolog.FatalLevel, zerolog.PanicLevel:
		_, file, line, ok := runtime.Caller(3)
		if ok {
			e.Str("file", fmt.Sprintf("%s:%d", file, line))
		}
	}
}

// WithCensoredSecretFields returns a logger context with all fields of struct v
// that have a `secret:"true"` tag censored and replaced with CensoredFieldPlaceholder.
// Nested structs are supported. The rootName is used as a prefix for all field names.
//
// Example:
//
//	type Config struct {
//	    Username string `json:"username"`
//	    Password string `json:"password" secret:"true"`
//	}
//	cfg := Config{Username: "admin", Password: "s3cr3t"}
//	ctx := WithCensoredSecretFields(logger.With(), "config", cfg)
//	logger := ctx.Logger()
//
// This will log:
//
//	{"config.Username":"admin","config.Password":"***"}
func WithCensoredSecretFields(ctx zerolog.Context, rootName string, v any) zerolog.Context {
	censorFields(&ctx, rootName, reflect.ValueOf(v))
	return ctx
}

// censorFields recursively processes struct fields, censoring those marked with secret tag.
func censorFields(ctx *zerolog.Context, path string, v reflect.Value) {
	v, ok := asStruct(v)
	if !ok {
		return
	}

	vType := v.Type()
	for i := 0; i < v.NumField(); i++ {
		fieldType := vType.Field(i)
		fieldValue := v.Field(i)

		// Check if this field is a nested struct
		if nestedStruct, ok := asStruct(fieldValue); ok {
			censorFields(ctx, path+"."+fieldType.Name, nestedStruct)
			continue
		}

		// Add field with censored value if secret, otherwise add actual value
		fieldPath := path + "." + fieldType.Name
		if isFieldSecret(fieldType) {
			*ctx = ctx.Str(fieldPath, CensoredFieldPlaceholder)
		} else {
			*ctx = ctx.Any(fieldPath, fieldValue.Interface())
		}
	}
}

// isFieldSecret checks if a struct field has the secret tag set.
func isFieldSecret(f reflect.StructField) bool {
	return f.Tag.Get("secret") != ""
}

// asStruct dereferences pointers and checks if the value is a struct.
// Returns the dereferenced struct value and true if successful.
func asStruct(v reflect.Value) (reflect.Value, bool) {
	// Dereference all pointer levels
	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return reflect.Value{}, false
		}
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return reflect.Value{}, false
	}

	return v, true
}
