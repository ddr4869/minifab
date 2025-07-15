package logger

import (
	"fmt"
	"sync"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	// Global logger instance
	Logger *zap.SugaredLogger
	// Mutex to protect logger initialization
	loggerMutex sync.RWMutex
	// Track if logger is initialized
	initialized bool
)

// LogLevel represents the logging level
type LogLevel string

const (
	DebugLevel LogLevel = "debug"
	InfoLevel  LogLevel = "info"
	WarnLevel  LogLevel = "warn"
	ErrorLevel LogLevel = "error"
	PanicLevel LogLevel = "panic"
	FatalLevel LogLevel = "fatal"
)

// Config holds the logger configuration
type Config struct {
	Level       LogLevel `json:"level"`
	Development bool     `json:"development"`
	Encoding    string   `json:"encoding"` // "json" or "console"
}

// DefaultConfig returns a default logger configuration
func DefaultConfig() *Config {
	return &Config{
		Level:       InfoLevel,
		Development: false,
		Encoding:    "console",
	}
}

// DevelopmentConfig returns a development logger configuration
func DevelopmentConfig() *Config {
	return &Config{
		Level:       DebugLevel,
		Development: true,
		Encoding:    "console",
	}
}

// Initialize initializes the global logger with the given configuration
func Initialize(config *Config) error {
	loggerMutex.Lock()
	defer loggerMutex.Unlock()

	if config == nil {
		config = DefaultConfig()
	}

	var zapConfig zap.Config

	if config.Development {
		zapConfig = zap.NewDevelopmentConfig()
		zapConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		zapConfig = zap.NewProductionConfig()
	}

	// Set log level
	level, err := zapcore.ParseLevel(string(config.Level))
	if err != nil {
		return err
	}
	zapConfig.Level = zap.NewAtomicLevelAt(level)

	// Set encoding
	zapConfig.Encoding = config.Encoding

	// Customize encoder config for better readability
	zapConfig.EncoderConfig.TimeKey = "timestamp"
	zapConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	zapConfig.EncoderConfig.CallerKey = "caller"
	zapConfig.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder

	// Build logger
	logger, err := zapConfig.Build(zap.AddCallerSkip(1))
	if err != nil {
		return err
	}

	Logger = logger.Sugar()
	initialized = true
	return nil
}

// InitializeDefault initializes the global logger with default configuration
func InitializeDefault() error {
	return Initialize(DefaultConfig())
}

// InitializeDevelopment initializes the global logger with development configuration
func InitializeDevelopment() error {
	return Initialize(DevelopmentConfig())
}

// GetLogger returns the global logger instance
func GetLogger() *zap.SugaredLogger {
	loggerMutex.RLock()
	if Logger != nil {
		defer loggerMutex.RUnlock()
		return Logger
	}
	loggerMutex.RUnlock()

	// Need to initialize - upgrade to write lock
	loggerMutex.Lock()
	defer loggerMutex.Unlock()

	// Double-check pattern - another goroutine might have initialized
	if Logger != nil {
		return Logger
	}

	// Initialize with default config if not already initialized
	if err := initialize(DefaultConfig()); err != nil {
		panic("Failed to initialize default logger: " + err.Error())
	}
	return Logger
}

// initialize is the internal initialization function (without mutex)
func initialize(config *Config) error {
	if config == nil {
		config = DefaultConfig()
	}

	var zapConfig zap.Config

	if config.Development {
		zapConfig = zap.NewDevelopmentConfig()
		zapConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		zapConfig = zap.NewProductionConfig()
	}

	// Set log level
	level, err := zapcore.ParseLevel(string(config.Level))
	if err != nil {
		return err
	}
	zapConfig.Level = zap.NewAtomicLevelAt(level)

	// Set encoding
	zapConfig.Encoding = config.Encoding

	// Customize encoder config for better readability
	zapConfig.EncoderConfig.TimeKey = "timestamp"
	zapConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	zapConfig.EncoderConfig.CallerKey = "caller"
	zapConfig.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder

	// Build logger
	logger, err := zapConfig.Build(zap.AddCallerSkip(1))
	if err != nil {
		return err
	}

	Logger = logger.Sugar()
	initialized = true
	return nil
}

// Debug logs a debug message
func Debug(args ...any) {
	GetLogger().Debug(args...)
}

// Debugf logs a formatted debug message
func Debugf(template string, args ...any) {
	GetLogger().Debugf(template, args...)
}

// Info logs an info message
func Info(args ...any) {
	GetLogger().Info(args...)
}

// Infof logs a formatted info message
func Infof(template string, args ...any) {
	GetLogger().Infof(template, args...)
}

// Warn logs a warning message
func Warn(args ...any) {
	GetLogger().Warn(args...)
}

// Warnf logs a formatted warning message
func Warnf(template string, args ...any) {
	GetLogger().Warnf(template, args...)
}

// Error logs an error message
func Error(args ...any) {
	GetLogger().Error(args...)
}

