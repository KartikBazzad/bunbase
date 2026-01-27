package parser

import (
	"fmt"
	"strconv"
	"strings"
)

type Command struct {
	Name string
	Args []string
	Line string
}

func Parse(line string) (*Command, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, fmt.Errorf("empty command")
	}

	parts := strings.Fields(line)
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty command")
	}

	if !strings.HasPrefix(parts[0], ".") {
		return nil, fmt.Errorf("commands must start with '.'")
	}

	cmd := &Command{
		Name: parts[0],
		Args: parts[1:],
		Line: line,
	}

	return cmd, nil
}

func ParseUint64(s string) (uint64, error) {
	return strconv.ParseUint(s, 10, 64)
}

func ValidateArgs(cmd *Command, count int) error {
	if len(cmd.Args) < count {
		return fmt.Errorf("expected %d argument(s), got %d", count, len(cmd.Args))
	}
	return nil
}

func ValidateDB(dbID uint64) error {
	if dbID == 0 {
		return fmt.Errorf("no database open")
	}
	return nil
}
