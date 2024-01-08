package logger

import (
	"context"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	gormlogger "gorm.io/gorm/logger"
	"log"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type ZapLogger struct {
	Logger *zap.Logger
	config *ZapConfig
}

type ZapConfig struct {
	DebugFile  string
	InfoFile   string
	ErrorFile  string
	MaxSize    int
	MaxBackups int
	MaxAge     int
	LocalTime  bool
	Compress   bool
}

var zapLoggerInstance *ZapLogger

func NewZapLogger(conf *ZapConfig) *ZapLogger {
	zapLoggerInstance = &ZapLogger{
		config: conf,
	}
	core := zapcore.NewTee(zapLoggerInstance.getDebugCore(), zapLoggerInstance.getInfoCore(), zapLoggerInstance.getErrorCore())
	logger := zap.New(core, zap.AddCaller())
	defer logger.Sync()
	zapLoggerInstance.Logger = logger
	return zapLoggerInstance
}

func (l *ZapLogger) getDebugCore() zapcore.Core {
	encoder := l.getJSONEncoder()
	writerSyncer := l.getWriteSyncer("DEBUG")
	logLevel := zap.LevelEnablerFunc(func(level zapcore.Level) bool {
		return level == zap.DebugLevel
	})
	return zapcore.NewCore(encoder, writerSyncer, logLevel)
}

func (l *ZapLogger) getInfoCore() zapcore.Core {
	encoder := l.getJSONEncoder()
	writerSyncer := l.getWriteSyncer("INFO")
	logLevel := zap.LevelEnablerFunc(func(level zapcore.Level) bool {
		return level == zap.InfoLevel
	})
	return zapcore.NewCore(encoder, writerSyncer, logLevel)
}

func (l *ZapLogger) getErrorCore() zapcore.Core {
	encoder := l.getJSONEncoder()
	writerSyncer := l.getWriteSyncer("ERROR")
	logLevel := zap.LevelEnablerFunc(func(level zapcore.Level) bool {
		return level >= zap.WarnLevel
	})
	return zapcore.NewCore(encoder, writerSyncer, logLevel)
}

func (l *ZapLogger) getWriteSyncer(logLevel string) zapcore.WriteSyncer {
	var fileName string
	switch logLevel {
	case "INFO":
		fileName = l.config.InfoFile
	case "DEBUG":
		fileName = l.config.DebugFile
	case "ERROR":
		fileName = l.config.ErrorFile
	default:
		log.Fatalln("Zap logger not config the level " + logLevel)
	}
	hook := lumberjack.Logger{
		Filename:   fileName,
		MaxSize:    l.config.MaxSize,
		MaxBackups: l.config.MaxBackups,
		MaxAge:     l.config.MaxAge,
		LocalTime:  l.config.LocalTime,
		Compress:   l.config.Compress,
	}
	return zapcore.AddSync(&hook)
}

func (l *ZapLogger) getJSONEncoder() zapcore.Encoder {
	return zapcore.NewJSONEncoder(zapcore.EncoderConfig{
		MessageKey:     "msg",
		LevelKey:       "level",
		TimeKey:        "time",
		NameKey:        "name",
		CallerKey:      "caller",
		FunctionKey:    "func",
		StacktraceKey:  "stack",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05.000"),
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	})
}

type ZapGormLogger struct {
	Logger *ZapLogger
	config *ZapGormConfig
}

type ZapGormConfig struct {
	SlowThreshold time.Duration
	LogLevel      gormlogger.LogLevel
}

func NewZapGorm(zapLogger *ZapLogger, conf *ZapGormConfig) *ZapGormLogger {
	if conf.SlowThreshold == 0 {
		conf.SlowThreshold = time.Second
	}
	logger := &ZapGormLogger{
		Logger: zapLogger,
		config: conf,
	}
	for i := 2; i < 15; i++ {
		_, file, _, ok := runtime.Caller(i)
		switch {
		case !ok:
		case strings.HasSuffix(file, "_test.go"):
		case strings.Contains(file, filepath.Join("gorm.io", "gorm")):
		default:
			logger.Logger.Logger.WithOptions(zap.AddCallerSkip(i))
		}
	}
	return logger
}

func (zgl *ZapGormLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	zgl.config.LogLevel = level
	return zgl
}

func (zgl *ZapGormLogger) Info(ctx context.Context, str string, args ...interface{}) {
	if zgl.config.LogLevel >= gormlogger.Info {
		zgl.Logger.Logger.Sugar().Debugf(str, args...)
	}
}

func (zgl *ZapGormLogger) Warn(ctx context.Context, str string, args ...interface{}) {
	if zgl.config.LogLevel >= gormlogger.Warn {
		zgl.Logger.Logger.Sugar().Warnf(str, args...)
	}
}

func (zgl *ZapGormLogger) Error(ctx context.Context, str string, args ...interface{}) {
	if zgl.config.LogLevel >= gormlogger.Error {
		zgl.Logger.Logger.Sugar().Errorf(str, args...)
	}
}

func (zgl *ZapGormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if zgl.config.LogLevel > gormlogger.Silent {
		elapsed := time.Since(begin)
		switch {
		case err != nil && zgl.config.LogLevel >= gormlogger.Error:
			sql, rows := fc()
			zgl.Logger.Logger.Error("trace", zap.Error(err), zap.Duration("elapsed", elapsed), zap.Int64("rows", rows), zap.String("sql", sql))
		case elapsed > zgl.config.SlowThreshold && zgl.config.SlowThreshold != 0 && zgl.config.LogLevel >= gormlogger.Warn:
			sql, rows := fc()
			zgl.Logger.Logger.Warn("trace", zap.Duration("elapsed", elapsed), zap.Int64("rows", rows), zap.String("sql", sql))
		case zgl.config.LogLevel == gormlogger.Info:
			sql, rows := fc()
			zgl.Logger.Logger.Debug("trace", zap.Duration("elapsed", elapsed), zap.Int64("rows", rows), zap.String("sql", sql))
		}
	}
}
