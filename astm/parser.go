// Package astm parses the ASTM E1394/ASCM record stream used by haematology
// and coagulation analyzers such as the Sysmex XP-300. It supports the common
// H/P/O/R/L layout and the low-level ACK/NAK handshake bytes.
package astm

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/telemex-ng/analyzer-protocol/values"
)

const (
	STX        = 0x02
	ETX        = 0x03
	EOT        = 0x04
	ENQ        = 0x05
	ACK        = 0x06
	NAK        = 0x15
	DefaultMax = 10 << 20
)

var ErrTooLarge = errors.New("astm: message exceeds maximum size")

// Message is the normalized analyzer result envelope.
type Message struct {
	Type         string
	ControlID    string
	SampleID     string
	QueryBarcode string
	PatientID    string
	PatientName  string
	TestCode     string
	TestName     string
	Kind         string
	Observations []Observation
	Raw          string
}

// Observation is one R record result line.
type Observation struct {
	TestCode       string
	TestName       string
	Value          string
	NumericValue   *float64
	Unit           string
	ReferenceRange string
	ReferenceLow   *float64
	ReferenceHigh  *float64
	AbnormalFlag   string
	ResultStatus   string
	ObservedAt     *time.Time
	Raw            string
}

// ReadFrame reads one ASTM frame. It terminates on an L record, ETX or EOT.
// Leading ENQ/ACK/NAK handshake bytes are ignored.
func ReadFrame(r *bufio.Reader, max int) ([]byte, error) {
	if max <= 0 {
		max = DefaultMax
	}
	payload := make([]byte, 0, 1024)
	for {
		b, err := r.ReadByte()
		if err != nil {
			return nil, err
		}
		switch b {
		case ETX, EOT:
			return trimFrame(payload), nil
		case ENQ, ACK, NAK:
			if len(payload) == 0 {
				continue
			}
			if len(payload) >= max {
				return nil, ErrTooLarge
			}
			payload = append(payload, b)
		case STX:
			continue
		case '\r', '\n':
			if len(payload) >= max {
				return nil, ErrTooLarge
			}
			payload = append(payload, b)
			if isTerminator(payload) {
				return trimFrame(payload), nil
			}
		default:
			if len(payload) >= max {
				return nil, ErrTooLarge
			}
			payload = append(payload, b)
		}
	}
}

func isTerminator(payload []byte) bool {
	text := strings.TrimRight(string(payload), "\r\n")
	if text == "" {
		return false
	}
	lines := strings.Split(text, "\r")
	last := strings.TrimSpace(lines[len(lines)-1])
	if last == "" {
		return false
	}
	fields := strings.Split(last, "|")
	return len(fields) > 0 && strings.EqualFold(strings.TrimSpace(fields[0]), "L")
}

func trimFrame(payload []byte) []byte {
	return []byte(strings.Trim(string(payload), "\x00\r\n"))
}

