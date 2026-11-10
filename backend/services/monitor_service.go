package services

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"
	"tinyrdm/backend/types"

	"github.com/redis/go-redis/v9"
)

type monitorItem struct {
	client    *redis.Client
	cmd       *redis.MonitorCmd
	ch        chan string
	stream    *serverStream
}

type monitorService struct {
	ctx       context.Context
	ctxCancel context.CancelFunc
	session   *redisSession
	mutex     sync.Mutex
	items     map[string]*monitorItem
}

var monitor *monitorService
var onceMonitor sync.Once

func Monitor() *monitorService {
	if monitor == nil {
		onceMonitor.Do(func() {
			monitor = &monitorService{
				session: newRedisSession(nil),
				items:   map[string]*monitorItem{},
			}
		})
	}
	return monitor
}

func (c *monitorService) Start(ctx context.Context) {
	c.ctx, c.ctxCancel = context.WithCancel(ctx)
	c.session.Start(ctx)
}

// StartMonitor start a monitor by server name
func (c *monitorService) StartMonitor(server string) (resp types.JSResp) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// stop the previous monitor if it is still running
	if item, ok := c.items[server]; ok && item.stream != nil {
		item.stream.stop(func() {
			item.client.Close()
			item.cmd.Stop()
		})
		c.session.close(server)
		delete(c.items, server)
	}

	sess, err := c.session.get(server, -1)
	if err != nil {
		resp.Msg = err.Error()
		return
	}
	client, ok := sess.client.(*redis.Client)
	if !ok {
		resp.Msg = "create redis client fail"
		return
	}

	item := &monitorItem{
		client: client,
		ch:     make(chan string),
	}
	item.stream = newServerStream(c.ctx, "monitor:"+strconv.FormatInt(time.Now().UnixNano(), 10), 300, 1*time.Second)
	item.cmd = client.Monitor(c.ctx, item.ch)
	item.cmd.Start()
	c.items[server] = item

	go c.processMonitor(item.stream, item.ch)
	resp.Success = true
	resp.Data = struct {
		EventName string `json:"eventName"`
	}{
		EventName: item.stream.eventName,
	}
	return
}

func (c *monitorService) processMonitor(stream *serverStream, ch <-chan string) {
	for {
		select {
		case data := <-ch:
			if data != "OK" {
				stream.push(data)
			}
		case <-stream.closeCh:
			// monitor stopped
			return
		}
	}
}

// StopMonitor stop monitor by server name
func (c *monitorService) StopMonitor(server string) (resp types.JSResp) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	item, ok := c.items[server]
	if !ok || item.stream == nil {
		resp.Success = true
		return
	}

	item.stream.stop(func() {
		item.client.Close()
		item.cmd.Stop()
	})
	c.session.close(server)
	delete(c.items, server)
	resp.Success = true
	return
}

// StopAll stop all monitor
func (c *monitorService) StopAll() {
	if c.ctxCancel != nil {
		c.ctxCancel()
	}

	for server := range c.items {
		c.StopMonitor(server)
	}
}

func (c *monitorService) ExportLog(logs []string) (resp types.JSResp) {
	filepath, err := SaveFileDialog(c.ctx, SaveDialogOptions{
		ShowHiddenFiles: false,
		DefaultFilename: fmt.Sprintf("monitor_log_%s.txt", time.Now().Format("20060102150405")),
		Filters: []FileFilter{
			{Pattern: "*.txt"},
		},
	})
	if err != nil {
		resp.Msg = err.Error()
		return
	}

	file, err := os.Create(filepath)
	if err != nil {
		resp.Msg = err.Error()
		return
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, line := range logs {
		_, _ = writer.WriteString(line + "\n")
	}
	writer.Flush()

	resp.Success = true
	return
}
