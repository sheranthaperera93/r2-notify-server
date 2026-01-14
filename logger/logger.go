package logger

import (
	"bytes"
	"os"
	"time"

	"r2-notify/config"
	"r2-notify/data"

	ai "github.com/microsoft/ApplicationInsights-Go/appinsights"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var Log *Logger

// TestSink holds a buffer and a logger for testing.
type TestSink struct {
	Buffer *bytes.Buffer
	Logger *Logger
}

type Logger struct {
	zapLogger *zap.Logger
	aiClient  ai.TelemetryClient
	useAzure  bool
	minLevel  zapcore.Level
}

type LogPayload struct {
	Service       string    // e.g. "r2-notify"
	Component     string    // e.g. "eventhub-consumer"
	Operation     string    // e.g. "ReceiveEvent"
	Message       string    // human-readable message
	CorrelationId string    // trace ID for distributed tracing
	UserId        string    // optional
	AppId         string    // optional
	Error         error     // optional
	Timestamp     time.Time // auto-populated
}

func Init() {
	Log = NewLogger()
}

// NewLogger initializes and returns a Logger instance based on the environment.
//
// Behavior:
//   - If the environment is set to "azure" and a valid Application Insights
//     instrumentation key is provided, the logger is configured to send logs
//     to Azure Application Insights using a TelemetryClient.
//   - Otherwise, the logger is configured to write structured JSON logs to
//     a local file with rotation, using Zap and Lumberjack.
//
// The local file logger writes to "./logs/r2-notify.log" with the following
// rotation settings:
//   - MaxSize:    10 MB per log file
//   - MaxBackups: 5 rotated files retained
//   - MaxAge:     30 days
//   - Compress:   true (old logs are compressed)
//
// This design allows the same logging API to be used across environments,
// while automatically routing logs to the appropriate sink.
//
// Example usage:
//
//	// Local environment (logs to file)
//	log := logger.NewLogger("local", "")
//	log.Info(logger.LogPayload{
//	    Service:   "r2-notify",
//	    Component: "main",
//	    Operation: "Startup",
//	    Message:   "Service started",
//	})
//
//	// Azure environment (logs to Application Insights)
//	log := logger.NewLogger("azure", os.Getenv("APP_INSIGHTS_INSTRUMENTATION_KEY"))
//	log.Info(logger.LogPayload{
//	    Service:   "r2-notify",
//	    Component: "eventhub-consumer",
//	    Operation: "ReceiveEvent",
//	    Message:   "Connected to Event Hub",
//	})
func NewLogger() *Logger {
	instrumentationKey := config.LoadConfig().AppInsightsInstrumentationKey
	if config.LoadConfig().LogMethod == data.LOG_METHOD_AZURE && instrumentationKey != "" {
		client := ai.NewTelemetryClient(instrumentationKey)
		return &Logger{aiClient: client, useAzure: true}
	}

	// File logger with rotation
	fileWriter := zapcore.AddSync(&lumberjack.Logger{
		Filename:   config.LoadConfig().LogFilePath,
		MaxSize:    config.LoadConfig().MaxLogFileSize,
		MaxBackups: 5,
		MaxAge:     30, // days
		Compress:   true,
	})

	// Console writer for stdout
	consoleWriter := zapcore.AddSync(os.Stdout)

	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "timestamp"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	// Get filtered log level from config
	filteredLevel := getLogLevel()

	fileCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		fileWriter,
		filteredLevel,
	)

	consoleCore := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderCfg),
		consoleWriter,
		filteredLevel,
	)

	core := zapcore.NewTee(fileCore, consoleCore)

	return &Logger{zapLogger: zap.New(core), useAzure: false}
}

func NewTestSink(level zapcore.Level) *TestSink {
	buf := &bytes.Buffer{}
	ws := zapcore.AddSync(buf)

	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "timestamp"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		ws,
		level,
	)

	return &TestSink{
		Buffer: buf,
		Logger: &Logger{zapLogger: zap.New(core), useAzure: false},
	}
}

