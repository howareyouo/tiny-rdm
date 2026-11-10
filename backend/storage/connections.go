package storage

import (
	"errors"
	"slices"
	"sync"
	"tinyrdm/backend/consts"
	"tinyrdm/backend/types"

	"gopkg.in/yaml.v3"
)

const CONNECTIONS_PROFILE = "connections.yaml"

type ConnectionsStorage struct {
	storage *localStorage
	mutex   sync.Mutex
}

func NewConnections() *ConnectionsStorage {
	return &ConnectionsStorage{
		storage: NewLocalStore(CONNECTIONS_PROFILE),
	}
}

func (c *ConnectionsStorage) defaultConnections() types.Connections {
	return types.Connections{}
}

func (c *ConnectionsStorage) loadConnections() (ret types.Connections) {
	b, err := c.storage.Load()
	ret = c.defaultConnections()
	if err != nil {
		return
	}

	if err = yaml.Unmarshal(b, &ret); err != nil {
		ret = c.defaultConnections()
		return
	}
	if len(ret) <= 0 {
		ret = c.defaultConnections()
	}
	return
}

// LoadProfile returns the raw connections profile saved on disk
func (c *ConnectionsStorage) LoadProfile() ([]byte, error) {
	return c.storage.Load()
}

// StoreProfile replaces the connections profile saved on disk
func (c *ConnectionsStorage) StoreProfile(data []byte) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	return c.storage.Store(data)
}

// resolveConnection fills in the defaults and the group name of a stored connection
func resolveConnection(config types.ConnectionConfig, group string) types.ConnectionConfig {
	config.Group = group
	if len(config.Network) <= 0 {
		config.Network = "tcp"
	}
	if config.Port <= 0 {
		config.Port = 6379
	}
	if len(config.DefaultFilter) <= 0 {
		config.DefaultFilter = "*"
	}
	if len(config.KeySeparator) <= 0 {
		config.KeySeparator = ":"
	}
	if config.ConnTimeout <= 0 {
		config.ConnTimeout = 60
	}
	if config.ExecTimeout <= 0 {
		config.ExecTimeout = 60
	}
	if config.LoadSize <= 0 {
		config.LoadSize = consts.DEFAULT_LOAD_SIZE
	}
	if config.RefreshInterval <= 0 {
		config.RefreshInterval = 5
	}
	if config.DBFilterList == nil {
		config.DBFilterList = []int{}
	}
	if config.DBFilterType != "show" && config.DBFilterType != "hide" {
		config.DBFilterType = "none"
	}
	if len(config.DBFilterList) <= 0 {
		// a filter without databases means no filtering at all
		config.DBFilterType = "none"
	}
	if config.Sentinel.Enable && len(config.Sentinel.Master) <= 0 {
		config.Sentinel.Master = "mymaster"
	}
	config.LastDB = resolveLastDB(config)
	return config
}

// resolveLastDB picks a database index that the db filter allows
func resolveLastDB(config types.ConnectionConfig) int {
	db := max(config.LastDB, 0)
	switch config.DBFilterType {
	case "show":
		if !slices.Contains(config.DBFilterList, db) {
			db = config.DBFilterList[0]
		}
	case "hide":
		if slices.Contains(config.DBFilterList, db) {
			for db = 0; slices.Contains(config.DBFilterList, db); db++ {
			}
		}
	}
	return db
}

// resolveConnections turns the stored tree into connections holding the group they belong to
func resolveConnections(conns types.Connections, group string) types.Connections {
	if conns == nil {
		return nil
	}

	ret := make(types.Connections, 0, len(conns))
	for _, conn := range conns {
		if conn.Type == "group" {
			groupConfig := conn.ConnectionConfig
			groupConfig.Group = group
			ret = append(ret, types.Connection{
				ConnectionConfig: groupConfig,
				Type:             conn.Type,
				Connections:      resolveConnections(conn.Connections, conn.Name),
			})
		} else {
			ret = append(ret, types.Connection{
				ConnectionConfig: resolveConnection(conn.ConnectionConfig, group),
			})
		}
	}
	return ret
}

// flattenConnections removes the group level of a connection tree
func flattenConnections(conns types.Connections) (ret types.Connections) {
	for _, conn := range conns {
		if conn.Type == "group" {
			ret = append(ret, conn.Connections...)
		} else {
			ret = append(ret, conn)
		}
	}
	return
}

// GetConnections get all store connections from local
func (c *ConnectionsStorage) GetConnections() types.Connections {
	return resolveConnections(c.loadConnections(), "")
}

// GetConnectionsFlat get all store connections from local flat(exclude group level)
func (c *ConnectionsStorage) GetConnectionsFlat() types.Connections {
	return flattenConnections(c.GetConnections())
}

// GetConnection get connection by name
func (c *ConnectionsStorage) GetConnection(name string) *types.Connection {
	var findConn func(types.Connections) *types.Connection
	findConn = func(conns types.Connections) *types.Connection {
		for i, conn := range conns {
			if conn.Type != "group" {
				if conn.Name == name {
					return &conns[i]
				}
			} else if ret := findConn(conn.Connections); ret != nil {
				return ret
			}
		}
		return nil
	}

	return findConn(c.GetConnections())
}

// GetGroup get one connection group by name
func (c *ConnectionsStorage) GetGroup(name string) *types.Connection {
	conns := c.GetConnections()

	for i, conn := range conns {
		if conn.Type == "group" && conn.Name == name {
			return &conns[i]
		}
	}
	return nil
}

