package parser

import (
	"bytes"
	"testing"
)

// Helper function to create a reader from a string
func newReader(s string) *bytes.Buffer {
	return bytes.NewBufferString(s)
}

func TestParsePUB(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		expectedCmd *Cmd
	}{
		{
			name:        "Valid PUB command",
			input:       "PUB foo 5\r\nhello\r\n",
			expectError: false,
			expectedCmd: &Cmd{
				Name:    PUB,
				Subject: "foo",
				Bytes:   []byte("hello"),
			},
		},
		{
			name:        "Valid PUB with empty message",
			input:       "PUB foo 0\r\n\r\n",
			expectError: false,
			expectedCmd: &Cmd{
				Name:    PUB,
				Subject: "foo",
				Bytes:   []byte(""),
			},
		},
		{
			name:        "PUB command with missing message",
			input:       "PUB foo 5\r\nhello",
			expectError: true,
		},
		{
			name:        "PUB command with invalid length",
			input:       "PUB foo five\r\nhello\r\n",
			expectError: true,
		},
		{
			name:        "PUB command with message exceeding max size",
			input:       "PUB foo 1048577\r\n" + string(make([]byte, 1048577)) + "\r\n",
			expectError: true,
		},
		{
			name:        "PUB command with negative length",
			input:       "PUB foo -5\r\nhello\r\n",
			expectError: true,
		},
		{
			name:        "PUB command with missing arguments",
			input:       "PUB foo\r\n",
			expectError: true,
		},
		{
			name:        "PUB command with extra arguments",
			input:       "PUB foo 5 exta\r\nhello\r\n",
			expectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reader := newReader(test.input)
			cmd, err := Parse(reader)
			if test.expectError {
				if err == nil {
					t.Errorf("Expected error, but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				} else {
					if cmd.Name != test.expectedCmd.Name {
						t.Errorf("Expected command name %s, got %s", test.expectedCmd.Name, cmd.Name)
					}
					if cmd.Subject != test.expectedCmd.Subject {
						t.Errorf("Expected subject %s, got %s", test.expectedCmd.Subject, cmd.Subject)
					}
					if !bytes.Equal(cmd.Bytes, test.expectedCmd.Bytes) {
						t.Errorf("Expected bytes %s, got %s", test.expectedCmd.Bytes, cmd.Bytes)
					}
				}
			}
		})
	}
}

func TestParseSUB(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		expectedCmd *Cmd
	}{
		{
			name:        "Valid SUB command",
			input:       "SUB foo 1\r\n",
			expectError: false,
			expectedCmd: &Cmd{
				Name:    SUB,
				Subject: "foo",
				ID:      "1",
			},
		},
		{
			name:        "SUB command with extra arguments (ignored)",
			input:       "SUB foo 1 extra\r\n",
			expectError: false,
			expectedCmd: &Cmd{
				Name:    SUB,
				Subject: "foo",
				ID:      "1",
				// QueueGroup is ignored for now
			},
		},
		{
			name:        "SUB command with missing arguments",
			input:       "SUB foo\r\n",
			expectError: true,
		},
		{
			name:        "SUB command with no arguments",
			input:       "SUB\r\n",
			expectError: true,
		},
		{
			name:        "SUB command with empty subject",
			input:       "SUB  1\r\n",
			expectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reader := newReader(test.input)
			cmd, err := Parse(reader)
			if test.expectError {
				if err == nil {
					t.Errorf("Expected error, but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				} else {
					if cmd.Name != test.expectedCmd.Name {
						t.Errorf("Expected command name %s, got %s", test.expectedCmd.Name, cmd.Name)
					}
					if cmd.Subject != test.expectedCmd.Subject {
						t.Errorf("Expected subject %s, got %s", test.expectedCmd.Subject, cmd.Subject)
					}
					if cmd.ID != test.expectedCmd.ID {
						t.Errorf("Expected ID %s, got %s", test.expectedCmd.ID, cmd.ID)
					}
				}
			}
		})
	}
}

