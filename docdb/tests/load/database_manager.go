package load

import (
	"fmt"
	"sync"

	"github.com/kartikbazzad/docdb/pkg/client"
)

// DatabaseContext holds per-database state.
type DatabaseContext struct {
	Name           string
	DBID           uint64
	Config         *DatabaseConfig
	Client         *client.Client
	LatencyMetrics *LatencyMetrics
	WALTracker     *WALTracker
	HealingTracker *HealingTracker
	Payloads       [][]byte
	mu             sync.RWMutex
}

// DatabaseManager manages multiple databases and their metrics.
type DatabaseManager struct {
	databases  map[string]*DatabaseContext
	clients    map[string]*client.Client
	mu         sync.RWMutex
	baseClient *client.Client
	socketPath string // Store socket path for creating new clients
}

// NewDatabaseManager creates a new database manager.
func NewDatabaseManager(baseClient *client.Client, socketPath string) *DatabaseManager {
	return &DatabaseManager{
		databases:  make(map[string]*DatabaseContext),
		clients:    make(map[string]*client.Client),
		baseClient: baseClient,
		socketPath: socketPath,
	}
}

// AddDatabase adds a database to the manager.
func (dm *DatabaseManager) AddDatabase(config DatabaseConfig, healingClient HealingStatsClient, walDir string) (*DatabaseContext, error) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if _, exists := dm.databases[config.Name]; exists {
		return nil, fmt.Errorf("database %s already exists", config.Name)
	}

	// Open database
	dbID, err := dm.baseClient.OpenDB(config.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to open database %s: %w", config.Name, err)
	}

	// Create per-database client
	dbClient := client.New(dm.socketPath)
	if err := dbClient.Connect(); err != nil {
		dm.baseClient.CloseDB(dbID)
		return nil, fmt.Errorf("failed to connect client for %s: %w", config.Name, err)
	}

	// Determine WAL directory
	walDirToUse := config.WALDir
	if walDirToUse == "" {
		walDirToUse = walDir
	}

	// Create context
	ctx := &DatabaseContext{
		Name:           config.Name,
		DBID:           dbID,
		Config:         &config,
		Client:         dbClient,
		LatencyMetrics: NewLatencyMetrics(),
		WALTracker:     NewWALTracker(walDirToUse, config.Name),
		HealingTracker: NewHealingTracker(healingClient, dbID),
		Payloads:       nil, // Will be generated later
	}

	dm.databases[config.Name] = ctx
	dm.clients[config.Name] = dbClient

	return ctx, nil
}

// GetDatabase returns a database context by name.
func (dm *DatabaseManager) GetDatabase(name string) (*DatabaseContext, error) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	ctx, exists := dm.databases[name]
	if !exists {
		return nil, fmt.Errorf("database %s not found", name)
	}
	return ctx, nil
}

// GetAllDatabases returns all database contexts.
func (dm *DatabaseManager) GetAllDatabases() []*DatabaseContext {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	contexts := make([]*DatabaseContext, 0, len(dm.databases))
	for _, ctx := range dm.databases {
		contexts = append(contexts, ctx)
	}
	return contexts
}

// CloseAll closes all databases and cleans up resources.
func (dm *DatabaseManager) CloseAll() error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	var firstErr error
	for name, ctx := range dm.databases {
		if ctx.Client != nil {
			if err := ctx.Client.CloseDB(ctx.DBID); err != nil && firstErr == nil {
				firstErr = fmt.Errorf("failed to close database %s: %w", name, err)
			}
			ctx.Client.Close()
		}
	}

	dm.databases = make(map[string]*DatabaseContext)
	dm.clients = make(map[string]*client.Client)

	return firstErr
}

// StartHealingTracking starts healing tracking for all databases.
func (dm *DatabaseManager) StartHealingTracking() error {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	for _, ctx := range dm.databases {
		if err := ctx.HealingTracker.Start(); err != nil {
			return fmt.Errorf("failed to start healing tracking for %s: %w", ctx.Name, err)
		}
	}
	return nil
}

// StopHealingTracking stops healing tracking for all databases.
func (dm *DatabaseManager) StopHealingTracking() error {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	var firstErr error
	for _, ctx := range dm.databases {
		if err := ctx.HealingTracker.Stop(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("failed to stop healing tracking for %s: %w", ctx.Name, err)
		}
	}
	return firstErr
}

// SampleWAL samples WAL size for all databases.
func (dm *DatabaseManager) SampleWAL() error {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	var firstErr error
	for _, ctx := range dm.databases {
		if err := ctx.WALTracker.Sample(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("failed to sample WAL for %s: %w", ctx.Name, err)
		}
	}
	return firstErr
}
