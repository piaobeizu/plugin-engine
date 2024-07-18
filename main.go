/*
 @Version : 1.0
 @Author  : steven.wong
 @Email   : 'wwangxiaoakng@modelbest.cn'
 @Time    : 2024/04/30 10:21:36
 Desc     :
*/

package main

import (
	"context"
	"runtime"
	"strings"

	"github.com/piaobeizu/plugin-engine/pkg/core"
	"github.com/piaobeizu/plugin-engine/pkg/log"

	"github.com/sirupsen/logrus"
)

var agent *core.Agent

func stop() {
	if err := recover(); err != nil {
		buf := make([]byte, 1<<16)
		ss := runtime.Stack(buf, true)
		msg := string(buf[:ss])
		var bt []string
		for _, row := range strings.Split(msg, "\n") {
			if !strings.HasPrefix(row, "\t") {
				continue
			}
			if strings.Contains(row, "main.") {
				continue
			}
			if strings.Contains(row, "panic") {
				continue
			}
			bt = append(bt, strings.TrimSpace(row))
		}
		logrus.Errorf("panic: %v\n", strings.Join(bt, "\n"))
	}
	agent.Stop()
}

func main() {
	defer stop()
	log.InitLog()
	ctx, cancel := context.WithCancel(context.Background())
	// start the agent
	agent = core.NewAgent(ctx, cancel)
	agent.Start()
	<-agent.Wait()
}
