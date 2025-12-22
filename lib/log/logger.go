package log

import (
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

func NewStreamSugaredLogger(nodeID string) *zap.SugaredLogger {
	return nil
}

// NewZapSugaredLogger 创建一个配置灵活的 zap.SugaredLogger
// 参数:
//   - logPath: 日志输出路径数组，如 ["app.log", "stdout"]，支持多个输出目标
//   - logFormat: 日志格式，"plant" 或 "json"
func NewZapSugaredLogger(
	logPath []string,
	logFormat string,
) *zap.SugaredLogger {

	if logPath == nil || len(logPath) == 0 {
		logPath = []string{"stdout"}
	}

	// 1. 配置日志级别
	atomicLevel := zap.NewAtomicLevel()
	atomicLevel.SetLevel(zap.InfoLevel)

	// 2. 创建编码器配置
	encoderConfig := zap.NewProductionEncoderConfig()

	// 3. 创建多个输出核心
	var cores []zapcore.Core
	var encoder zapcore.Encoder

	for _, path := range logPath {
		var writeSyncer zapcore.WriteSyncer

		switch path {
		case "stdout":
			writeSyncer = zapcore.AddSync(os.Stdout)
			encoder = zapcore.NewConsoleEncoder(encoderConfig)
		case "stderr":
			writeSyncer = zapcore.AddSync(os.Stderr)
			encoder = zapcore.NewConsoleEncoder(encoderConfig)
		default:
			// 文件输出
			// 使用 lumberjack 进行日志轮转
			writeSyncer = zapcore.AddSync(&lumberjack.Logger{
				Filename:  path,
				MaxSize:   100,  // MB
				MaxAge:    365,  // 保留天数
				Compress:  true, // 压缩旧日志
				LocalTime: true, // 使用本地时间
			})

			encoder = getEncoder(encoderConfig, logFormat)
		}

		// 创建 core
		core := zapcore.NewCore(
			encoder,
			writeSyncer,
			atomicLevel,
		)
		cores = append(cores, core)
	}

	// 4. 合并所有 cores
	teeCore := zapcore.NewTee(cores...)

	// 5. 创建 Logger
	logger := zap.New(teeCore,
		zap.AddCaller(),
		zap.AddCallerSkip(1),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)

	// 6. 替换全局 logger
	zap.ReplaceGlobals(logger)

	return logger.Sugar()
}

func getEncoder(encoderConfig zapcore.EncoderConfig, logFormat string) zapcore.Encoder {

	var encoder zapcore.Encoder

	switch strings.ToLower(logFormat) {
	case "plant", "text", "console":
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	case "json":
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	default:
		// 默认使用 JSON 格式
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	return encoder
}

// ensureDir 确保日志文件所在目录存在
func ensureDir(filePath string) {
	dir := filePath
	// 如果是文件路径，提取目录部分
	if strings.Contains(filePath, "/") {
		lastSlash := strings.LastIndex(filePath, "/")
		if lastSlash > 0 {
			dir = filePath[:lastSlash]
		}
	}

	if dir != "" && dir != "." && dir != ".." {
		_ = os.MkdirAll(dir, 0755)
	}
}

// 可选：提供一些便捷的预设配置
type LogConfig struct {
	// 输出路径，支持多个
	Paths []string
	// 格式：json 或 plant
	Format string
	// 日志级别
	Level string
	// 是否启用彩色输出（仅 plant 格式有效）
	EnableColor bool
	// 是否输出调用者信息
	EnableCaller bool
	// 是否输出堆栈信息
	EnableStackTrace bool
}

// NewLoggerWithConfig 使用配置结构体创建 Logger
func NewLoggerWithConfig(config LogConfig) *zap.SugaredLogger {
	// 如果未指定路径，使用默认
	if len(config.Paths) == 0 {
		config.Paths = []string{"stdout"}
	}

	// 如果未指定格式，使用默认
	if config.Format == "" {
		config.Format = "json"
	}

	// 创建基础 Logger
	logger := NewZapSugaredLogger(
		config.Paths,
		config.Format,
	)

	// 如果不是 plant 格式，直接返回
	if strings.ToLower(config.Format) != "plant" {
		return logger
	}

	// 对于 plant 格式，可以重新配置颜色等选项
	if config.EnableColor {
		// 重新构建一个带颜色的控制台编码器
		/*
			encoderConfig := zapcore.EncoderConfig{
				TimeKey:        "T",
				LevelKey:       "L",
				NameKey:        "N",
				CallerKey:      "C",
				FunctionKey:    zapcore.OmitKey,
				MessageKey:     "M",
				StacktraceKey:  "S",
				LineEnding:     zapcore.DefaultLineEnding,
				EncodeLevel:    zapcore.CapitalColorLevelEncoder, // 带颜色
				EncodeTime:     zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05.000"),
				EncodeDuration: zapcore.StringDurationEncoder,
				EncodeCaller:   zapcore.ShortCallerEncoder,
			}
		*/

		// 重新创建 cores 并替换
		desugar := logger.Desugar()
		if core := desugar.Core(); core != nil {
			// 这里简化处理，实际可能需要更复杂的替换逻辑
			// 对于简单使用，建议在 NewZapSugaredLogger 中通过参数控制颜色
		}
	}

	return logger
}

// 提供一些预设的快捷函数
func NewDevelopmentLogger() *zap.SugaredLogger {
	return NewZapSugaredLogger(
		[]string{"stdout"},
		"plant",
	)
}

func NewProductionLogger(logFile string) *zap.SugaredLogger {
	return NewZapSugaredLogger(
		[]string{logFile, "stdout"},
		"json",
	)
}

func NewTestLogger() *zap.SugaredLogger {
	return NewZapSugaredLogger(
		[]string{"test.log"},
		"json",
	)
}
