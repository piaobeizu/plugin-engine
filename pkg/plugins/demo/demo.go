/*
 @Version : 1.0
 @Author  : steven.wong
 @Email   : 'wangxk1991@gamil.com'
 @Time    : 2024/07/18 17:45:46
 Desc     :
*/

package demo

import (
	"context"
	"sync"

	"github.com/creasty/defaults"
	"github.com/panjf2000/ants/v2"
	"github.com/piaobeizu/plugin-engine/pkg/event"
	"github.com/piaobeizu/plugin-engine/pkg/plugins"
	"github.com/sirupsen/logrus"
)

type DemoConfig struct {
	Static  struct{} `json:"static" default:"{}"`
	Dynamic struct{} `json:"dynamic" default:"{}"`
}

type DemoPlugin struct {
	plugins.BasePlugin
}

var (
	plugin *DemoPlugin
)

func NewDemoPlugin(
	ctx context.Context,
	name plugins.PluginName,
	version string,
	cfg any,
	event *event.Event,
	pool *ants.Pool) *DemoPlugin {
	if plugin != nil {
		return plugin
	}
	subctx, cancel := context.WithCancel(ctx)
	plugin = &DemoPlugin{
		BasePlugin: plugins.BasePlugin{
			Context: subctx,
			Cancel:  cancel,
			Name:    name,
			Version: version,
			Config:  cfg,
			Event:   event,
			State:   plugins.PluginStateCreate,
			Mu:      &sync.RWMutex{},
			Pool:    pool,
		},
	}
	plugin.Log = logrus.WithFields(logrus.Fields{
		"plugin": name,
	})
	return plugin
}

func (p *DemoPlugin) Init() error {
	return nil
}

func (p *DemoPlugin) Run() {
	p.Pool.Submit(p.ListenEvent)
	p.State = plugins.PluginStateRunning

}

func (p *DemoPlugin) Stop() {
	p.State = plugins.PluginStateStopped
}

func (p *DemoPlugin) Metrics() *plugins.PluginMetrics {
	return nil
}

func (p *DemoPlugin) Health() plugins.PluginState {
	return p.State
}

func (p *DemoPlugin) ListenEvent() {

}

func (p *DemoPlugin) Status() error {
	return nil
}

func (p *DemoPlugin) RefreshConfig(cfg any) {
	p.SetConfig(cfg)
}

func (p *DemoPlugin) getCfg() *DemoConfig {
	var cfg *DemoConfig = &DemoConfig{}
	if err := p.ParseConfig(cfg); err != nil {
		if err := defaults.Set(cfg); err != nil {
			cfg = &DemoConfig{
				Static:  struct{}{},
				Dynamic: struct{}{},
			}
		}
	}
	return cfg
}
