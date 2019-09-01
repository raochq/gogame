package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

type ByteSize uint64

const (
	_           = iota
	KB ByteSize = 1 << (10 * iota)
	MB
	GB
	TB
	PB
	EB
)

const (
	_ = iota
	LogLevel_Fatal
	LogLevel_Error
	LogLevel_Warn
	LogLevel_Info
	LogLevel_Debug
)

var (
	prefix = [6]string{
		"",
		"\033[0;33mFATAL:\033[0m ",
		"\033[0;31mERROR:\033[0m ",
		"\033[0;35mWARN:\033[0m ",
		"\033[0;32mINFO:\033[0m ",
		"\033[0;36mDEBUG:\033[0m "}
)

var (
	gLogger *logger
)

type logger struct {
	sync.RWMutex
	errCount int32
	Config

	file                          *os.File
	debug, info, warn, err, fatal *log.Logger
}

func (l *logger) setOutPut(out io.Writer) {
	l.debug.SetOutput(out)
	l.info.SetOutput(out)
	l.warn.SetOutput(out)
	l.err.SetOutput(out)
	l.fatal.SetOutput(out)
}

func (l *logger) getFileSize() ByteSize {
	fi, err := l.file.Stat()
	if err != nil {
		Warn("get log file size failed, no trunc %s", err.Error())
		return 0.0
	}
	return ByteSize(fi.Size())
}

func (l *logger) trunc(fp, nfp string) {
	l.Lock()
	defer l.Unlock()
	if fp == nfp {
		return
	}
	if l.file != nil {
		ofp := l.file.Name()
		err := l.file.Close()
		if err != nil {
			log.Println("fail to close log file", err.Error())
			return
		}

		if nfp != "" && ofp != nfp {
			err = os.Rename(ofp, nfp)
			if err != nil {
				log.Println("fail to rename log file, no trunc", err.Error())
			}
		}
	}
	f, err := os.OpenFile(fp, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		atomic.AddInt32(&l.errCount, 1)
		l.err.Output(2, fmt.Sprintln(fmt.Sprintf("create log file failed %s", err.Error())))
		return
	}
	l.setOutPut(f)
	l.file = f
}

func (l *logger) doPrintln(level int, format string, v ...interface{}) {
	if l.LogLevel < level {
		return
	}
	l.RLock()
	switch level {
	case LogLevel_Debug:
		l.debug.Output(4, fmt.Sprintln(fmt.Sprintf(format, v...)))
	case LogLevel_Info:
		l.info.Output(4, fmt.Sprintln(fmt.Sprintf(format, v...)))
	case LogLevel_Warn:
		l.warn.Output(4, fmt.Sprintln(fmt.Sprintf(format, v...)))
	case LogLevel_Error:
		l.err.Output(4, fmt.Sprintln(fmt.Sprintf(format, v...)))
	case LogLevel_Fatal:
		l.fatal.Output(4, fmt.Sprintln(fmt.Sprintf(format, v...)))
	default:
		l.info.Output(4, fmt.Sprintln(fmt.Sprintf(format, v...)))
	}
	l.RUnlock()
}
func (l *logger) reload(conf Config) {
	l.Lock()
	l.Config = conf
	l.Unlock()
	l.setLogPath()
	Info("ReloadConfig ok")
}

func (l *logger) setLogPath() {
	_, err := os.Stat(l.LogPath)
	if err != nil && os.IsNotExist(err) {
		err = os.Mkdir(l.LogPath, 0755)
		if err != nil {
			Logger().Fatal(err.Error())
		}
	}
	f := l.LogPath + "/" + strings.TrimSuffix(filepath.Base(os.Args[0]), filepath.Ext(os.Args[0])) + ".log"
	oldFileName := ""
	if l.file != nil {
		l.RLock()
		oldFileName = l.file.Name()
		l.RUnlock()
	}
	if oldFileName != f {
		l.trunc(f, oldFileName)
	}
}

func (l *logger) Fatal(format string, v ...interface{}) {
	atomic.AddInt32(&l.errCount, 1)
	l.doPrintln(LogLevel_Fatal, format, v...)
	os.Exit(1)
}

func (l *logger) Error(format string, v ...interface{}) {
	atomic.AddInt32(&l.errCount, 1)
	l.doPrintln(LogLevel_Error, format, v...)
}

