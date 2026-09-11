package logging

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/realheyu/magpie/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	gormlogger "gorm.io/gorm/logger"
)

func New(cfg config.LogConfig) (*zap.Logger, error) {
	level, err := parseLevel(cfg.Level)
	if err != nil {
		return nil, err
	}

	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderCfg.EncodeLevel = zapcore.LowercaseLevelEncoder

	cores := make([]zapcore.Core, 0, 2)
	if cfg.Console {
		cores = append(cores, zapcore.NewCore(zapcore.NewConsoleEncoder(encoderCfg), zapcore.Lock(os.Stdout), level))
	}
	if cfg.File.Enabled {
		writer, err := fileWriter(cfg.File)
		if err != nil {
			return nil, err
		}
		cores = append(cores, zapcore.NewCore(zapcore.NewJSONEncoder(encoderCfg), writer, level))
	}
	if len(cores) == 0 {
		return nil, errors.New("日志至少需要启用 console 或 file 一个输出")
	}

	return zap.New(zapcore.NewTee(cores...), zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel)), nil
}

func NewGormLogger(logger *zap.Logger, level string) gormlogger.Interface {
	gormLevel := gormlogger.Warn
	if strings.EqualFold(level, "debug") {
		gormLevel = gormlogger.Info
	}
	if strings.EqualFold(level, "error") {
		gormLevel = gormlogger.Error
	}
	return &GormLogger{
		logger:        logger.Named("gorm"),
		level:         gormLevel,
		slowThreshold: 200 * time.Millisecond,
	}
}

func parseLevel(value string) (zapcore.LevelEnabler, error) {
	var level zapcore.Level
	if value == "" {
		value = "info"
	}
	if err := level.Set(strings.ToLower(value)); err != nil {
		return nil, fmt.Errorf("日志等级无效：%s", value)
	}
	return level, nil
}

func fileWriter(cfg config.LogFileConfig) (zapcore.WriteSyncer, error) {
	filename := cfg.Filename
	if filename == "" {
		filename = "logs/magpie.log"
	}
	if dir := filepath.Dir(filename); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}
	if cfg.MaxSizeMB <= 0 {
		cfg.MaxSizeMB = 100
	}
	if cfg.MaxBackups < 0 {
		cfg.MaxBackups = 0
	}
	if cfg.MaxAgeDays < 0 {
		cfg.MaxAgeDays = 0
	}
	return zapcore.AddSync(&lumberjack.Logger{
		Filename:   filename,
		MaxSize:    cfg.MaxSizeMB,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAgeDays,
		Compress:   cfg.Compress,
	}), nil
}

type GormLogger struct {
	logger        *zap.Logger
	level         gormlogger.LogLevel
	slowThreshold time.Duration
}

func (l *GormLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	clone := *l
	clone.level = level
	return &clone
}

func (l *GormLogger) Info(ctx context.Context, msg string, args ...any) {
	if l.level >= gormlogger.Info {
		l.logger.Info(fmt.Sprintf(msg, args...))
	}
}

func (l *GormLogger) Warn(ctx context.Context, msg string, args ...any) {
	if l.level >= gormlogger.Warn {
		l.logger.Warn(fmt.Sprintf(msg, args...))
	}
}

func (l *GormLogger) Error(ctx context.Context, msg string, args ...any) {
	if l.level >= gormlogger.Error {
		l.logger.Error(fmt.Sprintf(msg, args...))
	}
}

func (l *GormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.level <= gormlogger.Silent {
		return
	}
	elapsed := time.Since(begin)
	sql, rows := fc()
	fields := []zap.Field{
		zap.Duration("elapsed", elapsed),
		zap.Int64("rows", rows),
		zap.String("sql", sql),
	}
	if err != nil && l.level >= gormlogger.Error {
		l.logger.Error("sql error", append(fields, zap.Error(err))...)
		return
	}
	if l.slowThreshold > 0 && elapsed > l.slowThreshold && l.level >= gormlogger.Warn {
		l.logger.Warn("slow sql", fields...)
		return
	}
	if l.level >= gormlogger.Info {
		l.logger.Debug("sql", fields...)
	}
}
