package util

import (
	"gogame/base/logger"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"
)

//获取程序名
func GetAppName() string {
	return strings.TrimSuffix(filepath.Base(os.Args[0]), filepath.Ext(os.Args[0]))
}

//获取程序路径
func GetAppPath() string {
	fp, err := filepath.Abs(os.Args[0])
	if err != nil {
		return "."
	}
	return filepath.Dir(fp)

}

//判断文件或文件是否存在
func Exists(path string) bool {
	_, err := os.Stat(path) //os.Stat获取文件信息
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}

func PrintPanicStack() {
	if err := recover(); err != nil {
		buf := debug.Stack()
		logger.Error("panic: %v\n%s", err, buf)
	}
}
func TimeInterval(duration time.Duration, f func()) {
	tk := time.NewTicker(duration)
	go func() {
		for {
			<-tk.C
			f()
		}
	}()
}
func TimeIntervalCount(duration time.Duration, f func(), count uint32) {
	tk := time.NewTicker(duration)
	defer tk.Stop()
	go func() {
		for {
			<-tk.C
			f()
			count--
			if count <= 0 {
				return
			}
		}
	}()
}
