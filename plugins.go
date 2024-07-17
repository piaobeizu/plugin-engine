/*
 @Version : 1.0
 @Author  : steven.wong
 @Email   : 'wwangxiaoakng@modelbest.cn'
 @Time    : 2024/04/30 10:26:58
 Desc     :
*/

package engine

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/piaobeizu/plugin-engine/config"
	"github.com/piaobeizu/plugin-engine/event"
	"github.com/piaobeizu/plugin-engine/plugins"

	"github.com/panjf2000/ants/v2"
	"github.com/sirupsen/logrus"
)

type Plugin struct {
	ctx      context.Context
	cancel   context.CancelFunc
	Name     plugins.PluginName
	Version  string
	Location string
	Config   any
	queue    *event.Event
	pool     *ants.Pool
	plugin   plugins.Plugin
}

func NewPlugin(cfg config.Plugin, queue *event.Event, pool *ants.Pool, plugin plugins.Plugin) *Plugin {
	ctx, cancel := context.WithCancel(context.TODO())
	return &Plugin{
		ctx:      ctx,
		cancel:   cancel,
		Name:     plugins.PluginName(cfg.Name),
		Version:  cfg.Version,
		Location: cfg.Location,
		Config:   cfg.Config,
		queue:    queue,
		pool:     pool,
		plugin:   plugin,
	}
}

func (p *Plugin) Run() error {
	// do something
	if p.plugin == nil {
		return fmt.Errorf("plugin[%s] not found", p.Name)
	}
	if err := p.plugin.Init(); err != nil {
		return err
	}
	p.pool.Submit(p.plugin.Run)
	return nil
}

func (p *Plugin) Stop() {
	if p.plugin == nil {
		logrus.Errorf("plugin[%s] not found", p.Name)
	}
	p.cancel()
	p.plugin.Stop()
}

func (p *Plugin) Status() error {
	// do something
	return nil
}

func (p *Plugin) Health() plugins.PluginState {
	// do something
	if p.plugin == nil {
		logrus.Errorf("plugin[%s] not found", p.Name)
		return ""
	}
	return p.plugin.Health()
}

func (p *Plugin) Restart() error {
	// do something
	return nil
}

func (p *Plugin) RefreshCongfig(cfg any) {
	// do something
	if p.plugin == nil {
		logrus.Errorf("plugin[%s] not found", p.Name)
		panic(fmt.Errorf("plugin[%s] not found", p.Name))
	}
	p.plugin.RefreshConfig(cfg)
}

type Plugins struct {
	ctx      context.Context
	plugins  map[string]*Plugin
	config   []config.Plugin
	queue    *event.Event
	mu       sync.RWMutex
	pool     *ants.Pool
	pplugins map[plugins.PluginName]plugins.Plugin
}

func NewPlugins(ctx context.Context, config []config.Plugin, q *event.Event, pool *ants.Pool) *Plugins {
	plugins := &Plugins{
		ctx:      ctx,
		plugins:  map[string]*Plugin{},
		config:   config,
		queue:    q,
		mu:       sync.RWMutex{},
		pool:     pool,
		pplugins: map[plugins.PluginName]plugins.Plugin{},
	}
	return plugins
}

func (p *Plugins) AddPPlugin(name plugins.PluginName, plugin plugins.Plugin) *Plugins {
	p.pplugins[name] = plugin
	return p
}

func (p *Plugins) Start() {
	check := make(chan struct{}, 1)
	check <- struct{}{}
	// ticker for checking plugins
	timer := time.NewTimer(10 * time.Millisecond)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			ps := config.GetPlugins()
			p.mu.Lock()
			for _, plugin := range ps {
				if !plugin.Enabled {
					if _, ok := p.plugins[plugin.Name]; ok {
						p.plugins[plugin.Name].Stop()
						p.queue.Remove(plugin.Name)
						delete(p.plugins, plugin.Name)
					}
				} else if _, ok := p.plugins[plugin.Name]; !ok {
					p.queue.Register(plugin.Name)
					newPlugin := NewPlugin(plugin, p.queue, p.pool, p.pplugins[plugins.PluginName(plugin.Name)])
					p.plugins[plugin.Name] = newPlugin
				} else if p.plugins[plugin.Name].Health() == plugins.PluginStateStopped ||
					plugin.Version != p.plugins[plugin.Name].Version {
					logrus.WithFields(logrus.Fields{"plugin": plugin.Name}).Warn("plugin is not running, restart it")
					newPlugin := NewPlugin(plugin, p.queue, p.pool, p.pplugins[plugins.PluginName(plugin.Name)])
					p.plugins[plugin.Name] = newPlugin
				} else {
					p.plugins[plugin.Name].RefreshCongfig(plugin.Config)
				}
			}
			for _, plugin := range p.plugins {
				if plugin.Health() == plugins.PluginStateCreate {
					if err := plugin.Run(); err != nil {
						logrus.WithFields(logrus.Fields{"plugin": plugin.Name}).Errorf("plugin run failed: %+v", err)
					}
				}
			}
			p.mu.Unlock()
			logrus.Info("agent is running")
			timer.Reset(5 * time.Second)
		case <-p.ctx.Done():
			close(check)
			for _, plugin := range p.plugins {
				plugin.Stop()
				p.queue.Remove(string(plugin.Name))
				time.Sleep(100 * time.Millisecond)
				logrus.WithFields(logrus.Fields{"plugin": plugin.Name}).Info("plugin stoped")
			}
			return
		}
	}
}

func (p *Plugins) IsAllRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, plugin := range p.plugins {
		if plugin.Health() != plugins.PluginStateRunning {
			logrus.Infof("plugin[%s] is not running", plugin.Name)
			return false
		}
	}
	return true
}

func (p *Plugins) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	allDone := true
	p.pool.Submit(func() {
		for {
			allDone = true
			for _, plugin := range p.plugins {
				if plugin.Health() != plugins.PluginStateStopped {
					allDone = false
					continue
				}
			}
			if allDone {
				break
			}
			time.Sleep(300 * time.Millisecond)
		}
		cancel()
	})
	<-ctx.Done()
	if !allDone {
		cancel()
		os.Exit(1)
	}
}

func (p *Plugins) Close(plugin plugins.PluginName) {
	if _, ok := p.plugins[string(plugin)]; ok {
		p.plugins[string(plugin)].Stop()
	}
}