func (c *ConnectionsStorage) saveConnections(conns types.Connections) error {
	b, err := yaml.Marshal(&conns)
	if err != nil {
		return err
	}
	if err = c.storage.Store(b); err != nil {
		return err
	}
	return nil
}

// CreateConnection create new connection
func (c *ConnectionsStorage) CreateConnection(param types.ConnectionConfig) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	conn := c.GetConnection(param.Name)
	if conn != nil {
		return errors.New("duplicated connection name")
	}

	conns := c.loadConnections()
	var group *types.Connection
	if len(param.Group) > 0 {
		for i, conn := range conns {
			if conn.Type == "group" && conn.Name == param.Group {
				group = &conns[i]
				break
			}
		}
	}
	if group != nil {
		group.Connections = append(group.Connections, types.Connection{
			ConnectionConfig: param,
		})
	} else {
		if len(param.Group) > 0 {
			// no group matched, create new group
			conns = append(conns, types.Connection{
				Type: "group",
				Connections: types.Connections{
					types.Connection{
						ConnectionConfig: param,
					},
				},
			})
		} else {
			conns = append(conns, types.Connection{
				ConnectionConfig: param,
			})
		}
	}

	return c.saveConnections(conns)
}

// UpdateConnection update existing connection by name
func (c *ConnectionsStorage) UpdateConnection(name string, param types.ConnectionConfig) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	conns := c.loadConnections()
	var updated bool
	var retrieve func(types.Connections, string, types.ConnectionConfig) error
	retrieve = func(conns types.Connections, name string, param types.ConnectionConfig) error {
		for i, conn := range conns {
			if conn.Type != "group" {
				if name != param.Name && conn.Name == param.Name {
					return errors.New("duplicated connection name")
				} else if conn.Name == name && !updated {
					conns[i] = types.Connection{
						ConnectionConfig: param,
					}
					updated = true
				}
			} else {
				if err := retrieve(conn.Connections, name, param); err != nil {
					return err
				}
			}
		}
		return nil
	}

	err := retrieve(conns, name, param)
	if err != nil {
		return err
	}
	if !updated {
		return errors.New("connection not found")
	}

	return c.saveConnections(conns)
}

// DeleteConnection remove special connection
func (c *ConnectionsStorage) DeleteConnection(name string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	conns := c.loadConnections()
	var updated bool
	for i, conn := range conns {
		if conn.Type == "group" {
			for j, subConn := range conn.Connections {
				if subConn.Name == name {
					conns[i].Connections = append(conns[i].Connections[:j], conns[i].Connections[j+1:]...)
					updated = true
					break
				}
			}
		} else if conn.Name == name {
			conns = append(conns[:i], conns[i+1:]...)
			updated = true
			break
		}
		if updated {
			break
		}
	}
	if !updated {
		return errors.New("no match connection")
	}
	return c.saveConnections(conns)
}

// SaveSortedConnection save connection after sort
func (c *ConnectionsStorage) SaveSortedConnection(sortedConns types.Connections) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	conns := flattenConnections(c.loadConnections())
	takeConn := func(name string) (types.Connection, bool) {
		idx := slices.IndexFunc(conns, func(connection types.Connection) bool {
			return connection.Name == name
		})
		if idx >= 0 {
			ret := conns[idx]
			conns = append(conns[:idx], conns[idx+1:]...)
			return ret, true
		}
		return types.Connection{}, false
	}
	var replaceConn func(connections types.Connections) types.Connections
	replaceConn = func(cons types.Connections) types.Connections {
		var newConns types.Connections
		for _, conn := range cons {
			if conn.Type == "group" {
				newConns = append(newConns, types.Connection{
					ConnectionConfig: types.ConnectionConfig{
						Name: conn.Name,
					},
					Type:        "group",
					Connections: replaceConn(conn.Connections),
				})
			} else {
				if foundConn, ok := takeConn(conn.Name); ok {
					newConns = append(newConns, foundConn)
				}
			}
		}
		return newConns
	}
	conns = replaceConn(sortedConns)
	return c.saveConnections(conns)
}

// CreateGroup create a new group
func (c *ConnectionsStorage) CreateGroup(name string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	conns := c.loadConnections()
	for _, conn := range conns {
		if conn.Type == "group" && conn.Name == name {
			return errors.New("duplicated group name")
		}
	}

	conns = append(conns, types.Connection{
		ConnectionConfig: types.ConnectionConfig{
			Name: name,
		},
		Type: "group",
	})
	return c.saveConnections(conns)
}

// RenameGroup rename group
func (c *ConnectionsStorage) RenameGroup(name, newName string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	groupIndex := -1
	conns := c.loadConnections()
	for i, conn := range conns {
		if conn.Type == "group" {
			if conn.Name == newName {
				return errors.New("duplicated group name")
			} else if conn.Name == name {
				groupIndex = i
			}
		}
	}

	if groupIndex == -1 {
		return errors.New("group not found")
	}

	conns[groupIndex].Name = newName
	return c.saveConnections(conns)
}

// DeleteGroup remove specified group, include all connections under it
func (c *ConnectionsStorage) DeleteGroup(group string, includeConnection bool) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	conns := c.loadConnections()
	for i, conn := range conns {
		if conn.Type == "group" && conn.Name == group {
			conns = append(conns[:i], conns[i+1:]...)
			if includeConnection {
				conns = append(conns, conn.Connections...)
			}
			return c.saveConnections(conns)
		}
	}
	return errors.New("group not found")
}
