/*
 @Version : 1.0
 @Author  : steven.wong
 @Email   : 'wwangxiaoakng@modelbest.cn'
 @Time    : 2024/04/30 10:27:44
 Desc     :
*/

package event

import (
	"context"
	"fmt"
	"math/rand"
	"sync"

	"github.com/piaobeizu/plugin-engine/pkg/config"

	"github.com/panjf2000/ants/v2"
	"github.com/sirupsen/logrus"
)

type MsgKind string

var (
	letters = []rune("12345678abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
)

const (
	MSG_KINE_PANIC      MsgKind = "panic"
	MSG_KIND_GRPC_BATCH MsgKind = "grpc.batch"
	MSG_KIND_GRPC_MODEL MsgKind = "grpc.model"
)

type Message struct {
	id     string
	Action string
	Data   any
}

func id() string {
	b := make([]rune, 8)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
func (m *Message) String() string {
	return fmt.Sprintf("%s|%s|%+v", m.id, m.Action, m.Data)
}

func NewMessage(action string, data any) *Message {
	return &Message{
		id:     id(),
		Action: action,
		Data:   data,
	}
}

type Event struct {
	mu       sync.RWMutex
	ctx      context.Context
	config   *config.Event
	pool     *ants.Pool
	messages map[string]chan *Message
	funcs    map[string][]*Func
}

type Func struct {
	ctx    context.Context
	action string
	pool   *ants.Pool
	sync   bool
	panic  bool
	f      func(param any) error
	done   chan struct{}
}

func NewFunc(ctx context.Context, action string, f func(param any) error) *Func {
	subctx := context.WithoutCancel(ctx)
	return &Func{
		ctx:    subctx,
		action: action,
		f:      f,
	}
}

func (F *Func) Do(msg *Message) (err error) {
	if F.sync {
		for {
			select {
			case <-F.ctx.Done():
				logrus.WithFields(logrus.Fields{
					"func": F.action,
				}).Warnf("context done, msg: %s", msg)
				return nil
			default:
				err = F.f(msg.Data)
				if err != nil && F.panic {
					panic(err)
				}
				return err
			}
		}
	}
	F.pool.Submit(func() {
		for {
			select {
			case <-F.ctx.Done():
				logrus.WithFields(logrus.Fields{
					"func": F.action,
				}).Warnf("context done, msg: %s", msg)
				F.done <- struct{}{}
				return
			default:
				err = F.f(msg.Data)
				F.done <- struct{}{}
			}
		}
	})
	<-F.done
	if err != nil && F.panic {
		panic(err)
	}
	return err
}

func (F *Func) Sync(sync bool) *Func {
	if F == nil {
		panic("nil func")
	}
	F.sync = sync
	return F
}

func (F *Func) Panic(p bool) *Func {
	if F == nil {
		panic("nil func")
	}
	F.panic = p
	return F
}

func NewEvent(ctx context.Context, conf *config.Event, pool *ants.Pool) *Event {
	if ctx == nil {
		ctx = context.Background()
	}
	if conf == nil {
		conf = &config.Event{
			MsgSize: 1000,
		}
	}
	if pool == nil {
		pool, _ = ants.NewPool(1000)
	}
	return &Event{
		ctx:      ctx,
		mu:       sync.RWMutex{},
		config:   conf,
		pool:     pool,
		messages: make(map[string]chan *Message, 4),
		funcs:    make(map[string][]*Func, 4),
	}
}

func (q *Event) Register(name string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if _, ok := q.messages[name]; ok {
		return
	}
	q.messages[name] = make(chan *Message, q.config.MsgSize)
}

func (q *Event) Funcs(name string, action string) *Func {
	q.mu.RLock()
	defer q.mu.RUnlock()
	if _, ok := q.funcs[name]; !ok {
		panic(fmt.Sprintf("func:%s not found", name))
	}
	for _, f := range q.funcs[name] {
		if f.action == action {
			return f
		}
	}
	panic(fmt.Sprintf("func:%s.%s not found", name, action))
}

func (q *Event) AddFunc(name string, F *Func) {
	q.mu.RLock()
	defer q.mu.RUnlock()
	if _, ok := q.funcs[name]; !ok {
		q.funcs[name] = []*Func{}
	}
	q.funcs[name] = append(q.funcs[name], F)
}

// Remove removes a name from the queue
func (q *Event) Remove(name string) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if _, ok := q.messages[name]; !ok {
		return fmt.Errorf("queue of name:%s not found", name)
	}
	close(q.messages[name])
	delete(q.messages, name)
	return nil
}

func (q *Event) Send(name string, msg *Message) error {
	q.mu.RLock()
	defer q.mu.RUnlock()
	if _, ok := q.messages[name]; !ok {
		return fmt.Errorf("queue of name[%s] not found", name)
	}
	if len(q.messages[name]) >= q.config.MsgSize {
		logrus.WithFields(logrus.Fields{
			"name": name,
			"size": len(q.messages[name]),
		}).Warnf("queue of name[%s] is full", name)
		<-q.messages[name]
	}
	q.messages[name] <- msg
	return nil
}

func (q *Event) Receive(name string) chan *Message {
	return q.messages[name]
}

func (q *Event) GetEvents() map[string]chan *Message {
	return q.messages
}

func (q *Event) Close() {
	for k := range q.messages {
		close(q.messages[k])
		delete(q.messages, k)
	}
}

func (q *Event) Status() map[string]int {
	ret := make(map[string]int, 1)
	for k, v := range q.messages {
		ret[k] = len(v)
	}
	return ret
}
