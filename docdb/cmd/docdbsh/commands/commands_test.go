package commands_test

import (
	"strings"
	"testing"

	"github.com/kartikbazzad/docdb/cmd/docdbsh/commands"
	"github.com/kartikbazzad/docdb/cmd/docdbsh/parser"
)

func TestValidateArgs(t *testing.T) {
	cmd := &parser.Command{
		Name: ".test",
		Args: []string{"arg1", "arg2"},
	}

	if err := parser.ValidateArgs(cmd, 2); err != nil {
		t.Errorf("ValidateArgs(2) should not error, got: %v", err)
	}

	if err := parser.ValidateArgs(cmd, 3); err == nil {
		t.Error("ValidateArgs(3) should error")
	}
}

func TestValidateDB(t *testing.T) {
	if err := parser.ValidateDB(0); err == nil {
		t.Error("ValidateDB(0) should error")
	}

	if err := parser.ValidateDB(1); err != nil {
		t.Errorf("ValidateDB(1) should not error, got: %v", err)
	}
}

func TestParseUint64(t *testing.T) {
	tests := []struct {
		input    string
		expected uint64
		wantErr  bool
	}{
		{"0", 0, false},
		{"1", 1, false},
		{"123", 123, false},
		{"18446744073709551615", 18446744073709551615, false},
		{"-1", 0, true},
		{"abc", 0, true},
	}

	for _, tt := range tests {
		result, err := parser.ParseUint64(tt.input)
		if tt.wantErr {
			if err == nil {
				t.Errorf("ParseUint64(%q) should error", tt.input)
			}
		} else {
			if err != nil {
				t.Errorf("ParseUint64(%q) error: %v", tt.input, err)
			}
			if result != tt.expected {
				t.Errorf("ParseUint64(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		}
	}
}

func TestDecodePayload(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:    "raw string",
			input:   `raw:"Hello"`,
			want:    "",
			wantErr: true,
		},
		{
			name:    "raw unquoted",
			input:   `raw:Hello`,
			want:    "",
			wantErr: true,
		},
		{
			name:    "hex valid",
			input:   `hex:48656c6c6f`,
			want:    "",
			wantErr: true,
		},
		{
			name:    "hex invalid",
			input:   `hex:xyz`,
			want:    "",
			wantErr: true,
		},
		{
			name:    "hex odd length",
			input:   `hex:48656`,
			want:    "",
			wantErr: true,
		},
		{
			name:    "json valid",
			input:   `json:{"key":"value"}`,
			want:    `{"key":"value"}`,
			wantErr: false,
		},
		{
			name:    "json invalid",
			input:   `json:{invalid}`,
			want:    "",
			wantErr: true,
		},
		{
			name:    "missing prefix",
			input:   `Hello`,
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.DecodePayload(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("DecodePayload(%q) should error", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("DecodePayload(%q) error: %v", tt.input, err)
				}
				if string(result) != tt.want {
					t.Errorf("DecodePayload(%q) = %q, want %q", tt.input, result, tt.want)
				}
			}
		})
	}
}

func TestParseCommand(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantCmd  string
		wantArgs []string
		wantErr  bool
	}{
		{
			name:     "simple command",
			input:    ".help",
			wantCmd:  ".help",
			wantArgs: []string{},
			wantErr:  false,
		},
		{
			name:     "command with args",
			input:    ".open testdb",
			wantCmd:  ".open",
			wantArgs: []string{"testdb"},
			wantErr:  false,
		},
		{
			name:     "command with multiple args",
			input:    ".create 1 raw:Hello",
			wantCmd:  ".create",
			wantArgs: []string{"1", "raw:Hello"},
			wantErr:  false,
		},
		{
			name:     "missing dot prefix",
			input:    "help",
			wantCmd:  "",
			wantArgs: nil,
			wantErr:  true,
		},
		{
			name:     "empty command",
			input:    "",
			wantCmd:  "",
			wantArgs: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.Parse(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Parse(%q) should error", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("Parse(%q) error: %v", tt.input, err)
				}
				if result.Name != tt.wantCmd {
					t.Errorf("Parse(%q) = %v, want %v", tt.input, result.Name, tt.wantCmd)
				}
				if len(result.Args) != len(tt.wantArgs) {
					t.Errorf("Parse(%q) args = %v, want %v", tt.input, result.Args, tt.wantArgs)
				}
			}
		})
	}
}

func TestErrorResult(t *testing.T) {
	var sb strings.Builder
	result := commands.ErrorResult{Err: "test error"}
	result.Print(&sb)

	output := sb.String()
	if !strings.Contains(output, "ERROR") {
		t.Error("ErrorResult should contain ERROR")
	}
	if !strings.Contains(output, "test error") {
		t.Error("ErrorResult should contain error message")
	}
	if result.IsExit() {
		t.Error("ErrorResult.IsExit() should be false")
	}
}

func TestOKResult(t *testing.T) {
	var sb strings.Builder
	result := commands.OKResult{}
	result.Print(&sb)

	output := sb.String()
	if !strings.Contains(output, "OK") {
		t.Error("OKResult should contain OK")
	}
	if result.IsExit() {
		t.Error("OKResult.IsExit() should be false")
	}
}

func TestExitResult(t *testing.T) {
	result := commands.ExitResult{}
	if !result.IsExit() {
		t.Error("ExitResult.IsExit() should be true")
	}
}

func TestHelpResult(t *testing.T) {
	var sb strings.Builder
	result := commands.HelpResult{}
	result.Print(&sb)

	output := sb.String()
	if !strings.Contains(output, "DocDB Shell Commands") {
		t.Error("HelpResult should contain header")
	}
	if !strings.Contains(output, ".help") {
		t.Error("HelpResult should contain .help")
	}
}
