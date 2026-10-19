package initialize

import (
	"bokee/config"
	"bokee/global"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

func initLogger(cfg *config.Config) {
	var err error
	global.Log, err = NewLogger(cfg.Log)
	if err != nil {
		panic(fmt.Sprintf("日志初始化失败: %v", err))
	}
}

type levelLogger struct {
	level zapcore.Level
	name  string
}

// dateDirWriter 按 年/月/日 目录隔离，级别文件放在当天目录下
type dateDirWriter struct {
	mu         sync.Mutex
	logConfig  config.LogConfig
	levelName  string
	baseDir    string
	currentDay string // "2006-01-02"
	lj         *lumberjack.Logger
}

func newDateDirWriter(logConfig config.LogConfig, levelName, baseDir string) *dateDirWriter {
	w := &dateDirWriter{
		logConfig: logConfig,
		levelName: levelName,
		baseDir:   baseDir,
	}
	w.rotateIfNeeded()
	return w
}

func (w *dateDirWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.rotateIfNeeded()
	return w.lj.Write(p)
}

func (w *dateDirWriter) Sync() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.lj != nil {
		return w.lj.Close()
	}
	return nil
}

func (w *dateDirWriter) rotateIfNeeded() {
	now := time.Now()
	today := now.Format("2006-01-02")
	if w.currentDay == today && w.lj != nil {
		return
	}

	// 关闭旧文件
	if w.lj != nil {
		_ = w.lj.Close()
	}

	// 构建 年/月/日 目录：baseDir/2026/09/11
	year := now.Format("2006")
	month := now.Format("01")
	day := now.Format("02")
	dir := filepath.Join(w.baseDir, year, month, day)

	if err := os.MkdirAll(dir, 0755); err != nil {
		dir = w.baseDir
	}

	filename := filepath.Join(dir, w.levelName+".log")

	w.lj = &lumberjack.Logger{
		Filename:   filename,
		MaxSize:    w.logConfig.MaxSize,
		MaxBackups: w.logConfig.MaxBackups,
		MaxAge:     w.logConfig.MaxAge,
		Compress:   w.logConfig.Compress,
		LocalTime:  true,
	}
	w.currentDay = today
}

func NewLogger(logConfig config.LogConfig) (*zap.Logger, error) {
	globalLevel := parseLogLevel(logConfig.Level)
	logDir := logConfig.FilePath

	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("创建日志目录失败: %w", err)
	}

	encoder := newEncoder(logConfig.Format)

	var cores []zapcore.Core

	switch strings.ToLower(logConfig.Output) {
	case "stdout":
		cores = append(cores, newStdoutCore(encoder, globalLevel))
	case "file":
		fileCores := newFileCores(logConfig, encoder, globalLevel, logDir)
		cores = append(cores, fileCores...)
	case "both":
		cores = append(cores, newStdoutCore(encoder, globalLevel))
		fileCores := newFileCores(logConfig, encoder, globalLevel, logDir)
		cores = append(cores, fileCores...)
	default:
		return nil, fmt.Errorf("不支持的输出类型: %s (可选: stdout/file/both)", logConfig.Output)
	}

	if len(cores) == 0 {
		return nil, fmt.Errorf("未能创建任何日志核心(Core)")
	}

	core := zapcore.NewTee(cores...)
	logger := zap.New(core, zap.AddCaller())

	return logger, nil
}

func parseLogLevel(level string) zapcore.Level {
	switch strings.ToLower(level) {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "fatal":
		return zapcore.FatalLevel
	case "panic":
		return zapcore.PanicLevel
	default:
		return zapcore.InfoLevel
	}
}

func newEncoder(format string) zapcore.Encoder {
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
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

	if strings.ToLower(format) == "json" {
		return zapcore.NewJSONEncoder(encoderConfig)
	}
	return zapcore.NewConsoleEncoder(encoderConfig)
}

func newStdoutCore(encoder zapcore.Encoder, level zapcore.Level) zapcore.Core {
	return zapcore.NewCore(
		encoder,
		zapcore.AddSync(os.Stdout),
		level,
	)
}

// newFileCores 按级别拆分 + 年/月/日 目录隔离
func newFileCores(logConfig config.LogConfig, encoder zapcore.Encoder, globalLevel zapcore.Level, logDir string) []zapcore.Core {
	levelLoggers := []levelLogger{
		{zapcore.DebugLevel, "debug"},
		{zapcore.InfoLevel, "info"},
		{zapcore.WarnLevel, "warn"},
		{zapcore.ErrorLevel, "error"},
	}

	var cores []zapcore.Core
	for _, ll := range levelLoggers {
		if ll.level < globalLevel {
			continue
		}

		writer := newDateDirWriter(logConfig, ll.name, logDir)

		core := zapcore.NewCore(
			encoder,
			zapcore.AddSync(writer),
			ll.level,
		)
		cores = append(cores, core)
	}

	return cores
}
