package conf

import (
	"flag"
	"github.com/Terry-Mao/goconf"
	"gogame/base/logger"
	"gogame/base/util"
	"os"
)

var (
	Cfg = &Config{
		Base: BaseConfig{
			ZoneID:   1,
			GameData: "./data",
		},
		Log: logger.DefaultConfig(),
		GameServer: GameServerConfig{
			ServerID: 1301,
		},
		Router: RouterConfig{
			ServerID:   1801,
			PortalAddr: 8801,
			ServerAddr: 8802,
		},
		GameGate: GameGateConfig{
			ServerID:   1201,
			ListenPort: 8201,
		},
		Team: TeamConfig{
			ServerID:   1601,
			ListenPort: 8601,
		},
		IM: IMConfig{
			ServerID:   1901,
			ListenPort: 8901,
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
	Team       TeamConfig
	IM         IMConfig
}
type BaseConfig struct {
	ZoneID   int    `goconf:"base:zoneid"`
	GameData string `goconf:"base:gamedata"`
}
type GameServerConfig struct {
	ServerID int `goconf:"gamesvr:serverid"`
}
type RouterConfig struct {
	ServerID   int    `goconf:"router:serverid"`
	PortalAddr uint16 `goconf:"router:portal_listen"`
	ServerAddr uint16 `goconf:"router:server_listen"`
}
type GameGateConfig struct {
	ServerID   int    `goconf:"gate:serverid"`
	ListenPort uint16 `goconf:"gate:server_listen"`
}
type TeamConfig struct {
	ServerID   int    `goconf:"team:serverid"`
	ListenPort uint16 `goconf:"team:server_listen"`
}
type IMConfig struct {
	ServerID   int    `goconf:"im:serverid"`
	ListenPort uint16 `goconf:"im:server_listen"`
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