// Info logs an informational-level message with a structured payload.
//
// The method enforces a consistent logging schema by requiring a LogPayload,
// which includes fields such as Service, Component, Operation, CorrelationId,
// UserId, AppId, and Message. A timestamp is automatically added.
//
// Behavior:
//   - If the logger is configured for Azure (useAzure == true), the payload is
//     converted into a TraceTelemetry object and sent to Application Insights,
//     with all fields attached as custom properties. The severity level used
//     is Information.
//   - Otherwise, the payload is written to the local Zap logger, which outputs
//     structured JSON logs (typically to file with rotation).
//
// This ensures that both local logs and cloud logs share the same schema,
// making them easy to query and correlate across environments.
//
// Example usage:
//
//	logPayload := logger.LogPayload{
//	    Service:       "r2-notify",
//	    Component:     "eventhub-consumer",
//	    Operation:     "ReceiveEvent",
//	    Message:       "Received new notification event",
//	    CorrelationId: "abc-xyz-123",
//	    UserId:        "user-42",
//	    AppId:         "my-app",
//	}
//	log.Info(logPayload)
func (l *Logger) Info(payload LogPayload) {
	if !l.shouldLog(zap.InfoLevel) {
		return
	}
	payload.Timestamp = time.Now()
	if l.useAzure {
		trace := ai.NewTraceTelemetry(payload.Message, ai.Information)
		trace.Properties["service"] = payload.Service
		trace.Properties["component"] = payload.Component
		trace.Properties["operation"] = payload.Operation
		trace.Properties["correlationId"] = payload.CorrelationId
		trace.Properties["userId"] = payload.UserId
		trace.Properties["appId"] = payload.AppId
		l.aiClient.Track(trace)
	} else {
		l.zapLogger.Info(payload.Message,
			zap.String("service", payload.Service),
			zap.String("component", payload.Component),
			zap.String("operation", payload.Operation),
			zap.String("correlationId", payload.CorrelationId),
			zap.String("userId", payload.UserId),
			zap.String("appId", payload.AppId),
			zap.Time("timestamp", payload.Timestamp),
		)
	}
}

// Debug logs a debug-level message with a structured payload.
//
// The method enforces a consistent logging schema by requiring a LogPayload,
// which includes fields such as Service, Component, Operation, CorrelationId,
// UserId, AppId, and Message. A timestamp is automatically added.
//
// Behavior:
//   - If the logger is configured for Azure (useAzure == true), the payload is
//     converted into a TraceTelemetry object and sent to Application Insights,
//     with all fields attached as custom properties. The severity level used
//     is Verbose, which corresponds to debug-level logging.
//   - Otherwise, the payload is written to the local Zap logger, which outputs
//     structured JSON logs (typically to file with rotation).
//
// This ensures that both local logs and cloud logs share the same schema,
// making them easy to query and correlate across environments.
//
// Example usage:
//
//	logPayload := logger.LogPayload{
//	    Service:       "r2-notify",
//	    Component:     "eventhub-consumer",
//	    Operation:     "ReceiveEvent",
//	    Message:       "Debugging event payload parsing",
//	    CorrelationId: "abc-xyz-123",
//	    UserId:        "user-42",
//	    AppId:         "my-app",
//	}
//	log.Debug(logPayload)
func (l *Logger) Debug(payload LogPayload) {
	if !l.shouldLog(zap.DebugLevel) {
		return
	}
	payload.Timestamp = time.Now()
	if l.useAzure {
		trace := ai.NewTraceTelemetry(payload.Message, ai.Verbose)
		trace.Properties["service"] = payload.Service
		trace.Properties["component"] = payload.Component
		trace.Properties["operation"] = payload.Operation
		trace.Properties["correlationId"] = payload.CorrelationId
		trace.Properties["userId"] = payload.UserId
		trace.Properties["appId"] = payload.AppId
		l.aiClient.Track(trace)
	} else {
		l.zapLogger.Debug(payload.Message,
			zap.String("service", payload.Service),
			zap.String("component", payload.Component),
			zap.String("operation", payload.Operation),
			zap.String("correlationId", payload.CorrelationId),
			zap.String("userId", payload.UserId),
			zap.String("appId", payload.AppId),
			zap.Time("timestamp", payload.Timestamp),
		)
	}
}

