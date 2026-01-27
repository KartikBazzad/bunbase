package shell

import (
	"fmt"
	"sync"

	"github.com/kartikbazzad/docdb/cmd/docdbsh/client"
	"github.com/kartikbazzad/docdb/cmd/docdbsh/commands"
	"github.com/kartikbazzad/docdb/cmd/docdbsh/parser"
)

type Shell struct {
	socketPath string
	dbID       uint64
	txActive   bool
	client     *client.Client
	mu         sync.Mutex
}

func NewShell(socketPath string) (*Shell, error) {
	c := client.New(socketPath)
	return &Shell{
		socketPath: socketPath,
		client:     c,
		dbID:       0,
		txActive:   false,
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
	s.txActive = false
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

func (s *Shell) Execute(cmd *parser.Command) commands.Result {
	switch cmd.Name {
	case ".help":
		return commands.Help()
	case ".exit":
		return commands.Exit()
	case ".clear":
		return commands.Clear(s)
	case ".open":
		return commands.Open(s, cmd)
	case ".close":
		return commands.Close(s)
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
	default:
		return commands.ErrorResult{Err: fmt.Sprintf("unknown command: %s", cmd.Name)}
	}
}

func (s *Shell) GetClient() commands.Client {
	return s.client
}
