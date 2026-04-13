package log

import (
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/RenaLio/tudou/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type ctxLoggerKeyType struct{}

var ctxLoggerKey = ctxLoggerKeyType{}

func GetCtxLoggerKey() ctxLoggerKeyType {
	return ctxLoggerKey
}

type Logger struct {
	*zap.Logger
}

// NewLog 根据配置创建 zap logger，支持 console / file 输出以及 json / console 编码
func NewLog(conf *config.Config) *Logger {
	// 解析日志级别
	var level zapcore.Level
	switch conf.Log.Level {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	default:
		level = zapcore.DebugLevel
	}

	// 通用编码器配置
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.RFC3339TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// 构建 cores
	var cores []zapcore.Core

	// 控制台输出
	if conf.Log.Mode == "console" || conf.Log.Mode == "both" {
		var consoleEncoder zapcore.Encoder
		if conf.Log.ConsoleEncoding == "json" {
			consoleEncoder = zapcore.NewJSONEncoder(encoderConfig)
		} else {
			encoderConfig.EncodeCaller = zapcore.FullCallerEncoder
			consoleEncoder = zapcore.NewConsoleEncoder(encoderConfig)
		}
		consoleCore := zapcore.NewCore(
			consoleEncoder,
			zapcore.AddSync(os.Stdout),
			level,
		)
		cores = append(cores, consoleCore)
	}

	// 文件输出
	if conf.Log.Mode == "file" || conf.Log.Mode == "both" {
		// 确保日志目录存在
		if conf.Log.LogPath != "" {
			_ = os.MkdirAll(conf.Log.LogPath, 0o755)
		}

		// 普通日志文件 writer
		normalWriter := getLoggerWriter(&fileOutputOption{
			Filename:   filepath.Join(conf.Log.LogPath, conf.Log.FileName),
			MaxSize:    conf.Log.MaxSize,
			MaxBackups: conf.Log.MaxBackups,
			MaxAge:     conf.Log.MaxAge,
			Compress:   conf.Log.Compress,
		})
		// 错误日志文件 writer（仅记录 error 及以上级别）
		errorWriter := getLoggerWriter(&fileOutputOption{
			Filename:   filepath.Join(conf.Log.LogPath, conf.Log.ErrorFileName),
			MaxSize:    conf.Log.MaxSize,
			MaxBackups: conf.Log.MaxBackups,
			MaxAge:     conf.Log.MaxAge,
			Compress:   conf.Log.Compress,
		})

		var fileEncoder zapcore.Encoder
		if conf.Log.FileEncoding == "json" {
			fileEncoder = zapcore.NewJSONEncoder(encoderConfig)
		} else {
			fileEncoder = zapcore.NewConsoleEncoder(encoderConfig)
		}

		// 普通文件 core：记录 >= level 的日志
		normalCore := zapcore.NewCore(
			fileEncoder,
			zapcore.AddSync(normalWriter),
			level,
		)
		cores = append(cores, normalCore)

		// 错误文件 core：仅记录 >= ErrorLevel 的日志
		errorCore := zapcore.NewCore(
			fileEncoder,
			zapcore.AddSync(errorWriter),
			zapcore.ErrorLevel, // 只记录 error 及以上
		)
		cores = append(cores, errorCore)
	}

	// 合并多个 core
	core := zapcore.NewTee(cores...)

	// 构建 logger，添加 caller 信息
	if conf.Env != "prod" {
		return &Logger{zap.New(core, zap.Development(), zap.AddCaller(), zap.AddStacktrace(zap.ErrorLevel))}
	}

	return &Logger{zap.New(core, zap.AddCaller(), zap.AddStacktrace(zap.ErrorLevel))}
}

func getLoggerWriter(option *fileOutputOption) io.Writer {
	return &lumberjack.Logger{
		Filename:   option.Filename,
		MaxSize:    option.MaxSize,
		MaxBackups: option.MaxBackups,
		MaxAge:     option.MaxAge,
		Compress:   option.Compress,
	}
}

// Inject 向 context 注入带字段的 logger
func (l *Logger) Inject(ctx context.Context, fields ...zap.Field) context.Context {
	return context.WithValue(ctx, ctxLoggerKey, l.FromContext(ctx).With(fields...))
}

// FromContext 从 context 中提取 logger，若不存在则返回当前 logger
func (l *Logger) FromContext(ctx context.Context) *Logger {
	if logger, ok := ctx.Value(ctxLoggerKey).(*zap.Logger); ok {
		return &Logger{logger}
	}
	return l
}

type fileOutputOption struct {
	Filename   string
	MaxSize    int
	MaxBackups int
	MaxAge     int
	Compress   bool
}
