package services

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"tinyrdm/backend/consts"
	"tinyrdm/backend/types"
	redis2 "tinyrdm/backend/utils/redis"

	"github.com/redis/go-redis/v9"
)

// sessionItem is a live redis connection bound to one server and one database
type sessionItem struct {
	name       string
	client     redis.UniversalClient
	ctx        context.Context
	cancelFunc context.CancelFunc
	db         int   // current database index
	stepSize   int64 // load step size
}

// redisSession manages the lifetime of redis connections
// each consumer (browser, cli, monitor, pubsub) holds its own session so closing a connection in one consumer does not affect the others
type redisSession struct {
	mutex sync.Mutex
	ctx   context.Context
	items map[string]*sessionItem
	onCmd func(server, cmd string, cost int64)
}

func newRedisSession(onCmd func(server, cmd string, cost int64)) *redisSession {
	return &redisSession{
		items: map[string]*sessionItem{},
		onCmd: onCmd,
	}
}

func (s *redisSession) Start(ctx context.Context) {
	s.ctx = ctx
}

// dialRedisClient creates a redis client and registers the command hook
func dialRedisClient(ctx context.Context, conf types.ConnectionConfig, onCmd func(server, cmd string, cost int64)) (client redis.UniversalClient, err error) {
	client, err = Connection().createRedisClient(conf)
	if err != nil {
		err = fmt.Errorf("create connection error: %s", err.Error())
		return
	}

	if onCmd != nil {
		hook := redis2.NewHook(conf.Name, func(cmd string, cost int64) {
			onCmd(conf.Name, cmd, cost)
		})
		_ = client.Do(ctx, "CLIENT", "SETNAME", url.QueryEscape(conf.Name)).Err()
		// add hook to each node in cluster mode
		if cluster, ok := client.(*redis.ClusterClient); ok {
			if err = cluster.ForEachShard(ctx, func(ctx context.Context, cli *redis.Client) error {
				cli.AddHook(hook)
				return nil
			}); err != nil {
				client.Close()
				err = fmt.Errorf("get cluster nodes error: %s", err.Error())
				return
			}
		} else {
			client.AddHook(hook)
		}
	}

	if _, err = client.Ping(ctx).Result(); err != nil && !errors.Is(err, redis.Nil) {
		client.Close()
		err = errors.New("can not connect to redis server:" + err.Error())
		return
	}
	return
}

// get returns a connected session for the server, dialing a new one when needed
// if db >= 0, the session is (re)dialed targeting that database index
func (s *redisSession) get(server string, db int) (*sessionItem, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if item, ok := s.items[server]; ok {
		if db < 0 || item.db == db {
			// return without switch database directly
			return item, nil
		}
		// close previous connection if database is not the same
		s.removeItemLocked(item)
	}

	conf := Connection().getConnection(server)
	if conf == nil {
		return nil, fmt.Errorf("no match connection \"%s\"", server)
	}

	ctx, cancelFunc := context.WithCancel(s.ctx)
	var connConfig = conf.ConnectionConfig
	connConfig.LastDB = db
	client, err := dialRedisClient(ctx, connConfig, s.onCmd)
	if err != nil {
		cancelFunc()
		return nil, err
	}
	item := &sessionItem{
		name:       server,
		client:     client,
		ctx:        ctx,
		cancelFunc: cancelFunc,
		stepSize:   int64(conf.LoadSize),
		db:         db,
	}
	if item.stepSize <= 0 {
		item.stepSize = consts.DEFAULT_LOAD_SIZE
	}
	s.items[server] = item
	return item, nil
}

// getWithDBFallback behaves like get, but retries with database 0 when the target database index does not exist
func (s *redisSession) getWithDBFallback(server string, db int) (item *sessionItem, selectedDB int, err error) {
	selectedDB = db
	item, err = s.get(server, db)
	if err != nil {
		if strings.Contains(err.Error(), "DB index is out of range") && db != 0 {
			if item, err = s.get(server, 0); err != nil {
				item = nil
			} else {
				selectedDB = 0
			}
		}
	}
	return
}

// close a session of the server
func (s *redisSession) close(server string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if item, ok := s.items[server]; ok {
		s.removeItemLocked(item)
	}
}

// closeAll closes all sessions
func (s *redisSession) closeAll() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	for _, item := range s.items {
		s.removeItemLocked(item)
	}
}

// removeItemLocked closes the connection of a session, the session mutex must be held
func (s *redisSession) removeItemLocked(item *sessionItem) {
	if item.cancelFunc != nil {
		item.cancelFunc()
	}
	if item.client != nil {
		item.client.Close()
	}
	delete(s.items, item.name)
}
