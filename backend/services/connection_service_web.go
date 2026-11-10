//go:build web

package services

import (
	"tinyrdm/backend/types"
)

// ExportConnectionsToBytes exports connections as zip bytes for web download
func (c *connectionService) ExportConnectionsToBytes() ([]byte, string, error) {
	data, err := c.exportConnectionsArchive()
	if err != nil {
		return nil, "", err
	}
	return data, connectionsArchiveName(), nil
}

// ImportConnectionsFromBytes imports connections from uploaded zip bytes
func (c *connectionService) ImportConnectionsFromBytes(data []byte) (resp types.JSResp) {
	if err := c.importConnectionsArchive(data); err != nil {
		resp.Msg = err.Error()
		return
	}
	resp.Success = true
	return
}