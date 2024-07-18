package config

import (
	"encoding/json"
	"io"
	"log"
	"os"

	"github.com/piaobeizu/plugin-engine/pkg/rds"

	"github.com/caarlos0/env"
	"github.com/sirupsen/logrus"
)

// design idea: we store configurations in redis, allowing us to dynamically adjust our sidecar strategy.
// if reretrieval fails, we'll fallback to using the default configuration.

type Envs struct {
	RedisUrl          string `env:"REDIS_URL"`
	TaskName          string `env:"TASK_NAME"`
	ModelSeries       string `env:"MODEL_SERIES"`
	IP                string `env:"POD_IP"`
	ProxyURL          string `env:"PROXY_URL"`
	ModelURL          string `env:"MODEL_URL" envDefault:"127.0.0.1"`
	ModelPort         int64  `env:"MODEL_PORT" envDefault:"8001"`
	ModelProcessCount int64  `env:"MODEL_PROCESS_COUNT" envDefault:"1"`
	LBPolicy          string `env:"LOAD_BALANCER_POLICY" envDefault:"ONE"`
}

var (
	REDIS_KEY        = "d2:agent:"
	REDIS_KEY_CONFIG = REDIS_KEY + "confi"
	REDIS_KEY_MODELS = REDIS_KEY + "models"
	REDIS_KEY_ROUTES = REDIS_KEY + "routes"
)

type PluginConfig struct {
	Static  any `json:"static,omitempty"`
	Dynamic any `json:"dynamic,omitempty"`
}

type Plugin struct {
	Name     string        `json:"name"`
	Version  string        `json:"version"`
	Enabled  bool          `json:"enabled"`
	Location string        `json:"location"`
	Config   *PluginConfig `json:"config,omitempty"`
}
type Event struct {
	MsgSize int `json:"msg_size"`
}

type Config struct {
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
		jsonFile, err := os.Open("/etc/agent/config.json")
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
