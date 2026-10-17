package main

import (
	"encoding/binary"
	"encoding/json"
	"net"
	"testing"
)

// TestFraming checks the wire format greetd-ipc(7) documents: a 32-bit
// native-endian length, then that many bytes of JSON.
func TestFraming(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()
	c := &greetdClient{conn: client}

	go func() {
		_ = c.createSession("me")
		_ = c.post("hunter2")
		_ = c.postEmpty()
		_ = c.startSession([]string{"/usr/local/bin/inua-session"})
	}()

	read := func() map[string]any {
		t.Helper()
		var header [4]byte
		if _, err := server.Read(header[:]); err != nil {
			t.Fatalf("read header: %v", err)
		}
		size := binary.LittleEndian.Uint32(header[:])
		payload := make([]byte, size)
		if _, err := server.Read(payload); err != nil {
			t.Fatalf("read payload: %v", err)
		}
		var m map[string]any
		if err := json.Unmarshal(payload, &m); err != nil {
			t.Fatalf("unmarshal %q: %v", payload, err)
		}
		return m
	}

	if m := read(); m["type"] != "create_session" || m["username"] != "me" {
		t.Errorf("create_session = %v", m)
	}
	if m := read(); m["type"] != "post_auth_message_response" || m["response"] != "hunter2" {
		t.Errorf("post = %v", m)
	}
	if m := read(); m["type"] != "post_auth_message_response" {
		t.Errorf("postEmpty = %v", m)
	} else if _, set := m["response"]; set {
		t.Errorf("postEmpty set response: %v", m)
	}
	if m := read(); m["type"] != "start_session" {
		t.Errorf("start_session = %v", m)
	}
}

// TestRead feeds one framed response and checks it is decoded.
func TestRead(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()
	c := &greetdClient{conn: client}

	go func() {
		payload, _ := json.Marshal(map[string]any{
			"type":              "auth_message",
			"auth_message_type": "secret",
			"auth_message":      "Password: ",
		})
		var header [4]byte
		binary.LittleEndian.PutUint32(header[:], uint32(len(payload)))
		_, _ = server.Write(header[:])
		_, _ = server.Write(payload)
	}()

	r, err := c.read()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if r.Type != "auth_message" || r.AuthMessageType != "secret" || r.AuthMessage != "Password: " {
		t.Errorf("read = %+v", r)
	}
}

// TestSplitExec keeps the desktop entry Exec parsing honest.
func TestSplitExec(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want []string
	}{
		{"/usr/local/bin/inua-session", []string{"/usr/local/bin/inua-session"}},
		{"start-sway", []string{"start-sway"}},
		{"uwsm start -e -D Hyprland hyprland.desktop", []string{"uwsm", "start", "-e", "-D", "Hyprland", "hyprland.desktop"}},
		{"/usr/bin/firefox %u", []string{"/usr/bin/firefox"}},
		{`"/opt/my app/run" --flag`, []string{"/opt/my app/run", "--flag"}},
	} {
		got, err := splitExec(tc.in)
		if err != nil {
			t.Errorf("splitExec(%q): %v", tc.in, err)
			continue
		}
		if len(got) != len(tc.want) {
			t.Errorf("splitExec(%q) = %q, want %q", tc.in, got, tc.want)
			continue
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("splitExec(%q) = %q, want %q", tc.in, got, tc.want)
				break
			}
		}
	}
}