func TestParseUNSUB(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		expectedCmd *Cmd
	}{
		{
			name:        "Valid UNSUB command",
			input:       "UNSUB foo\r\n",
			expectError: false,
			expectedCmd: &Cmd{
				Name:    UNSUB,
				Subject: "foo",
			},
		},
		{
			name:        "UNSUB command with max messages",
			input:       "UNSUB 1 5\r\n",
			expectError: false,
			expectedCmd: &Cmd{
				Name:                   UNSUB,
				Subject:                "1",
				MaxMessagesBeforeUnsub: 5,
			},
		},
		{
			name:        "UNSUB command with invalid max messages",
			input:       "UNSUB 1 five\r\n",
			expectError: true,
		},
		{
			name:        "UNSUB command with missing arguments",
			input:       "UNSUB\r\n",
			expectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reader := newReader(test.input)
			cmd, err := Parse(reader)
			if test.expectError {
				if err == nil {
					t.Errorf("Expected error, but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				} else {
					if cmd.Name != test.expectedCmd.Name {
						t.Errorf("Expected command name %s, got %s", test.expectedCmd.Name, cmd.Name)
					}
					if cmd.ID != test.expectedCmd.ID {
						t.Errorf("Expected ID %s, got %s", test.expectedCmd.ID, cmd.ID)
					}
					if cmd.MaxMessagesBeforeUnsub != test.expectedCmd.MaxMessagesBeforeUnsub {
						t.Errorf("Expected MaxMessagesBeforeUnsub %d, got %d", test.expectedCmd.MaxMessagesBeforeUnsub, cmd.MaxMessagesBeforeUnsub)
					}
				}
			}
		})
	}
}

func TestParseCONNECT(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		expectedCmd *Cmd
	}{
		{
			name:        "Valid CONNECT command",
			input:       "CONNECT {\"name\":\"client\",\"verbose\":true}\r\n",
			expectError: false,
			expectedCmd: &Cmd{
				Name: CONNECT,
				ConnectData: ConnectCommand{
					Name:    "client",
					Verbose: true,
				},
			},
		},
		{
			name:        "CONNECT command with empty JSON",
			input:       "CONNECT {}\r\n",
			expectError: false,
			expectedCmd: &Cmd{
				Name:        CONNECT,
				ConnectData: ConnectCommand{},
			},
		},
		{
			name:        "CONNECT command with invalid JSON",
			input:       "CONNECT {name:\"client\"}\r\n",
			expectError: true,
		},
		{
			name:        "CONNECT command with missing JSON",
			input:       "CONNECT \r\n",
			expectError: true,
		},
		{
			name:        "CONNECT command with extra arguments",
			input:       "CONNECT {\"name\":\"client\"} extra\r\n",
			expectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reader := newReader(test.input)
			cmd, err := Parse(reader)
			if test.expectError {
				if err == nil {
					t.Errorf("Expected error, but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				} else {
					if cmd.Name != test.expectedCmd.Name {
						t.Errorf("Expected command name %s, got %s", test.expectedCmd.Name, cmd.Name)
					}
					if cmd.ConnectData.Name != test.expectedCmd.ConnectData.Name {
						t.Errorf("Expected ConnectData.Name %s, got %s", test.expectedCmd.ConnectData.Name, cmd.ConnectData.Name)
					}
					if cmd.ConnectData.Verbose != test.expectedCmd.ConnectData.Verbose {
						t.Errorf("Expected ConnectData.Verbose %v, got %v", test.expectedCmd.ConnectData.Verbose, cmd.ConnectData.Verbose)
					}
				}
			}
		})
	}
}

func TestParsePONG(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		expectedCmd *Cmd
	}{

		{
			name:        "PONG command",
			input:       "PONG\r\n",
			expectError: false,
			expectedCmd: &Cmd{
				Name: PONG,
			},
		},

		{
			name:        "PING command without CRLF",
			input:       "PING",
			expectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reader := newReader(test.input)
			cmd, err := Parse(reader)
			if test.expectError {
				if err == nil {
					t.Errorf("Expected error, but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				} else {
					if cmd.Name != test.expectedCmd.Name {
						t.Errorf("Expected command name %s, got %s", test.expectedCmd.Name, cmd.Name)
					}
				}
			}
		})
	}
}

func TestParseUnknownCommand(t *testing.T) {
	input := "UNKNOWN_CMD\r\n"
	reader := newReader(input)
	_, err := Parse(reader)
	if err == nil {
		t.Errorf("Expected error for unknown command, but got nil")
	}
}

func TestParseEmptyCommand(t *testing.T) {
	input := "\r\n"
	reader := newReader(input)
	_, err := Parse(reader)
	if err == nil {
		t.Errorf("Expected error for empty command, but got nil")
	}
}

func TestParseIncompleteCommand(t *testing.T) {
	input := "PUB foo 5\r\nhel"
	reader := newReader(input)
	_, err := Parse(reader)
	if err == nil {
		t.Errorf("Expected error for incomplete command, but got nil")
	}
}
