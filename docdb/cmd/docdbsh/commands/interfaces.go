package commands

import (
	"github.com/kartikbazzad/docdb/internal/ipc"
	"github.com/kartikbazzad/docdb/internal/types"
)

type Client interface {
	OpenDB(name string) (uint64, error)
	CloseDB(dbID uint64) error
	Execute(dbID uint64, ops []ipc.Operation) ([]byte, error)
	Stats() (*types.Stats, error)
	CreateCollection(dbID uint64, name string) error
	DeleteCollection(dbID uint64, name string) error
	ListCollections(dbID uint64) ([]string, error)
	ListDBs() ([]*types.DBInfo, error)
}

type Shell interface {
	GetDB() uint64
	SetDB(uint64)
	ClearDB()
	GetClient() Client
	GetDBName() string
	SetDBName(name string)
	GetPretty() bool
	SetPretty(pretty bool)
	GetHistory() []string
	GetCollection() string
	SetCollection(collection string)
}
