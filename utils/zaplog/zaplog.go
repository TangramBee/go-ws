package zaplog

import (
	"net"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

func init() {

	config := Config{
		true,
		true,
		"/data/logs/go/",
		"go_university_circles.log",
		512,
		30,
		7,
	}

	Configure(config)

}

func Logger() *zap.Logger {
	return DefaultZapLogger
}

// zaplog.Trace(req.Id).Info("11111111111")
func Trace(id string) *zap.Logger {
	return DefaultZapLogger.With(zap.String("id", id))
}

// Configuration for logging
type Config struct {
	// EncodeLogsAsJson makes the log framework log JSON
	EncodeLogsAsJson bool
	// FileLoggingEnabled makes the framework log to a file
	// the fields below can be skipped if this value is false!
	FileLoggingEnabled bool
	// Directory to log to to when filelogging is enabled
	Directory string
	// Filename is the name of the logfile which will be placed inside the directory
	Filename string
	// MaxSize the max size in MB of the logfile before it's rolled
	MaxSize int
	// MaxBackups the max number of rolled files to keep
	MaxBackups int
	// MaxAge the max age in days to keep a logfile
	MaxAge int
}

var LocalIp interface{}

// How to log, by example:
// logger.Info("Importing new file, zap.String("source", filename), zap.Int("size", 1024))
// To log a stacktrace:
// Logger.Warn("It went wrong, zap.Stack())

// DefaultZapLogger is the default logger instance that should be used to log
// It's assigned a default value here for tests (which do not call log.Configure())
var DefaultZapLogger = newZapLogger(false, os.Stdout)

// Debug Log a message at the debug level. Messages include any context that's
// accumulated on the logger, as well as any fields added at the log site.
//
// Use zap.String(key, value), zap.Int(key, value) to log fields. These fields
// will be marshalled as JSON in the logfile and key value pairs in the console!
func Debug(msg string, fields ...zapcore.Field) {
	DefaultZapLogger.Debug(msg, fields...)
}

// Info log a message at the info level. Messages include any context that's
// accumulated on the logger, as well as any fields added at the log site.
//
// Use zap.String(key, value), zap.Int(key, value) to log fields. These fields
// will be marshalled as JSON in the logfile and key value pairs in the console!
func Info(msg string, fields ...zapcore.Field) {
	DefaultZapLogger.Info(msg, fields...)
}

// Warn log a message at the warn level. Messages include any context that's
// accumulated on the logger, as well as any fields added at the log site.
//
// Use zap.String(key, value), zap.Int(key, value) to log fields. These fields
// will be marshalled as JSON in the logfile and key value pairs in the console!
func Warn(msg string, fields ...zapcore.Field) {
	DefaultZapLogger.Warn(msg, fields...)
}

// Error Log a message at the error level. Messages include any context that's
// accumulated on the logger, as well as any fields added at the log site.
//
// Use zap.String(key, value), zap.Int(key, value) to log fields. These fields
// will be marshalled as JSON in the logfile and key value pairs in the console!
func Error(msg string, fields ...zapcore.Field) {
	DefaultZapLogger.Warn(msg, fields...)
}

// Panic Log a message at the Panic level. Messages include any context that's
// accumulated on the logger, as well as any fields added at the log site.
//
// Use zap.String(key, value), zap.Int(key, value) to log fields. These fields
// will be marshalled as JSON in the logfile and key value pairs in the console!
func Panic(msg string, fields ...zapcore.Field) {
	DefaultZapLogger.Panic(msg, fields...)
}

// Fatal Log a message at the fatal level. Messages include any context that's
// accumulated on the logger, as well as any fields added at the log site.
//
// Use zap.String(key, value), zap.Int(key, value) to log fields. These fields
// will be marshalled as JSON in the logfile and key value pairs in the console!
func Fatal(msg string, fields ...zapcore.Field) {
	DefaultZapLogger.Fatal(msg, fields...)
}

// Time constructs a Field with the given key and value. The encoder
// controls how the time is serialized.
func Time(key string, val time.Time) zapcore.Field {
	return zap.Time(key, val)
}

// Duration constructs a field with the given key and value. The encoder
// controls how the duration is serialized.
func Duration(key string, val time.Duration) zapcore.Field {
	return zap.Duration(key, val)
}

// Int constructs a field with the given key and value.
func Int(key string, val int) zapcore.Field {
	return zap.Int(key, val)
}

// Float64 constructs a field that carries a float64. The way the
// floating-point value is represented is encoder-dependent, so marshaling is
// necessarily lazy.
func Float64(key string, val float64) zapcore.Field {
	return zap.Float64(key, val)
}

// String constructs a field with the given key and value.
func String(key string, val string) zapcore.Field {
	return zap.String(key, val)
}

// Object constructs a field with the given key and ObjectMarshaler. It
// provides a flexible, but still type-safe and efficient, way to add map- or
// struct-like user-defined types to the logging context. The struct's
// MarshalLogObject method is called lazily.
func Object(key string, val zapcore.ObjectMarshaler) zapcore.Field {
	return zap.Object(key, val)
}

// Any takes a key and an arbitrary value and chooses the best way to represent
// them as a field, falling back to a reflection-based approach only if
// necessary.
//
// Since byte/uint8 and rune/int32 are aliases, Any can't differentiate between
// them. To minimize surprises, []byte values are treated as binary blobs, byte
// values are treated as uint8, and runes are always treated as integers.
func Any(key string, val interface{}) zapcore.Field {
	return zap.Any(key, val)
}

// Reflect constructs a field with the given key and an arbitrary object. It uses
// an encoding-appropriate, reflection-based function to lazily serialize nearly
// any object into the logging context, but it's relatively slow and
// allocation-heavy. Outside tests, Any is always a better choice.
//
// If encoding fails (e.g., trying to serialize a map[int]string to JSON), Reflect
// includes the error message in the final log output.
func Reflect(key string, val interface{}) zapcore.Field {
	return zap.Reflect(key, val)
}

// AtLevel logs the message at a specific log level
func AtLevel(level zapcore.Level, msg string, fields ...zapcore.Field) {
	switch level {
	case zapcore.DebugLevel:
		Debug(msg, fields...)
	case zapcore.PanicLevel:
		Panic(msg, fields...)
	case zapcore.ErrorLevel:
		Error(msg, fields...)
	case zapcore.WarnLevel:
		Warn(msg, fields...)
	case zapcore.InfoLevel:
		Info(msg, fields...)
	case zapcore.FatalLevel:
		Fatal(msg, fields...)
	default:
		Warn("Logging at unkown level", zap.Any("level", level))
		Warn(msg, fields...)
	}
}

// Configure sets up the logging framework
//
// In production, the container logs will be collected and file logging should be disabled. However,
// during development it's nicer to see logs as text and optionally write to a file when debugging
// problems in the containerized pipeline
//
// The output log file will be located at /var/log/auth-logic/auth-logic.log and
// will be rolled when it reaches 20MB with a maximum of 1 backup.
func Configure(config Config) {
	writers := []zapcore.WriteSyncer{}
	if config.FileLoggingEnabled {
		writers = append(writers, newRollingFile(config))
	}

	DefaultZapLogger = newZapLogger(config.EncodeLogsAsJson, zapcore.NewMultiWriteSyncer(writers...))
	zap.RedirectStdLog(DefaultZapLogger)

}

func newRollingFile(config Config) zapcore.WriteSyncer {
	if err := os.MkdirAll(config.Directory, 0775); err != nil {
		Error("failed create log directory", zap.Error(err), zap.String("path", config.Directory))
		return nil
	}

	return zapcore.AddSync(&lumberjack.Logger{
		Filename:   config.Directory + config.Filename,
		MaxSize:    config.MaxSize,    //megabytes
		MaxAge:     config.MaxAge,     //days
		MaxBackups: config.MaxBackups, //files
	})
}

// GetLogger method
func GetLogger(config Config) *zap.Logger {
	writers := []zapcore.WriteSyncer{}
	if config.FileLoggingEnabled {
		writers = append(writers, newRollingFile(config))
	}

	logger := newZapLogger(config.EncodeLogsAsJson, zapcore.NewMultiWriteSyncer(writers...))
	return logger
}

func newZapLogger(encodeAsJSON bool, output zapcore.WriteSyncer) *zap.Logger {

	encCfg := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	encoder := zapcore.NewConsoleEncoder(encCfg)
	if encodeAsJSON {
		encoder = zapcore.NewJSONEncoder(encCfg)
	}

	if LocalIp == nil {
		LocalIp = GetLocalIP()
	}
	logger := zap.New(zapcore.NewCore(encoder, output, zap.NewAtomicLevel()), zap.AddCaller(), zap.AddCallerSkip(1))
	return logger.With(zap.String("localIp", LocalIp.(string)))
}

// getLocalIP method
func GetLocalIP() string {
	var addr string = ""
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return addr
	}
	for _, v := range addrs {
		if ipnet, ok := v.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				addr = ipnet.IP.String()
			}

		}
	}
	return addr
}
