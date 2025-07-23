package logging

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	once      sync.Once
	logWriter io.Writer
	logLevel  zerolog.Level
	logFile   string
)

func CleanPath(path string) string {
	return strings.ReplaceAll(path, "\\", "/")
}

func levelColor(level zerolog.Level) string {
	switch level {
	case zerolog.TraceLevel:
		return "\033[36m" // Cyan
	case zerolog.DebugLevel:
		return "\033[34m" // Blue
	case zerolog.InfoLevel:
		return "\033[32m" // Green
	case zerolog.WarnLevel:
		return "\033[33m" // Yellow
	case zerolog.ErrorLevel:
		return "\033[31m" // Red
	case zerolog.FatalLevel:
		return "\033[35m" // Magenta
	case zerolog.PanicLevel:
		return "\033[41m\033[97m" // White on Red background
	default:
		return "\033[0m" // Reset
	}
}

func Init(baseDir string, levelStr string) {
	once.Do(func() {
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs

		level, err := zerolog.ParseLevel(strings.ToLower(levelStr))
		if err != nil {
			level = zerolog.InfoLevel
		}
		logLevel = level

		logDir := filepath.Join(baseDir, "logs")
		_ = os.MkdirAll(logDir, 0755)
		logFile = filepath.Join(logDir, time.Now().Format("2006-01-02_15.04.05")+".log")

		fileWriter := &lumberjack.Logger{
			Filename:   CleanPath(logFile),
			MaxSize:    2, // MB, low for easy sharing
			MaxBackups: 5,
			MaxAge:     14,
			Compress:   false,
		}

		consoleWriter := zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: "15:04:05",
			NoColor:    false,
		}
		consoleWriter.FormatLevel = func(i interface{}) string {
			level, ok := i.(string)
			if !ok {
				return strings.ToUpper(fmt.Sprintf("%s", i))
			}
			lvl, _ := zerolog.ParseLevel(level)
			return levelColor(lvl) + strings.ToUpper(level) + "\033[0m"
		}

		logWriter = io.MultiWriter(consoleWriter, fileWriter)

		zerolog.SetGlobalLevel(logLevel)
		log.Logger = zerolog.New(logWriter).With().Timestamp().Logger()

		log.Info().Str("logfile", CleanPath(logFile)).Str("level", logLevel.String()).Msg("Logger initialized")
	})
}

func SetLevel(levelStr string) {
	level, err := zerolog.ParseLevel(strings.ToLower(levelStr))
	if err != nil {
		log.Warn().Str("input", levelStr).Msg("Invalid log level, keeping previous")
		return
	}
	zerolog.SetGlobalLevel(level)
	logLevel = level
	log.Info().Str("level", logLevel.String()).Msg("Log level changed")
}

func GetLogFile() string {
	return CleanPath(logFile)
}
