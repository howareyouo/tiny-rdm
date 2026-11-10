package services

import (
	"context"
	"strconv"
	"sync"
	"time"
	"tinyrdm/backend/types"

	"github.com/redis/go-redis/v9"
)

type pubsubItem struct {
	client    redis.UniversalClient
	pubsub    *redis.PubSub
	stream    *serverStream
}

type subMessage struct {
	Timestamp int64  `json:"timestamp"`
	Channel   string `json:"channel"`
	Message   string `json:"message"`
}

type pubsubService struct {
	ctx       context.Context
	ctxCancel context.CancelFunc
	session   *redisSession
	mutex     sync.Mutex
	items     map[string]*pubsubItem
}

var pubsub *pubsubService
var oncePubsub sync.Once

func Pubsub() *pubsubService {
	if pubsub == nil {
		oncePubsub.Do(func() {
			pubsub = &pubsubService{
				session: newRedisSession(nil),
				items:   map[string]*pubsubItem{},
			}
		})
	}
	return pubsub
}

func (p *pubsubService) Start(ctx context.Context) {
	p.ctx, p.ctxCancel = context.WithCancel(ctx)
	p.session.Start(ctx)
}

// Publish publish message to channel
func (p *pubsubService) Publish(server, channel, payload string) (resp types.JSResp) {
	rdb, err := Browser().getRedisClient(server, -1)
	if err != nil {
		resp.Msg = err.Error()
		return
	}

	var received int64
	received, err = rdb.client.Publish(p.ctx, channel, payload).Result()
	if err != nil {
		resp.Msg = err.Error()
		return
	}

	resp.Success = true
	resp.Data = struct {
		Received int64 `json:"received"`
	}{
		Received: received,
	}
	return
}

// StartSubscribe start to subscribe a channel
func (p *pubsubService) StartSubscribe(server string) (resp types.JSResp) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// stop the previous subscription if it is still running
	if item, ok := p.items[server]; ok && item.stream != nil {
		item.stream.stop(func() {
			item.pubsub.Close()
		})
		p.session.close(server)
		delete(p.items, server)
	}

	sess, err := p.session.get(server, -1)
	if err != nil {
		resp.Msg = err.Error()
		return
	}

	item := &pubsubItem{
		client: sess.client,
	}
	item.stream = newServerStream(p.ctx, "sub:"+strconv.FormatInt(time.Now().UnixNano(), 10), 300, 300*time.Millisecond)
	item.pubsub = sess.client.PSubscribe(p.ctx, "*")
	p.items[server] = item

	go p.processSubscribe(item.stream, item.pubsub.Channel())
	resp.Success = true
	resp.Data = struct {
		EventName string `json:"eventName"`
	}{
		EventName: item.stream.eventName,
	}
	return
}

func (p *pubsubService) processSubscribe(stream *serverStream, ch <-chan *redis.Message) {
	for {
		select {
		case data := <-ch:
			if data == nil {
				// subscription closed
				return
			}
			stream.push(subMessage{
				Timestamp: time.Now().UnixMilli(),
				Channel:   data.Channel,
				Message:   data.Payload,
			})
		case <-stream.closeCh:
			return
		}
	}
}

// StopSubscribe stop subscribe by server name
func (p *pubsubService) StopSubscribe(server string) (resp types.JSResp) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	item, ok := p.items[server]
	if !ok || item.stream == nil {
		resp.Success = true
		return
	}

	item.stream.stop(func() {
		item.pubsub.Close()
	})
	p.session.close(server)
	delete(p.items, server)
	resp.Success = true
	return
}

// StopAll stop all subscribe
func (p *pubsubService) StopAll() {
	if p.ctxCancel != nil {
		p.ctxCancel()
	}

	for server := range p.items {
		p.StopSubscribe(server)
	}
}
