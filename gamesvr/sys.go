package main

import (
	"flag"
	"github.com/Terry-Mao/goconf"
	"github.com/raochq/gogame/base/logger"
	"github.com/raochq/gogame/base/util"
	"path/filepath"
)

var (
	conf     *Config
	confFile string
	server   *Server
)

type Server struct {
}

type Config struct {
	ZoneID   int    `goconf:"base:zoneid"`
	ServerID int    `goconf:"base:serverid"`
	GameData string `goconf:"base:gamedata"`
	LogPath  string `goconf:"log:path"`
	LogLevel int    `goconf:"log:level"`
}

func (g *Server) Init() {
	logger.Info("=============== Server start ===============")
	g.Reload()
}

func (g *Server) Destroy() {
	logger.Info("=============== Server stop ===============\n")
}

func (g *Server) MainLoop(sig <-chan byte) {
	<-sig
}

func (g *Server) Reload() {
	logger.Info("Reload csv files")
}

func (g *Server) ReportState() {
	logger.Info("ReportState")
}

func loadConfig(file string) (*Config, error) {
	gconf := goconf.New()
	if err := gconf.Parse(file); err != nil {
		return nil, err
	}

	cfg := &Config{
		ZoneID:   1,
		ServerID: 1,
		GameData: "./data",
		LogPath:  "./gamesvr.log",
		LogLevel: 3,
	}
	if err := gconf.Unmarshal(cfg); err != nil {
		return nil, err
	}
	if !filepath.IsAbs(cfg.GameData) {
		cfg.GameData, _ = filepath.Abs(util.GetAppPath() + "/" + cfg.GameData)
	}
	if !filepath.IsAbs(cfg.LogPath) {
		cfg.LogPath, _ = filepath.Abs(util.GetAppPath() + "/" + cfg.LogPath)
	}
	return cfg, nil
}
func InitCfg() error {
	flag.StringVar(&confFile, "conf", "", " set config file path")
	flag.Parse()
	if confFile == "" {
		fp, _ := filepath.Abs(util.GetAppPath() + "/../etc")
		if !util.Exists(fp) {
			fp = util.GetAppPath()
		}
		confFile = fp + "/" + util.GetAppName() + ".conf"
	}
	var err error
	conf, err = loadConfig(confFile)
	if err != nil {
		return err
	}
	return nil
}
