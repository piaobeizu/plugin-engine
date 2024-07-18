/*
 @Version : 1.0
 @Author  : steven.wong
 @Email   : 'wwangxiaoakng@modelbest.cn'
 @Time    : 2024/04/30 16:15:48
 Desc     :
*/

package core

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/piaobeizu/plugin-engine/pkg/config"
	"github.com/piaobeizu/plugin-engine/pkg/event"
	"github.com/piaobeizu/plugin-engine/pkg/utils"

	"github.com/panjf2000/ants/v2"
	"github.com/sirupsen/logrus"
)

type Agent struct {
	ctx     context.Context
	cancel  context.CancelFunc
	plugins *Plugins
	singal  chan os.Signal
	stop    chan struct{}
	pool    *ants.Pool
	queue   *event.Event
}

// handlepanic is a panic handler for agent
func (a *Agent) handlepanic(i interface{}) {
	file, line := utils.FindCaller(5)
	logrus.WithFields(logrus.Fields{
		"panic": fmt.Sprintf("%s:%d", file, line),
	}).Errorf("%v\n%s", i, string(debug.Stack()))
	//TODO: we can achieve to restart the plugin here
	a.stop <- struct{}{}
}

func NewAgent(ctx context.Context, cancel context.CancelFunc) *Agent {
	// step 1, we need to get the configurations
	logrus.Print("start to get the configurations")
	conf := config.GetConfig()

	// step 2, we need to create an agent
	a := &Agent{
		ctx:    ctx,
		cancel: cancel,
		singal: make(chan os.Signal, 1),
		stop:   make(chan struct{}, 1),
	}
	// step 3, we need to create a goroutine pool
	pool, err := ants.NewPool(2000, func(opts *ants.Options) {
		opts.ExpiryDuration = 60 * time.Second
		opts.Nonblocking = false
		opts.PreAlloc = true
		opts.MaxBlockingTasks = 10
		opts.PanicHandler = a.handlepanic
	})
	if err != nil {
		panic(err)
	}
	// step 4, we need to create a queue
	queue := event.NewEvent(ctx, config.GetEvent(), pool)
	queue.Register("father")

	// step 5, we need to create a plugins
	plugins := NewPlugins(ctx, conf.Plugins, queue, pool)

	// step 6, we need to set the plugins and pool to the agent
	a.plugins = plugins
	a.pool = pool
	a.queue = queue

	// step 7, we need to register the signals
	signal.Notify(a.singal, syscall.SIGTERM)
	signal.Notify(a.singal, syscall.SIGINT)
	signal.Notify(a.singal, syscall.SIGQUIT)
	signal.Notify(a.singal, syscall.SIGHUP)
	return a
}

func (a *Agent) Start() error {
	// start the plugins
	a.pool.Submit(a.plugins.Start)
	ticker := time.NewTicker(1000 * time.Millisecond)
	// graceful shutdown
	a.pool.Submit(func() {
		for {
			select {
			case <-a.ctx.Done():
				logrus.Info("context done, stop the server")
				a.stop <- struct{}{}
				return
			case <-a.singal:
				logrus.Info("receive signal, stop the server")
				a.stop <- struct{}{}
				return
			// check all plugins status and write to file,
			// this file will be used by the k8s liveness probe
			case <-ticker.C:
				allRunning := 0
				if a.plugins.IsAllRunning() {
					allRunning = 1
				}
				if err := utils.WriteFile(allRunning, "/var/run/agent/allrun", true); err != nil {
					panic(err)
				}
			}
		}
	})
	return nil
}

func (a *Agent) Wait() chan struct{} {
	return a.stop
}

func (a *Agent) Stop() {
	a.cancel()
	a.plugins.Stop()
	a.queue.Close()
	a.pool.Release()
	close(a.stop)
	logrus.Printf("agent stopped, byebye!")
}
