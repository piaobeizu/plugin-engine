/*
 @Version : 1.0
 @Author  : steven.wong
 @Email   : 'wwangxiaoakng@modelbest.cn'
 @Time    : 2024/04/30 13:59:16
 Desc     :
*/

package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/piaobeizu/plugin-engine/event"

	"github.com/panjf2000/ants/v2"
	"github.com/sirupsen/logrus"
)

// type PluginHealth string

// const (
//
//	PluginHealthGreen  PluginHealth = "green"
//	PluginHealthYellow PluginHealth = "yellow"
//	PluginHealthRed    PluginHealth = "red"
//
// )
type PluginName string

type PluginState string

const (
	PluginStateCreate  PluginState = "create"
	PluginStateInit    PluginState = "init"
	PluginStateRunning PluginState = "running"
	PluginStateStopped PluginState = "stopped"
	PluginStatePaused  PluginState = "paused"
)

type PluginMetrics struct {
}

type Plugin interface {
	Init() error             // init the plugin
	Run()                    // run the plugin
	Stop()                   // stop the plugin
	Metrics() *PluginMetrics // stats the plugin
	Health() PluginState     // get the health of the plugin
	RefreshConfig(cfg any)   // refresh the config of the plugin
}

type BasePlugin struct {
	Context context.Context
	Cancel  context.CancelFunc
	Name    PluginName
	Version string
	Config  any
	Event   *event.Event
	State   PluginState
	Log     *logrus.Entry
	Mu      *sync.RWMutex
	Stop    chan error
	Pool    *ants.Pool
}

func (b *BasePlugin) ParseConfig(cfg any) error {
	b.Mu.RLock()
	defer b.Mu.RUnlock()
	str, err := json.Marshal(b.Config)
	if err != nil {
		return err
	}
	err = json.Unmarshal(str, cfg)
	if err != nil {
		return err
	}
	return nil
}

func (b *BasePlugin) SetConfig(cfg any) {
	b.Mu.Lock()
	defer b.Mu.Unlock()
	b.Config = cfg
}

func (b *BasePlugin) BaseInit() error {
	if b.Event == nil {
		return fmt.Errorf("queue is nil")
	}
	if b.Event.GetEvents()["father"] == nil {
		return fmt.Errorf("queue father is nil")
	}
	if b.Event.GetEvents()[string(b.Name)] == nil {
		return fmt.Errorf("queue %s is nil", b.Name)
	}
	return nil
}

func (b *BasePlugin) GetState() PluginState {
	return b.State
}

func (b *BasePlugin) GetName() PluginName {
	return b.Name
}

func (b *BasePlugin) GetVersion() string {
	return b.Version
}

func (b *BasePlugin) SendEvent(to PluginName, action string, data any) error {
	msg := event.NewMessage(action, data)
	return b.Event.Send(string(to), msg)
}

func (b *BasePlugin) RecvEvent() chan *event.Message {
	return b.Event.Receive(string(b.Name))
}
