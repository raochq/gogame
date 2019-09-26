package service

import (
	"github.com/google/gops/agent"
	"gogame/base/logger"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

const (
	SIG_STOP   = 0
	SIG_RELOAD = 1
	SIG_REPORT = 2
)

var services []*Service

type IService interface {
	Init()
	Destroy()
	MainLoop(sig <-chan byte)
}

//服务基类
type Service struct {
	impl IService
	sig  chan byte
	wg   sync.WaitGroup
}

func Register(s IService) {
	if s == nil {
		logger.Fatal("Register a nil IService")
	}
	svr := new(Service)
	svr.sig = make(chan byte)
	svr.impl = s
	services = append(services, svr)
}

func onInit() {
	for _, s := range services {
		s.impl.Init()
	}
	for _, s := range services {
		s.wg.Add(1)
		go func() {
			s.impl.MainLoop(s.sig)
			s.wg.Done()
		}()
	}
}

func stop() {
	for i := len(services) - 1; i >= 0; i-- {
		s := services[i]
		s.sig <- SIG_STOP
		s.wg.Wait()
		s.impl.Destroy()
	}
}
func Reload() {
	for _, s := range services {
		s.sig <- SIG_RELOAD
	}
}
func ReportState() {
	for _, s := range services {
		s.sig <- SIG_REPORT
	}
}

//使用gops性能监控
func monitor() {
	if err := agent.Listen(agent.Options{
		Addr:            "",
		ConfigDir:       "",
		ShutdownCleanup: true,
	}); err != nil {
		logger.Fatal("gops listen fail %v", err)
	}
}

//服务启动
func Run(svr ...IService) {
	for _, s := range svr {
		Register(s)
	}
	monitor()
	onInit()
	listenSignal()
}

//监听外部信号
func listenSignal() {
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGABRT, syscall.SIGTERM, syscall.SIGHUP, syscall.Signal(0x1e), syscall.Signal(0x1f), syscall.Signal(29))
	for {
		sig := <-ch
		switch sig {
		case syscall.SIGHUP: //重新读配置文件
			logger.Info("获得SIGHUP（1）信号， reload服务")
			Reload()
		case syscall.Signal(29): //获取服务状态
			logger.Info("获得SIGIO（29）信号， 获取服务状态")
			ReportState()
		default: // 退出
			logger.Info("获得%v（%v）信号， 退出服务", sig, int(sig.(syscall.Signal)))
			stop()
			return
		}
	}
}
