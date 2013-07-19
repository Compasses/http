package client

import (
	"bytes"
	"testing"
)

var phaseStringTests = []struct {
	phase
	expected string
}{
	{0, "requestline"},
	{1, "headers"},
	{2, "body"},
	{3, "UNKNOWN"},
}

func TestPhaseString(t *testing.T) {
	for _, tt := range phaseStringTests {
		actual := tt.phase.String()
		if actual != tt.expected {
			t.Errorf("phase(%d).String(): expected %q, got %q", tt.phase, tt.expected, actual)
		}
	}
}

func TestPhaseError(t *testing.T) {
	var c Conn
	err := c.WriteHeader("Host", "localhost")
	if _, ok := err.(*phaseError); !ok {
		t.Fatalf("expected %T, got %v", new(phaseError), err)
	}
	expected := `phase error: expected headers, got requestline`
	if actual := err.Error(); actual != expected {
		t.Fatalf("phaseError.Error(): expected %q, got %q", expected, actual)
	}
}

func TestNewConn(t *testing.T) {
	var b bytes.Buffer
	NewConn(&b)
}

var writeRequestLineTests = []struct {
	method, uri, version string
	expected             string
}{
	{"GET", "/foo", "HTTP/1.0", "GET /foo HTTP/1.0\r\n"},
}

func TestConnWriteRequestLine(t *testing.T) {
	for _, tt := range writeRequestLineTests {
		var b bytes.Buffer
		c := NewConn(&b)
		if err := c.WriteRequestLine(tt.method, tt.uri, tt.version); err != nil {
			t.Fatalf("Conn.WriteRequestLine(%q, %q, %q): %v", tt.method, tt.uri, tt.version, err)
		}
		if actual := b.String(); actual != tt.expected {
			t.Errorf("Conn.WriteRequestLine(%q, %q, %q): expected %q, got %q", tt.method, tt.uri, tt.version, tt.expected, actual)
		}
	}
}

func TestConnDoubleRequestLine(t *testing.T) {
	var b bytes.Buffer
	c := NewConn(&b)
	if err := c.WriteRequestLine("GET", "/hello", "HTTP/0.9"); err != nil {
		t.Fatal(err)
	}
	err := c.WriteRequestLine("GET", "/hello", "HTTP/0.9")
	expected := `phase error: expected requestline, got headers`
	if actual := err.Error(); actual != expected {
		t.Fatalf("phaseError.Error(): expected %q, got %q", expected, actual)
	}
}

var writeHeaderTests = []struct {
	key, value string
	expected   string
}{
	{"Host", "localhost", "Host: localhost\r\n"},
}

func TestConnWriteHeader(t *testing.T) {
	for _, tt := range writeHeaderTests {
		var b bytes.Buffer
		c := NewConn(&b)
		c.StartHeaders()
		if err := c.WriteHeader(tt.key, tt.value); err != nil {
			t.Fatalf("Conn.WriteHeader(%q, %q): %v", tt.key, tt.value, err)
		}
		if actual := b.String(); actual != tt.expected {
			t.Errorf("Conn.WriteHeader(%q, %q): expected %q, got %q", tt.key, tt.value, tt.expected, actual)
		}
	}
}

func TestStartBody(t *testing.T) {
	var b bytes.Buffer
	c := NewConn(&b)
	c.StartHeaders()
	if err := c.WriteHeader("Host", "localhost"); err != nil {
		t.Fatal(err)
	}
	c.StartBody()
	err := c.WriteHeader("Connection", "close")
	if _, ok := err.(*phaseError); !ok {
		t.Fatalf("expected %T, got %v", new(phaseError), err)
	}
	expected := `phase error: expected headers, got body`
	if actual := err.Error(); actual != expected {
		t.Fatalf("phaseError.Error(): expected %q, got %q", expected, actual)
	}
	expected = "Host: localhost\r\n\r\n"
	if actual := b.String(); actual != expected {
		t.Fatalf("StartBody: expected %q, got %q", expected, actual)
	}
}

func TestDoubleWriteBody(t *testing.T) {
	var b bytes.Buffer
	c := NewConn(&b)
	c.StartBody()
	if err := c.WriteBody([]byte{}); err != nil {
		t.Fatal(err)
	}
	err := c.WriteBody([]byte{})
	expected := `phase error: expected body, got requestline`
	if actual := err.Error(); actual != expected {
		t.Fatalf("phaseError.Error(): expected %q, got %q", expected, actual)
	}
}

type header struct{ key, value string }
type writeTest struct {
	headers  []header
	body     string
	expected string
}

var writeTests = []writeTest{
	{[]header{{"Host", "localhost"}, {"Connection", "close"}},
		"abcd1234",
		"Host: localhost\r\nConnection: close\r\n\r\nabcd1234",
	},
}

// test only method, real call will come from Client.
func (c *Conn) Write(t *testing.T, w writeTest) {
	c.StartHeaders()
	for _, h := range w.headers {
		if err := c.WriteHeader(h.key, h.value); err != nil {
			t.Fatal(err)
		}
	}
	c.StartBody()
	if err := c.WriteBody([]byte(w.body)); err != nil {
		t.Fatal(err)
	}
}

func TestWrite(t *testing.T) {
	for _, tt := range writeTests {
		var b bytes.Buffer
		c := NewConn(&b)
		c.Write(t, tt)
		if actual := b.String(); actual != tt.expected {
			t.Errorf("TestWrite: expected %q, got %q", tt.expected, actual)
		}
	}
}