// Errorf logs a formatted error message
func Errorf(template string, args ...any) {
	GetLogger().Errorf(template, args...)
}

// Fatal logs a fatal message and exits
func Fatal(args ...any) {
	GetLogger().Fatal(args...)
}

// Fatalf logs a formatted fatal message and exits
func Fatalf(template string, args ...any) {
	GetLogger().Fatalf(template, args...)
}

// Panic logs a panic message and panics
func Panic(args ...any) {
	GetLogger().Panic(args...)
}

// Panicf logs a formatted panic message and panics
func Panicf(template string, args ...any) {
	GetLogger().Panicf(template, args...)
}

// With adds structured context to the logger
func With(args ...any) *zap.SugaredLogger {
	return GetLogger().With(args...)
}

// Named creates a named logger
func Named(name string) *zap.SugaredLogger {
	return GetLogger().Named(name)
}

// Sync flushes any buffered log entries
func Sync() error {
	if Logger != nil {
		return Logger.Sync()
	}
	return nil
}

// ErrorAndReturn logs an error message and returns a new error with the same message
func ErrorAndReturn(msg string, args ...any) error {
	logger := GetLogger()

	// Create the formatted message
	var formattedMsg string
	if len(args) > 0 {
		formattedMsg = fmt.Sprintf(msg, args...)
	} else {
		formattedMsg = msg
	}

	// Log the error
	logger.Error(formattedMsg)

	// Return the error
	return errors.New(formattedMsg)
}

// ErrorfAndReturn logs a formatted error message and returns a new error with the same message
func ErrorfAndReturn(template string, args ...any) error {
	logger := GetLogger()

	// Create the formatted message
	formattedMsg := fmt.Sprintf(template, args...)

	// Log the error
	logger.Error(formattedMsg)

	// Return the error
	return errors.New(formattedMsg)
}

// WrapError logs an error with additional context and returns a wrapped error
func WrapError(err error, msg string, args ...any) error {
	if err == nil {
		return nil
	}

	logger := GetLogger()

	// Create the formatted context message
	var contextMsg string
	if len(args) > 0 {
		contextMsg = fmt.Sprintf(msg, args...)
	} else {
		contextMsg = msg
	}

	// Log with structured fields including the original error
	logger.With(
		"error", err.Error(),
		"context", contextMsg,
	).Error("Error occurred with context")

	// Return wrapped error using pkg/errors
	return errors.Wrap(err, contextMsg)
}

// WrapErrorf logs an error with formatted additional context and returns a wrapped error
func WrapErrorf(err error, template string, args ...any) error {
	if err == nil {
		return nil
	}

	logger := GetLogger()

	// Create the formatted context message
	contextMsg := fmt.Sprintf(template, args...)

	// Log with structured fields including the original error
	logger.With(
		"error", err.Error(),
		"context", contextMsg,
	).Error("Error occurred with context")

	// Return wrapped error using pkg/errors
	return errors.Wrap(err, contextMsg)
}

// ErrorWithFields logs an error with structured fields and returns the error
func ErrorWithFields(msg string, fields map[string]any, args ...any) error {
	logger := GetLogger()

	// Create the formatted message
	var formattedMsg string
	if len(args) > 0 {
		formattedMsg = fmt.Sprintf(msg, args...)
	} else {
		formattedMsg = msg
	}

	// Convert fields map to key-value pairs for zap
	var zapFields []any
	for key, value := range fields {
		zapFields = append(zapFields, key, value)
	}

	// Log with structured fields
	logger.With(zapFields...).Error(formattedMsg)

	// Return the error
	return errors.New(formattedMsg)
}

// WarnAndReturn logs a warning message and returns a new error with the same message
func WarnAndReturn(msg string, args ...any) error {
	logger := GetLogger()

	// Create the formatted message
	var formattedMsg string
	if len(args) > 0 {
		formattedMsg = fmt.Sprintf(msg, args...)
	} else {
		formattedMsg = msg
	}

	// Log the warning
	logger.Warn(formattedMsg)

	// Return the error
	return errors.New(formattedMsg)
}

// LogIfError logs an error if it's not nil and returns the same error
func LogIfError(err error, msg string, args ...any) error {
	if err == nil {
		return nil
	}

	logger := GetLogger()

	// Create the formatted context message
	var contextMsg string
	if len(args) > 0 {
		contextMsg = fmt.Sprintf(msg, args...)
	} else {
		contextMsg = msg
	}

	// Log with structured fields
	logger.With(
		"error", err.Error(),
	).Error(contextMsg)

	return err
}

// MustNotError logs a fatal error and exits if err is not nil
func MustNotError(err error, msg string, args ...any) {
	if err == nil {
		return
	}

	logger := GetLogger()

	// Create the formatted context message
	var contextMsg string
	if len(args) > 0 {
		contextMsg = fmt.Sprintf(msg, args...)
	} else {
		contextMsg = msg
	}

	// Log fatal error with structured fields
	logger.With(
		"error", err.Error(),
	).Fatal(contextMsg)
}
