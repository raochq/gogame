package logger

import "time"

type Config struct {
	LogPath      string `goconf:"log:path"`
	LogLevel     int    `goconf:"log:level"`
	MaxSize      ByteSize
	CheckSize    time.Duration
	CheckExpired time.Duration
	Expired      time.Duration
	Interval     time.Duration
}

type Rotate struct {
	Size ByteSize
}

func DefaultConfig() Config {
	return Config{
		LogLevel:     LogLevel_Info,
		CheckSize:    2 * time.Minute,
		CheckExpired: 2 * time.Hour,
	}
}
