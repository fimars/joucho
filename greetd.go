package main

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
)

// greetdClient speaks greetd's IPC protocol over the socket in
// $GREETD_SOCK: a 32-bit length in native byte order, then that many bytes
// of JSON. Each request is answered by one response. See greetd-ipc(7).
type greetdClient struct {
	conn net.Conn
}

// greetdResponse is one answer from greetd. Only the fields of its Type
// are set.
type greetdResponse struct {
	Type            string `json:"type"`              // auth_message, success, error
	AuthMessageType string `json:"auth_message_type"` // secret, visible, info, error
	AuthMessage     string `json:"auth_message"`
	ErrorType       string `json:"error_type"`
	Description     string `json:"description"`
}

// greetdDial connects to greetd. An error means we were not started by it.
func greetdDial() (*greetdClient, error) {
	path := os.Getenv("GREETD_SOCK")
	if path == "" {
		return nil, errors.New("GREETD_SOCK is not set: not started by greetd (use --demo to try the interface)")
	}
	conn, err := net.Dial("unix", path)
	if err != nil {
		return nil, fmt.Errorf("greetd: %w", err)
	}
	return &greetdClient{conn: conn}, nil
}

// send writes one framed message. Linux on every architecture we run on
// is little-endian, which is greetd's native byte order.
func (c *greetdClient) send(v any) error {
	payload, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if len(payload) > maxMessage {
		return fmt.Errorf("greetd: message of %d bytes is too large", len(payload))
	}
	var header [4]byte
	binary.LittleEndian.PutUint32(header[:], uint32(len(payload)))
	if _, err := c.conn.Write(header[:]); err != nil {
		return err
	}
	_, err = c.conn.Write(payload)
	return err
}

// maxMessage bounds what greetd may send us, so a broken socket cannot
// make the greeter allocate without end.
const maxMessage = 1 << 20

// createSession starts a conversation for user.
func (c *greetdClient) createSession(user string) error {
	return c.send(map[string]any{"type": "create_session", "username": user})
}

// post answers an authentication message with what the user typed. A
// secret or visible message is a question, so the response is always set,
// empty when nothing was typed.
func (c *greetdClient) post(response string) error {
	return c.send(map[string]any{"type": "post_auth_message_response", "response": response})
}

// postEmpty answers an informative or error message, which is not a
// question: the response field is left out, as greetd allows.
func (c *greetdClient) postEmpty() error {
	return c.send(map[string]any{"type": "post_auth_message_response"})
}

// startSession asks greetd to start cmd as the user, authenticated.
func (c *greetdClient) startSession(cmd []string) error {
	return c.send(map[string]any{"type": "start_session", "cmd": cmd})
}

// cancel ends the conversation: no session is started for it.
func (c *greetdClient) cancel() {
	_ = c.send(map[string]any{"type": "cancel_session"})
}

func (c *greetdClient) close() {
	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
	}
}

// read waits for greetd's next response.
func (c *greetdClient) read() (greetdResponse, error) {
	var r greetdResponse
	var header [4]byte
	if _, err := io.ReadFull(c.conn, header[:]); err != nil {
		return r, err
	}
	size := binary.LittleEndian.Uint32(header[:])
	if size > maxMessage {
		return r, fmt.Errorf("greetd: response of %d bytes is too large", size)
	}
	payload := make([]byte, size)
	if _, err := io.ReadFull(c.conn, payload); err != nil {
		return r, err
	}
	if err := json.Unmarshal(payload, &r); err != nil {
		return r, fmt.Errorf("greetd: %w", err)
	}
	return r, nil
}
