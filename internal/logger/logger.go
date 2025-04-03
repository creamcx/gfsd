package logger

import (
	"astro-sarafan/internal/config"
	"os"

	"github.com/fatih/color"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const permissions = 0o644

func New(cfg config.Logger) (*zap.Logger, error) {
	var level zapcore.Level
	switch cfg.Level {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	default:
		level = zapcore.InfoLevel
	}

	var output zapcore.WriteSyncer
	switch cfg.Sink {
	case "stdout":
		output = os.Stdout
	default:
		file, err := os.OpenFile(cfg.Sink, os.O_WRONLY|os.O_CREATE|os.O_APPEND, permissions)
		if err != nil {
			return nil, err
		}
		output = zapcore.AddSync(file)
	}

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:       "time",
		LevelKey:      "level",
		NameKey:       "logger",
		CallerKey:     "caller",
		MessageKey:    "msg",
		StacktraceKey: "stacktrace",
		LineEnding:    zapcore.DefaultLineEnding,
		EncodeLevel:   colorLevelEncoder(),
		EncodeTime:    zapcore.TimeEncoderOfLayout("[2006-01-02 15:04:05]"),
		EncodeCaller:  zapcore.ShortCallerEncoder,
		EncodeName:    zapcore.FullNameEncoder,
	}

	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderConfig),
		output,
		level,
	)

	logger := zap.New(core, zap.AddCaller())
	return logger, nil
}

func colorLevelEncoder() zapcore.LevelEncoder {
	return func(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
		switch l {
		case zapcore.DebugLevel:
			enc.AppendString(color.MagentaString("DEBUG:"))
		case zapcore.InfoLevel:
			enc.AppendString(color.BlueString("INFO:"))
		case zapcore.WarnLevel:
			enc.AppendString(color.YellowString("WARN:"))
		case zapcore.ErrorLevel:
			enc.AppendString(color.RedString("ERROR:"))
		default:
			enc.AppendString(l.String() + ":")
		}
	}
}