// Parse reads one ASTM record stream.
func Parse(raw []byte) (Message, error) {
	text := normalize(raw)
	if strings.TrimSpace(text) == "" {
		return Message{}, errors.New("astm: empty message")
	}
	records := splitRecords(text)
	if len(records) == 0 {
		return Message{}, errors.New("astm: no records")
	}

	msg := Message{Type: "ASTM", Kind: "UNKNOWN", Raw: string(raw)}
	var currentOrderTestCode, currentOrderTestName string
	hasQueryRecord := false
	hasQCSample := false

	for _, record := range records {
		parts := strings.Split(record, "|")
		if len(parts) == 0 {
			continue
		}
		switch strings.ToUpper(strings.TrimSpace(parts[0])) {
		case "H":
			// Header. Delimiters are fixed to the ASTM defaults; vendor
			// profiles that change them need an explicit profile parser.
		case "P":
			msg.PatientID = clean(field(parts, 3))
			msg.PatientName = formatName(clean(field(parts, 5)))
		case "O":
			// O-4 is the instrument specimen ID and carries the tube
			// barcode in bidirectional workflows; O-3 is the tray/position
			// sample ID. Prefer O-4, then fall back to O-3.
			sample := clean(field(parts, 3))
			if sample == "" {
				sample = clean(field(parts, 2))
			}
			if msg.SampleID == "" {
				msg.SampleID = sample
			}
			if isQCSample(sample) {
				hasQCSample = true
			}
			currentOrderTestCode = component(clean(field(parts, 4)), 0)
			currentOrderTestName = component(clean(field(parts, 4)), 1)
			if msg.TestCode == "" {
				msg.TestCode = currentOrderTestCode
				msg.TestName = currentOrderTestName
			}
		case "R":
			obs := Observation{
				TestCode:       firstNonEmpty(component(clean(field(parts, 2)), 0), currentOrderTestCode),
				TestName:       firstNonEmpty(component(clean(field(parts, 2)), 1), currentOrderTestName),
				Value:          clean(field(parts, 3)),
				Unit:           clean(field(parts, 4)),
				ReferenceRange: clean(field(parts, 5)),
				AbnormalFlag:   strings.ToUpper(clean(field(parts, 6))),
				ResultStatus:   strings.ToUpper(clean(field(parts, 8))),
				ObservedAt:     values.Timestamp(clean(field(parts, 9))),
				Raw:            record,
			}
			obs.NumericValue = values.Numeric(obs.Value)
			obs.ReferenceLow, obs.ReferenceHigh = values.Range(obs.ReferenceRange)
			if obs.TestName == "" {
				obs.TestName = obs.TestCode
			}
			if obs.TestCode != "" || obs.Value != "" {
				msg.Observations = append(msg.Observations, obs)
			}
		case "Q":
			// Q-3 carries the specimen/barcode for a host query. A Q
			// record whose specimen is a control material is QC, not an
			// order lookup.
			specimen := clean(field(parts, 2))
			if isQCField(specimen) {
				hasQCSample = true
			} else {
				msg.QueryBarcode = barcodeField(specimen)
				hasQueryRecord = true
			}
		case "L", "C":
			// L is the terminator; C is a free-text comment.
		}
	}

	if msg.SampleID == "" {
		// Some analyzers put the sample ID only on the O record's
		// instrument field; already handled above. An empty sample ID
		// stays empty so ingest can record it as unmatched.
	}
	switch {
	case hasQueryRecord:
		msg.Kind = "QUERY"
	case hasQCSample || isQCSample(msg.SampleID):
		msg.Kind = "QC"
	case len(msg.Observations) > 0:
		msg.Kind = "RESULT"
	default:
		msg.Kind = "UNKNOWN"
	}
	msg.ControlID = controlID(raw)
	return msg, nil
}

// BuildACK returns the ASTM low-level acknowledgement byte. Accepted messages
// receive <ACK>; rejected messages receive <NAK>.
func BuildACK(accepted bool) []byte {
	if accepted {
		return []byte{ACK}
	}
	return []byte{NAK}
}

func controlID(raw []byte) string {
	sum := sha256.Sum256(raw)
	return "astm-" + hex.EncodeToString(sum[:])[:32]
}

func normalize(raw []byte) string {
	text := string(raw)
	replacer := strings.NewReplacer("\x00", "", "\x02", "", "\x03", "", "\x04", "",
		"\x05", "", "\x06", "", "\x15", "", "\x17", "")
	text = replacer.Replace(text)
	text = strings.ReplaceAll(text, "\r\n", "\r")
	text = strings.ReplaceAll(text, "\n", "\r")
	return text
}

func splitRecords(text string) []string {
	out := []string{}
	for _, line := range strings.Split(text, "\r") {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func field(parts []string, n int) string {
	if n < 0 || n >= len(parts) {
		return ""
	}
	return parts[n]
}

func component(raw string, n int) string {
	parts := strings.Split(raw, "^")
	if n < 0 || n >= len(parts) {
		return ""
	}
	return strings.TrimSpace(parts[n])
}

func formatName(raw string) string {
	parts := strings.Split(raw, "^")
	if len(parts) >= 2 {
		family := strings.TrimSpace(parts[0])
		given := strings.TrimSpace(parts[1])
		switch {
		case given != "" && family != "":
			return given + " " + family
		case given != "":
			return given
		default:
			return family
		}
	}
	return strings.TrimSpace(raw)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func isQCSample(sample string) bool {
	upper := strings.ToUpper(strings.TrimSpace(sample))
	for _, prefix := range []string{"QC", "CTRL", "CONTROL", "CAL", "CALIB", "BLANK", "WATER"} {
		if strings.HasPrefix(upper, prefix) {
			return true
		}
	}
	return false
}

func isQCField(raw string) bool {
	for _, part := range strings.Split(raw, "^") {
		if isQCSample(part) {
			return true
		}
	}
	return false
}

// barcodeField returns the last non-empty component of an ASTM specimen field.
// Vendors put the tray/position in earlier components and the barcode last.
func barcodeField(raw string) string {
	parts := strings.Split(raw, "^")
	for i := len(parts) - 1; i >= 0; i-- {
		if value := strings.TrimSpace(parts[i]); value != "" {
			return value
		}
	}
	return ""
}

func clean(raw string) string {
	return strings.TrimSpace(raw)
}
