package conf

import (
	"flag"
	"github.com/Terry-Mao/goconf"
	"github.com/raochq/gogame/base/logger"
	"github.com/raochq/gogame/base/util"
	"os"
)

var (
	Cfg = &Config{
		Base: BaseConfig{
			ZoneID:   1,
			ServerID: 1,
			GameData: "./data",
		},
		Log:        logger.DefaultConfig(),
		GameServer: GameServerConfig{},
		Router: RouterConfig{
			PortalAddr: 8010,
			ServerAddr: 8011,
		},
		GameGate: GameGateConfig{
			ListenPort: 8080,
		},
	}
	confFile string
)

type Config struct {
	Base       BaseConfig
	Log        logger.Config
	GameServer GameServerConfig
	Router     RouterConfig
	GameGate   GameGateConfig
}
type BaseConfig struct {
	ZoneID   int    `goconf:"base:zoneid"`
	GameData string `goconf:"base:gamedata"`
	ServerID int    `goconf:"base:serverid"`
}
type GameServerConfig struct {
}
type RouterConfig struct {
	PortalAddr uint16 `goconf:"router:portal_listen"`
	ServerAddr uint16 `goconf:"router:server_listen"`
}
type GameGateConfig struct {
	ListenPort uint16 `goconf:"gate:server_listen"`
}

func loadConfig(file string) error {
	gconf := goconf.New()
	if err := gconf.Parse(file); err != nil {
		return err
	}
	if err := gconf.Unmarshal(&Cfg.Base); err != nil {
		return err
	}
	if err := gconf.Unmarshal(&Cfg.Log); err != nil {
		return err
	}
	if err := gconf.Unmarshal(&Cfg.GameServer); err != nil {
		return err
	}
	if err := gconf.Unmarshal(&Cfg.GameGate); err != nil {
		return err
	}
	if err := gconf.Unmarshal(&Cfg.Router); err != nil {
		return err
	}
	return nil
}

func init() {
	flag.StringVar(&confFile, "conf", "", " set config file path")
	flag.Parse()
	if confFile == "" {
		fileName := util.GetAppName() + ".conf"
		if !util.Exists(fileName) {
			fp, _ := os.UserConfigDir()
			fp = fp + "/" + fileName
			if util.Exists(fp) {
				confFile = fp
			} else {
				fp = util.GetAppPath() + "/" + fileName
				if util.Exists(fp) {
					confFile = fp
				}
			}
		}
	}
	if confFile != "" {
		if !util.Exists(confFile) {
			logger.Fatal("init failed, not found %s", confFile)
		}
		err := loadConfig(confFile)
		if err != nil {
			logger.Fatal("init failed,loadConfig error %v", err)
		}
	}
	logger.ReloadConfig(Cfg.Log)
}