func (l *logger) Warn(format string, v ...interface{}) {
	l.doPrintln(LogLevel_Warn, format, v...)
}

func (l *logger) Info(format string, v ...interface{}) {
	l.doPrintln(LogLevel_Info, format, v...)
}

func (l *logger) Debug(format string, v ...interface{}) {
	l.doPrintln(LogLevel_Debug, format, v...)
}

func suffix(t time.Time) string {
	return t.Format("-2006010215")
}

func toNextBound(d time.Duration) time.Duration {
	return time.Now().Truncate(d).Add(d).Sub(time.Now())
}

func (l *logger) loop() error {
	interval := time.After(toNextBound(l.Interval))
	expired := time.After(l.CheckExpired)
	sizeExt := 1
	for {
		var size <-chan time.Time
		if toNextBound(l.Interval) != l.CheckSize {
			size = time.After(l.CheckSize)
		}
		select {
		case t := <-interval:
			interval = time.After(l.Interval)
			if l.file != nil {
				fp := l.file.Name()
				l.trunc(fp, fp+suffix(t))
				sizeExt = 1
				l.Info("log truncated by time interval")
			}
		case <-expired:
			expired = time.After(l.CheckExpired)
			if l.file != nil {
				fp := l.file.Name()
				err := filepath.Walk(filepath.Dir(fp),
					func(path string, info os.FileInfo, err error) error {
						if err != nil {
							return nil
						}
						isLog := strings.Contains(info.Name(), ".log")

						//log.Println("strings.Contains(", info.Name(), " log') isLog = ", isLog)
						if time.Since(info.ModTime()) > l.Expired && isLog && info.IsDir() == false {
							if err := os.Remove(path); err != nil {
								return err
							}
							l.Info("remove expired log files %s", filepath.Base(path))
						}
						return nil
					})
				if err != nil {
					l.Warn("remove expired logger failed %s", err.Error())
				}
			}
		case t := <-size:
			if l.file != nil {
				if l.getFileSize() < l.MaxSize {
					continue
				}
				fp := l.file.Name()
				l.trunc(fp, fp+suffix(t)+"."+strconv.Itoa(sizeExt))
				sizeExt++
				l.Info("log over size, truncated")
			}
		}
	}
}

// Debug log debug protocol with cyan color.
func Debug(format string, v ...interface{}) {
	Logger().Debug(format, v...)
}

// Info log normal protocol.
func Info(format string, v ...interface{}) {
	Logger().Info(format, v...)
}

// Warn log error protocol
func Warn(format string, v ...interface{}) {
	Logger().Warn(format, v...)
}

// Error log error protocol with red color.
func Error(format string, v ...interface{}) {
	Logger().Error(format, v...)
}

// Fatal log error protocol
func Fatal(format string, v ...interface{}) {
	Logger().Fatal(format, v...)
}

// Fatal log error protocol
func ErrCount() int32 {
	ec := atomic.LoadInt32(&gLogger.errCount)
	if ec < 0 {
		Warn("error count overflow")
		return -1
	}
	return ec
}

func newLogger(conf Config) *logger {
	l := new(logger)
	l.debug = log.New(os.Stdout, prefix[LogLevel_Debug], log.LstdFlags|log.Lshortfile)
	l.info = log.New(os.Stdout, prefix[LogLevel_Info], log.LstdFlags|log.Lshortfile)
	l.warn = log.New(os.Stderr, prefix[LogLevel_Warn], log.LstdFlags|log.Lshortfile)
	l.err = log.New(os.Stderr, prefix[LogLevel_Error], log.LstdFlags|log.Lshortfile)
	l.fatal = log.New(os.Stderr, prefix[LogLevel_Fatal], log.LstdFlags|log.Lshortfile)
	l.Config = conf
	return l
}

func Logger() *logger {
	if gLogger == nil {
		l := newLogger(DefaultConfig())
		if atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(&gLogger)), nil, unsafe.Pointer(l)) {
			go l.loop()
		}
	}
	return gLogger
}

func ReloadConfig(conf Config) {
	if gLogger == nil {
		l := newLogger(conf)
		if atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(&gLogger)), nil, unsafe.Pointer(l)) {
			go l.loop()
			gLogger = l
		} else {
			if gLogger != nil {
				gLogger.reload(conf)
			} else {
				log.Println("ReloadConfig failed")
			}
		}
	} else {
		gLogger.reload(conf)
	}

}
