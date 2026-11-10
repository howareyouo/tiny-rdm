package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"tinyrdm/backend/types"
	sliceutil "tinyrdm/backend/utils/slice"
	strutil "tinyrdm/backend/utils/string"

	"github.com/redis/go-redis/v9"
)

type cliService struct {
	ctx        context.Context
	ctxCancel  context.CancelFunc
	session    *redisSession
	mutex      sync.Mutex
	selectedDB map[string]int
}

type cliOutput struct {
	Null    bool     `json:"null,omitempty"`    // output content is null
	Content []string `json:"content,omitempty"` // output content
	Prompt  string   `json:"prompt,omitempty"`  // new line prompt, empty if not ready to input
}

var cli *cliService
var onceCli sync.Once

func Cli() *cliService {
	if cli == nil {
		onceCli.Do(func() {
			cli = &cliService{
				session:    newRedisSession(nil),
				selectedDB: map[string]int{},
			}
		})
	}
	return cli
}

func (c *cliService) runCommand(server, data string) {
	if cmds := strutil.SplitCmd(data); len(cmds) > 0 && len(cmds[0]) > 0 {
		if client, err := c.getRedisClient(server); err == nil {
			args := sliceutil.Map(cmds, func(i int) any {
				return cmds[i]
			})
			if result, err := client.Do(c.ctx, args...).Result(); err == nil || errors.Is(err, redis.Nil) {
				if strings.ToLower(cmds[0]) == "select" {
					// switch database
					if db, ok := strutil.AnyToInt(cmds[1]); ok {
						c.selectedDB[server] = db
					}
				}

				c.echo(server, result, true)
			} else {
				c.echoError(server, err.Error())
			}
			return
		}
	}

	c.echoReady(server)
}

func (c *cliService) echo(server string, data any, newLineReady bool) {
	var output cliOutput
	if data != nil {
		str := strutil.AnyToString(data, "", 0)
		output.Content = strings.Split(str, "\n")
	}
	if newLineReady {
		output.Prompt = fmt.Sprintf("%s:db%d> ", server, c.selectedDB[server])
	}
	EventsEmit(c.ctx, "cmd:output:"+server, output)
}

func (c *cliService) echoReady(server string) {
	c.echo(server, "", true)
}

func (c *cliService) echoError(server, data string) {
	c.echo(server, "\x1b[31m"+data+"\x1b[0m", true)
}

func (c *cliService) getRedisClient(server string) (redis.UniversalClient, error) {
	item, err := c.session.get(server, -1)
	if err != nil {
		return nil, err
	}
	return item.client, nil
}

func (c *cliService) Start(ctx context.Context) {
	c.ctx, c.ctxCancel = context.WithCancel(ctx)
	c.session.Start(ctx)
}

// StartCli start a cli session
func (c *cliService) StartCli(server string, db int) (resp types.JSResp) {
	client, err := c.getRedisClient(server)
	if err != nil {
		resp.Msg = err.Error()
		return
	}
	client.Do(c.ctx, "select", db)
	c.selectedDB[server] = db

	// monitor input
	EventsOn(c.ctx, "cmd:input:"+server, func(data ...interface{}) {
		if len(data) > 0 {
			if str, ok := data[0].(string); ok {
				c.runCommand(server, str)
				return
			}
		}
		c.echoReady(server)
	})

	// echo prefix
	c.echoReady(server)
	resp.Success = true
	return
}

// CloseCli close cli session
func (c *cliService) CloseCli(server string) (resp types.JSResp) {
	c.session.close(server)
	c.mutex.Lock()
	delete(c.selectedDB, server)
	c.mutex.Unlock()
	EventsOff(c.ctx, "cmd:input:"+server)
	resp.Success = true
	return
}

// CloseAll close all cli sessions
func (c *cliService) CloseAll() {
	if c.ctxCancel != nil {
		c.ctxCancel()
	}

	for server := range c.selectedDB {
		c.CloseCli(server)
	}
	c.session.closeAll()
}
