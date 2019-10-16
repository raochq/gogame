package conf

import (
	"flag"
	"github.com/Terry-Mao/goconf"
	"gogame/base/logger"
	"gogame/base/util"
	"os"
)

var (
	logCfg        = logger.DefaultConfig()
	gameServerCfg = &GameServerConfig{
		ServerID: 1301,
	}
	routerCfg = &RouterConfig{
		PortalAddr: 8801,
		ServerAddr: 8802,
	}
	gameGateCfg = &GameGateConfig{
		ListenPort:  8201,
		RouterAddrs: []string{"127.0.0.1:8801"},
	}
	teamCfg = &TeamConfig{
		ServerID:   1601,
		ListenPort: 8601,
	}
	imCfg = &IMConfig{
		ServerID:   1901,
		ListenPort: 8901,
	}

	confFile string
)

type BaseConfig struct {
	ZoneID   int    `goconf:"base:zoneid"`
	ServerID int    `goconf:"router:serverid"`
	GameData string `goconf:"base:gamedata"`
}
type RedisConfig struct {
	RedisMaxIdle     int    `goconf:"redis:idle"`       //最大空闲连接数
	RedisIdleTimeout int    `goconf:"redis:timeout"`    //空闲连接超时时间
	ZoneRedisAddr    string `goconf:"redis:addr"`       //redis密码
	ZoneRedisAuth    string `goconf:"redis:auth"`       //redis地址
	ZoneRedisIndex   int    `goconf:"redis:index"`      //redis的dbindex，集群使用时必须为0
	CrossRedisAddr   string `goconf:"redis:crossaddr"`  //跨服redis地址
	CrossRedisAuth   string `goconf:"redis:crossauth"`  //跨服redis密码
	CrossRedisIndex  int    `goconf:"redis:crossindex"` //跨服redis的dbindex，集群使用时必须为0
}
type GameServerConfig struct {
	BaseConfig
	RedisConfig
	ServerID int `goconf:"gamesvr:serverid"`
}
type RouterConfig struct {
	BaseConfig
	PortalAddr uint16 `goconf:"router:portal_listen"`
	ServerAddr uint16 `goconf:"router:server_listen"`
}
type GameGateConfig struct {
	BaseConfig
	RedisConfig
	ListenPort  uint16   `goconf:"gate:listenPort"`
	RouterAddrs []string `goconf:"gate:routers"`
}
type TeamConfig struct {
	BaseConfig
	ServerID   int    `goconf:"team:serverid"`
	ListenPort uint16 `goconf:"team:server_listen"`
}
type IMConfig struct {
	BaseConfig
	RedisConfig
	ServerID   int    `goconf:"im:serverid"`
	ListenPort uint16 `goconf:"im:server_listen"`
}

func loadConfig(file string) error {
	gconf := goconf.New()
	if err := gconf.Parse(file); err != nil {
		return err
	}
	baseCfg := BaseConfig{
		ZoneID:   1,
		GameData: "./data",
	}
	if err := gconf.Unmarshal(&baseCfg); err != nil {
		return err
	}

	if err := gconf.Unmarshal(&logCfg); err != nil {
		return err
	}
	redisCfg := RedisConfig{
		RedisMaxIdle:     10,
		RedisIdleTimeout: 240e9,
		ZoneRedisAddr:    "127.0.0.1:6379",
		ZoneRedisAuth:    "123456",
		ZoneRedisIndex:   0,
		CrossRedisAddr:   "127.0.0.1:6379",
		CrossRedisAuth:   "123456",
		CrossRedisIndex:  8,
	}
	if err := gconf.Unmarshal(&redisCfg); err != nil {
		return err
	}

	if err := gconf.Unmarshal(gameGateCfg); err != nil {
		return err
	} else {
		gameGateCfg.BaseConfig = baseCfg
		gameGateCfg.RedisConfig = redisCfg
	}
	if err := gconf.Unmarshal(gameServerCfg); err != nil {
		return err
	} else {
		gameServerCfg.BaseConfig = baseCfg
		gameServerCfg.RedisConfig = redisCfg
	}
	if err := gconf.Unmarshal(routerCfg); err != nil {
		return err
	} else {
		routerCfg.BaseConfig = baseCfg
	}

	if err := gconf.Unmarshal(imCfg); err != nil {
		return err
	} else {
		imCfg.BaseConfig = baseCfg
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
	logger.ReloadConfig(logCfg)
}

func GameGateConf() *GameGateConfig {
	return gameGateCfg
}
func RouteConf() *RouterConfig {
	return routerCfg
}
func TeamConf() *TeamConfig {
	return teamCfg
}
func IMConf() *IMConfig {
	return imCfg
}
