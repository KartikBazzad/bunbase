package catalog

import (
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/kartikbazzad/docdb/internal/logger"
	"github.com/kartikbazzad/docdb/internal/types"
)

var (
	ErrCatalogLoad   = errors.New("failed to load catalog")
	ErrCatalogWrite  = errors.New("failed to write catalog")
	ErrDBExists      = errors.New("database already exists")
	ErrDBNotFound    = errors.New("database not found")
	ErrInvalidDBName = errors.New("invalid database name")
)

const (
	DBIDSize    = 8
	NameLenSize = 2
	StatusSize  = 1
	EntryHeader = DBIDSize + NameLenSize + StatusSize
)

type Catalog struct {
	mu       sync.RWMutex
	file     *os.File
	path     string
	entries  map[uint64]*types.CatalogEntry
	names    map[string]uint64
	nextDBID uint64
	logger   *logger.Logger
}

func NewCatalog(path string, log *logger.Logger) *Catalog {
	return &Catalog{
		path:    path,
		entries: make(map[uint64]*types.CatalogEntry),
		names:   make(map[string]uint64),
		logger:  log,
	}
}

func (c *Catalog) Load() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(c.path), 0755); err != nil {
		return err
	}

	file, err := os.OpenFile(c.path, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return ErrCatalogLoad
	}

	c.file = file

	info, err := file.Stat()
	if err != nil {
		return ErrCatalogLoad
	}

	if info.Size() == 0 {
		c.nextDBID = 1
		return nil
	}

	data, err := os.ReadFile(c.path)
	if err != nil {
		return ErrCatalogLoad
	}

	offset := 0
	c.nextDBID = 1

	for offset < len(data) {
		if offset+EntryHeader > len(data) {
			break
		}

		dbID := binary.LittleEndian.Uint64(data[offset : offset+DBIDSize])
		offset += DBIDSize

		nameLen := binary.LittleEndian.Uint16(data[offset : offset+NameLenSize])
		offset += NameLenSize

		status := types.DBStatus(data[offset])
		offset += StatusSize

		if offset+int(nameLen) > len(data) {
			break
		}

		name := string(data[offset : offset+int(nameLen)])
		offset += int(nameLen)

		entry := &types.CatalogEntry{
			DBID:      dbID,
			DBName:    name,
			CreatedAt: time.Now(),
			Status:    status,
		}

		c.entries[dbID] = entry
		c.names[name] = dbID

		if dbID >= c.nextDBID {
			c.nextDBID = dbID + 1
		}
	}

	c.logger.Info("Catalog loaded: %d databases", len(c.entries))
	return nil
}

func (c *Catalog) Create(name string) (uint64, error) {
	if err := ValidateDBName(name); err != nil {
		return 0, ErrInvalidDBName
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.names[name]; exists {
		return 0, ErrDBExists
	}

	dbID := c.nextDBID
	c.nextDBID++

	entry := &types.CatalogEntry{
		DBID:      dbID,
		DBName:    name,
		CreatedAt: time.Now(),
		Status:    types.DBActive,
	}

	if err := c.writeEntry(entry); err != nil {
		c.nextDBID--
		return 0, err
	}

	c.entries[dbID] = entry
	c.names[name] = dbID

	c.logger.Info("Created database: %s (id=%d)", name, dbID)
	return dbID, nil
}

func (c *Catalog) Delete(dbID uint64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, exists := c.entries[dbID]
	if !exists {
		return ErrDBNotFound
	}

	entry.Status = types.DBDeleted
	delete(c.names, entry.DBName)

	if err := c.writeEntry(entry); err != nil {
		entry.Status = types.DBActive
		c.names[entry.DBName] = dbID
		return err
	}

	c.logger.Info("Deleted database: %s (id=%d)", entry.DBName, dbID)
	return nil
}

func (c *Catalog) Get(dbID uint64) (*types.CatalogEntry, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[dbID]
	if !exists {
		return nil, ErrDBNotFound
	}

	return entry, nil
}

func (c *Catalog) GetByName(name string) (uint64, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	dbID, exists := c.names[name]
	if !exists {
		return 0, ErrDBNotFound
	}

	entry, exists := c.entries[dbID]
	if !exists || entry.Status != types.DBActive {
		return 0, ErrDBNotFound
	}

	return dbID, nil
}

func (c *Catalog) List() []*types.CatalogEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()

	list := make([]*types.CatalogEntry, 0, len(c.entries))
	for _, entry := range c.entries {
		if entry.Status == types.DBActive {
			list = append(list, entry)
		}
	}

	return list
}

func (c *Catalog) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.file != nil {
		return c.file.Close()
	}

	return nil
}

func (c *Catalog) writeEntry(entry *types.CatalogEntry) error {
	buf := make([]byte, EntryHeader+len(entry.DBName))

	offset := 0
	binary.LittleEndian.PutUint64(buf[offset:], entry.DBID)
	offset += DBIDSize

	binary.LittleEndian.PutUint16(buf[offset:], uint16(len(entry.DBName)))
	offset += NameLenSize

	buf[offset] = byte(entry.Status)
	offset += StatusSize

	copy(buf[offset:], entry.DBName)

	if _, err := c.file.Write(buf); err != nil {
		return ErrCatalogWrite
	}

	return nil
}
