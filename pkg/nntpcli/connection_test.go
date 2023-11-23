package nntpcli

import (
	"bufio"
	"errors"
	"fmt"
	"strings"
	"testing"
)

type mockConn struct {
	read  string
	write string
	err   error
}

func (m *mockConn) Read(b []byte) (int, error) {
	if m.err != nil {
		return 0, m.err
	}
	copy(b, m.read)
	return len(m.read), nil
}

func (m *mockConn) Write(b []byte) (int, error) {
	if m.err != nil {
		return 0, m.err
	}
	if !strings.Contains(m.write, string(b)) {
		return 0, errors.New("unexpected write")
	}
	return len(b), nil
}

func (m *mockConn) Close() error {
	return nil
}

func TestConn_cmd(t *testing.T) {
	tests := []struct {
		name       string
		expectCode uint
		format     string
		args       []interface{}
		read       string
		write      string
		err        error
		wantCode   uint
		wantLine   string
		wantErr    error
		close      bool
	}{
		{
			name:       "successful cmd",
			expectCode: 200,
			format:     "TEST %d",
			args:       []interface{}{123},
			read:       "200 Test response\r\n",
			write:      "TEST 123\r\n",
			wantCode:   200,
			wantLine:   "Test response",
			wantErr:    nil,
			close:      false,
		},
		{
			name:       "connection closed",
			expectCode: 200,
			format:     "TEST %d",
			args:       []interface{}{123},
			read:       "",
			write:      "",
			err:        fmt.Errorf("connection closed"),
			wantCode:   0,
			wantLine:   "",
			close:      true,
			wantErr:    ProtocolError("connection closed"),
		},
		{
			name:       "short response",
			expectCode: 200,
			format:     "TEST %d",
			args:       []interface{}{123},
			read:       "123\r\n",
			write:      "TEST 123\r\n",
			wantCode:   0,
			wantLine:   "",
			wantErr:    ProtocolError("short response: 123"),
			close:      false,
		},
		{
			name:       "invalid response code",
			expectCode: 200,
			format:     "TEST %d",
			args:       []interface{}{123},
			read:       "ABC Test response\r\n",
			write:      "TEST 123\r\n",
			wantCode:   0,
			wantLine:   "",
			wantErr:    ProtocolError("invalid response code: ABC Test response"),
			close:      false,
		},
		{
			name:       "unexpected response code",
			expectCode: 200,
			format:     "TEST %d",
			args:       []interface{}{123},
			read:       "201 Test response\r\n",
			write:      "TEST 123\r\n",
			wantCode:   201,
			wantLine:   "Test response",
			wantErr:    NntpError{201, "Test response"},
			close:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := &conn{
				conn: &mockConn{
					read:  tt.read,
					write: tt.write,
					err:   tt.err,
				},
				r:     bufio.NewReader(strings.NewReader(tt.read)),
				br:    nil,
				close: tt.close,
			}

			code, line, err := conn.cmd(tt.expectCode, tt.format, tt.args...)
			if err != nil && !errors.Is(err, tt.wantErr) {
				t.Errorf("unexpected error: %v, want: %v", err, tt.wantErr)
			}
			if code != tt.wantCode {
				t.Errorf("unexpected code: got %d, want %d", code, tt.wantCode)
			}
			if line != tt.wantLine {
				t.Errorf("unexpected line: got %q, want %q", line, tt.wantLine)
			}
		})
	}
}

func TestConn_Post(t *testing.T) {
	tests := []struct {
		name      string
		chunkSize int64
		p         []byte
		read      string
		write     string
		err       error
		wantErr   error
		close     bool
	}{
		{
			name:      "successful post",
			chunkSize: 5,
			p:         []byte("hello world"),
			read:      "335 Send article to be transferred\r\n240 Test response\r\n",
			write:     "POST\r\nhello \r\n world\r\n.\r\n",
			err:       nil,
			wantErr:   nil,
		},
		{
			name:      "connection closed",
			chunkSize: 5,
			p:         []byte("hello world"),
			read:      "",
			write:     "",
			err:       fmt.Errorf("connection closed"),
			wantErr:   ProtocolError("connection closed"),
			close:     true,
		},
		{
			name:      "write error",
			chunkSize: 5,
			p:         []byte("hello world"),
			read:      "",
			write:     "",
			err:       fmt.Errorf("write error"),
			wantErr:   fmt.Errorf("write error"),
			close:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := &conn{
				conn: &mockConn{
					read:  tt.read,
					write: tt.write,
					err:   tt.err,
				},
				r:     bufio.NewReader(strings.NewReader(tt.read)),
				br:    nil,
				close: tt.close,
			}

			err := conn.Post(tt.p, tt.chunkSize)
			if err != nil && err.Error() != tt.wantErr.Error() {
				t.Errorf("unexpected error: %v, want: %v", err, tt.wantErr)
			}
		})
	}
}
