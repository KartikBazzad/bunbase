package shell

import (
	"fmt"
	"sync"

	"github.com/kartikbazzad/docdb/cmd/docdbsh/client"
	"github.com/kartikbazzad/docdb/cmd/docdbsh/commands"
	"github.com/kartikbazzad/docdb/cmd/docdbsh/parser"
)

type Shell struct {
	socketPath        string
	dbID              uint64
	dbName            string
	currentCollection string // v0.2: current collection context
	txActive          bool
	pretty            bool
	history           []string
	client            *client.Client
	mu                sync.Mutex
}

func NewShell(socketPath string) (*Shell, error) {
	c := client.New(socketPath)
	return &Shell{
		socketPath:        socketPath,
		client:            c,
		dbID:              0,
		dbName:            "",
		currentCollection: "_default",
		txActive:          false,
		pretty:            false,
		history:           make([]string, 0, 100),
	}, nil
}

func (s *Shell) Connect() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.client.Connect()
}

func (s *Shell) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.client.Close()
}

func (s *Shell) SetDB(dbID uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dbID = dbID
}

func (s *Shell) GetDB() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.dbID
}

func (s *Shell) ClearDB() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dbID = 0
	s.dbName = ""
	s.currentCollection = "_default"
	s.txActive = false
}

func (s *Shell) SetCollection(collection string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if collection == "" {
		s.currentCollection = "_default"
	} else {
		s.currentCollection = collection
	}
}

func (s *Shell) GetCollection() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.currentCollection
}

func (s *Shell) BeginTx() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.txActive {
		return fmt.Errorf("transaction already active")
	}
	s.txActive = true
	return nil
}

func (s *Shell) CommitTx() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.txActive {
		return fmt.Errorf("no active transaction")
	}
	s.txActive = false
	return nil
}

func (s *Shell) RollbackTx() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.txActive {
		return fmt.Errorf("no active transaction")
	}
	s.txActive = false
	return nil
}

func (s *Shell) IsTxActive() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.txActive
}

func (s *Shell) SetDBName(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dbName = name
}

func (s *Shell) GetDBName() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.dbName
}

func (s *Shell) SetPretty(pretty bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pretty = pretty
}

func (s *Shell) GetPretty() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pretty
}

func (s *Shell) AddToHistory(cmd string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.history = append(s.history, cmd)
	if len(s.history) > 100 {
		s.history = s.history[1:]
	}
}

func (s *Shell) GetHistory() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	hist := make([]string, len(s.history))
	copy(hist, s.history)
	return hist
}

func (s *Shell) Execute(cmd *parser.Command) commands.Result {
	switch cmd.Name {
	case ".help":
		return commands.Help()
	case ".exit":
		return commands.Exit()
	case ".clear":
		return commands.Clear(s)
	case ".open", ".use":
		return commands.Open(s, cmd)
	case ".close":
		return commands.Close(s)
	case ".ls":
		return commands.ListDBs(s)
	case ".pwd":
		return commands.PWD(s)
	case ".pretty":
		return commands.Pretty(s, cmd)
	case ".history":
		return commands.History(s)
	case ".create":
		return commands.Create(s, cmd)
	case ".read":
		return commands.Read(s, cmd)
	case ".update":
		return commands.Update(s, cmd)
	case ".delete":
		return commands.Delete(s, cmd)
	case ".stats":
		return commands.Stats(s)
	case ".mem":
		return commands.Mem(s)
	case ".wal":
		return commands.WAL(s)
	case ".heal":
		return commands.Heal(s, cmd)
	case ".heal-all":
		return commands.HealAll(s)
	case ".heal-stats":
		return commands.HealStats(s)
	case ".use":
		return commands.UseCollection(s, cmd)
	case ".collections":
		return commands.ListCollections(s)
	case ".create-collection":
		return commands.CreateCollection(s, cmd)
	case ".drop-collection":
		return commands.DropCollection(s, cmd)
	case ".patch":
		return commands.Patch(s, cmd)
	default:
		return commands.ErrorResult{Err: fmt.Sprintf("unknown command: %s", cmd.Name)}
	}
}

func (s *Shell) GetClient() commands.Client {
	return s.client
}
