package config

import (
	"encoding/json"
	"io"
	"log"
	"os"

	"github.com/piaobeizu/plugin-engine/rds"

	"github.com/caarlos0/env"
	"github.com/sirupsen/logrus"
)

// design idea: we store configurations in redis, allowing us to dynamically adjust our sidecar strategy.
// if reretrieval fails, we'll fallback to using the default configuration.

type Envs struct {
	REDIS_KEY    string `env:"PE_REDIS_KEY" envDefault:"pe:agent:"`
	LOCAL_CONFIG string `env:"PE_LOCAL_CONFIG" envDefault:"/etc/agent/config.json"`
	DEBUG_MODE   string `env:"PE_DEBUG_MODE" envDefault:"debug"`
}

var (
	REDIS_KEY_CONFIG = GetEnvs().REDIS_KEY + "config"
	REDIS_KEY_MODELS = GetEnvs().REDIS_KEY + "models"
	REDIS_KEY_ROUTES = GetEnvs().REDIS_KEY + "routes"
)

type PluginConfig struct {
	Static  any `json:"static,omitempty"`
	Dynamic any `json:"dynamic,omitempty"`
}

type Plugin struct {
	Name     string        `json:"name"`
	Package  string        `json:"package"`
	Version  string        `json:"version"`
	Enabled  bool          `json:"enabled"`
	Location string        `json:"location"`
	Config   *PluginConfig `json:"config,omitempty"`
}
type Event struct {
	MsgSize int `json:"msg_size"`
}

type GoroutinePool struct {
	Size             int `json:"size"`
	ExpiryDuration   int `json:"expiry_duration"`
	MaxBlockingTasks int `json:"max_blocking_tasks"`
}
type System struct {
	GoroutinePool *GoroutinePool `json:"goroutine_pool"`
}

type Config struct {
	System  *System  `json:"system"`
	Event   *Event   `json:"event"`
	Plugins []Plugin `json:"plugins"`
}

func GetConfig() Config {
	// define default config
	defaultCfg := Config{}
	rds := rds.RedisClient()
	agent, err := rds.Get(REDIS_KEY_CONFIG)
	if err != nil {
		logrus.Warnf("Failed to get config from redis: %+v, try to read it from config file", err)
		jsonFile, err := os.Open(GetEnvs().LOCAL_CONFIG)
		if err != nil {
			panic(err)
		}
		// 要记得关闭
		defer jsonFile.Close()
		jsons, err := io.ReadAll(jsonFile)
		if err != nil {
			panic(err)
		}
		agent = string(jsons)
	}
	var ret Config
	err = json.Unmarshal([]byte(agent), &ret)
	if err != nil {
		logrus.Warnf("Failed to unmarshal config[%s] from redis: %+v", agent, err)
		ret = defaultCfg
	}

	return ret
}

func GetPlugins() []Plugin {
	return GetConfig().Plugins
}

func GetEvent() *Event {
	return GetConfig().Event
}

func GetEnvs() Envs {
	envs := Envs{}
	if err := env.Parse(&envs); err != nil {
		log.Fatalf("Failed to parse config from env: %+v", err)
	}
	return envs
}
