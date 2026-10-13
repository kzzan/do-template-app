// Package log 基于 zerolog 的日志封装：按配置输出到 console / file / both。
package log

import (
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog"
	"github.com/spf13/viper"
)

type Logger struct {
	zerolog.Logger
}

// NewLogger 依据配置构造 logger。
//
// 配置项：log.level（debug|info|warn|error）、log.mode（console|file|both）、log.file。
func NewLogger(conf *viper.Viper) *Logger {
	zerolog.TimeFieldFormat = time.RFC3339

	level, err := zerolog.ParseLevel(conf.GetString("log.level"))
	if err != nil || level == zerolog.NoLevel {
		level = zerolog.InfoLevel
	}

	var writers []io.Writer
	mode := conf.GetString("log.mode")
	if mode == "" || mode == "console" || mode == "both" {
		writers = append(writers, zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})
	}
	if mode == "file" || mode == "both" {
		file := conf.GetString("log.file")
		if file == "" {
			file = "storage/logs/server.log"
		}
		if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
			panic(err)
		}
		f, err := os.OpenFile(file, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			panic(err)
		}
		writers = append(writers, f)
		// ponytail: 无轮转，日志文件会无限增长；需要时引入 lumberjack
	}

	return &Logger{zerolog.New(zerolog.MultiLevelWriter(writers...)).Level(level).With().Timestamp().Logger()}
}
