package mllp

import (
	"bufio"
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestWriteReadRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	payload := []byte("MSH|^~\\&|A|B|C|D|20260101120000||ORU^R01|1|P|2.3.1\r")
	if err := WriteMessage(&buf, payload); err != nil {
		t.Fatalf("write: %v", err)
	}
	got, err := ReadMessage(bufio.NewReader(&buf), DefaultMax)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("payload = %q, want %q", got, payload)
	}
}

func TestReadMessageToleratesLeadingWhitespace(t *testing.T) {
	frame := append([]byte{'\r', '\n', 0}, Frame([]byte("hello"))...)
	got, err := ReadMessage(bufio.NewReader(bytes.NewReader(frame)), DefaultMax)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(got) != "hello" {
		t.Fatalf("payload = %q", got)
	}
}

func TestReadMessageRejectsUnframedData(t *testing.T) {
	_, err := ReadMessage(bufio.NewReader(strings.NewReader("MSH|not-mllp")), DefaultMax)
	if !errors.Is(err, ErrMissingStartBlock) {
		t.Fatalf("error = %v, want ErrMissingStartBlock", err)
	}
}

func TestReadMessageTooLarge(t *testing.T) {
	payload := []byte(strings.Repeat("x", 64))
	_, err := ReadMessage(bufio.NewReader(bytes.NewReader(Frame(payload))), 8)
	if !errors.Is(err, ErrTooLarge) {
		t.Fatalf("error = %v, want ErrTooLarge", err)
	}
}

func TestFrameEndsWithMllpTerminator(t *testing.T) {
	frame := Frame([]byte("abc"))
	if len(frame) != 6 || frame[0] != StartBlock || frame[4] != EndBlock || frame[5] != CarriageReturn {
		t.Fatalf("frame = %#v", frame)
	}
}
