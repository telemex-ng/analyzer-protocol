// Package mllp implements the Minimal Lower Layer Protocol framing used by
// HL7 v2 analyzers over TCP. A frame is <VT>payload<FS><CR>.
package mllp

import (
	"bufio"
	"errors"
	"io"
)

const (
	StartBlock     = 0x0b
	EndBlock       = 0x1c
	CarriageReturn = 0x0d
	DefaultMax     = 10 << 20 // 10 MiB
)

var (
	ErrTooLarge          = errors.New("mllp: message exceeds maximum size")
	ErrMissingStartBlock = errors.New("mllp: missing start block")
)

// ReadMessage reads one MLLP frame. It tolerates leading CR/LF/NUL bytes that
// some serial-to-TCP bridges emit, but rejects arbitrary unframed data rather
// than silently treating it as a message.
func ReadMessage(r *bufio.Reader, max int) ([]byte, error) {
	if max <= 0 {
		max = DefaultMax
	}
	for {
		b, err := r.ReadByte()
		if err != nil {
			return nil, err
		}
		switch b {
		case StartBlock:
			return readBody(r, max)
		case CarriageReturn, '\n', 0:
			continue
		default:
			return nil, ErrMissingStartBlock
		}
	}
}

func readBody(r *bufio.Reader, max int) ([]byte, error) {
	payload := make([]byte, 0, 1024)
	for {
		b, err := r.ReadByte()
		if err != nil {
			return nil, err
		}
		if b == EndBlock {
			// Consume any terminator bytes that are already buffered. Do not
			// block waiting for one: a sender may omit the optional CR, and
			// the next ReadMessage tolerates leading CR/LF anyway.
			for r.Buffered() > 0 {
				next, err := r.Peek(1)
				if err != nil || (next[0] != CarriageReturn && next[0] != '\n') {
					break
				}
				_, _ = r.ReadByte()
			}
			return payload, nil
		}
		if len(payload) >= max {
			return nil, ErrTooLarge
		}
		payload = append(payload, b)
	}
}

// WriteMessage writes one MLLP frame to w.
func WriteMessage(w io.Writer, payload []byte) error {
	frame := make([]byte, 0, len(payload)+3)
	frame = append(frame, StartBlock)
	frame = append(frame, payload...)
	frame = append(frame, EndBlock, CarriageReturn)
	_, err := w.Write(frame)
	return err
}

// Frame returns the MLLP bytes for payload.
func Frame(payload []byte) []byte {
	frame := make([]byte, 0, len(payload)+3)
	frame = append(frame, StartBlock)
	frame = append(frame, payload...)
	frame = append(frame, EndBlock, CarriageReturn)
	return frame
}