// Warn logs a warning-level message with a structured payload.
//
// The method enforces a consistent logging schema by requiring a LogPayload,
// which includes fields such as Service, Component, Operation, CorrelationId,
// UserId, AppId, and Message. A timestamp is automatically added.
//
// Behavior:
//   - If the logger is configured for Azure (useAzure == true), the payload is
//     converted into a TraceTelemetry object and sent to Application Insights,
//     with all fields attached as custom properties.
//   - Otherwise, the payload is written to the local Zap logger, which outputs
//     structured JSON logs (typically to file with rotation).
//
// This ensures that both local logs and cloud logs share the same schema,
// making them easy to query and correlate across environments.
//
// Example usage:
//
//	logPayload := logger.LogPayload{
//	    Service:       "r2-notify",
//	    Component:     "eventhub-consumer",
//	    Operation:     "ReceiveEvent",
//	    Message:       "Partition lag detected",
//	    CorrelationId: "abc-xyz-123",
//	    UserId:        "user-42",
//	    AppId:         "my-app",
//	}
//	log.Warn(logPayload)
func (l *Logger) Warn(payload LogPayload) {
	if !l.shouldLog(zap.WarnLevel) {
		return
	}
	payload.Timestamp = time.Now()
	if l.useAzure {
		trace := ai.NewTraceTelemetry(payload.Message, ai.Warning)
		trace.Properties["service"] = payload.Service
		trace.Properties["component"] = payload.Component
		trace.Properties["operation"] = payload.Operation
		trace.Properties["correlationId"] = payload.CorrelationId
		trace.Properties["userId"] = payload.UserId
		trace.Properties["appId"] = payload.AppId
		l.aiClient.Track(trace)
	} else {
		l.zapLogger.Warn(payload.Message,
			zap.String("service", payload.Service),
			zap.String("component", payload.Component),
			zap.String("operation", payload.Operation),
			zap.String("correlationId", payload.CorrelationId),
			zap.String("userId", payload.UserId),
			zap.String("appId", payload.AppId),
			zap.Time("timestamp", payload.Timestamp),
		)
	}
}

// Error logs an error-level message with a structured payload.
//
// The method enforces a consistent logging schema by requiring a LogPayload,
// which includes fields such as Service, Component, Operation, CorrelationId,
// UserId, AppId, and Message. A timestamp is automatically added.
//
// Behavior:
//   - If the logger is configured for Azure (useAzure == true), the payload is
//     converted into a TraceTelemetry object and sent to Application Insights,
//     with all fields attached as custom properties. If the payload includes
//     an error, its string value is added to the telemetry properties.
//   - Otherwise, the payload is written to the local Zap logger, which outputs
//     structured JSON logs (typically to file with rotation).
//
// This ensures that both local logs and cloud logs share the same schema,
// making them easy to query and correlate across environments.
//
// Example usage:
//
//	logPayload := logger.LogPayload{
//	    Service:       "r2-notify",
//	    Component:     "notification-service",
//	    Operation:     "CreateNotification",
//	    Message:       "Failed to insert notification",
//	    CorrelationId: "abc-xyz-123",
//	    UserId:        "user-42",
//	    AppId:         "my-app",
//	    Error:         err,
//	}
//	log.Error(logPayload)
func (l *Logger) Error(payload LogPayload) {
	if !l.shouldLog(zap.ErrorLevel) {
		return
	}
	payload.Timestamp = time.Now()
	if l.useAzure {
		trace := ai.NewTraceTelemetry(payload.Message, ai.Error)
		trace.Properties["service"] = payload.Service
		trace.Properties["component"] = payload.Component
		trace.Properties["operation"] = payload.Operation
		trace.Properties["correlationId"] = payload.CorrelationId
		trace.Properties["userId"] = payload.UserId
		trace.Properties["appId"] = payload.AppId
		if payload.Error != nil {
			trace.Properties["error"] = payload.Error.Error()
		}
		l.aiClient.Track(trace)
	} else {
		fields := []zap.Field{
			zap.String("service", payload.Service),
			zap.String("component", payload.Component),
			zap.String("operation", payload.Operation),
			zap.String("correlationId", payload.CorrelationId),
			zap.String("userId", payload.UserId),
			zap.String("appId", payload.AppId),
			zap.Time("timestamp", payload.Timestamp),
		}
		if payload.Error != nil {
			fields = append(fields, zap.Error(payload.Error))
		}
		l.zapLogger.Error(payload.Message, fields...)
	}
}

// Get log level from config
func getLogLevel() zapcore.Level {
	switch config.LoadConfig().LogLevel {
	case data.DEBUG:
		return zapcore.DebugLevel
	case data.INFO:
		return zapcore.InfoLevel
	case data.WARN:
		return zapcore.WarnLevel
	case data.ERROR:
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

// Should log checks if the given log level meets the minimum level set in the logger.
func (l *Logger) shouldLog(level zapcore.Level) bool {
	return level >= l.minLevel
}

// Flush ensures logs are written before shutdown
func (l *Logger) Flush() {
	if l.useAzure {
		l.aiClient.Channel().Flush()
	} else {
		_ = l.zapLogger.Sync()
	}
}
