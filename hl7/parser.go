// Package hl7 parses the subset of HL7 v2.x needed for analyzer result
// ingestion (ORU^R01) and builds a matching MLLP-framed ACK. It is deliberately
// strict about the message header and lenient about optional fields, because
// analyzer vendors differ in which optional segments they populate.
package hl7

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/telemex-ng/analyzer-protocol/values"
)

// Message is the normalized analyzer result envelope.
type Message struct {
	Type              string
	ControlID         string
	SampleID          string
	QueryBarcode      string
	PatientID         string
	PatientName       string
	TestCode          string
	TestName          string
	SendingApp        string
	SendingFacility   string
	ReceivingApp      string
	ReceivingFacility string
	Version           string
	Kind              string
	Observations      []Observation
	Raw               string
}

// Observation is one OBX result line.
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

type parser struct {
	fieldSep byte
	compSep  byte
	repSep   byte
	escSep   byte
	subSep   byte
}

type segment struct {
	name  string
	parts []string
	p     *parser
}

// Parse reads one HL7 v2 message. Segment separators may be CR, LF or CRLF.
func Parse(raw []byte) (Message, error) {
	text := normalize(raw)
	lines := splitSegments(text)
	if len(lines) == 0 {
		return Message{}, errors.New("hl7: empty message")
	}
	p, err := newParser(lines[0])
	if err != nil {
		return Message{}, err
	}
	msh := p.segment(lines[0])
	if msh.name != "MSH" {
		return Message{}, errors.New("hl7: first segment must be MSH")
	}

	msg := Message{
		Type:              clean(msh.field(9)),
		ControlID:         clean(msh.field(10)),
		SendingApp:        clean(msh.field(3)),
		SendingFacility:   clean(msh.field(4)),
		ReceivingApp:      clean(msh.field(5)),
		ReceivingFacility: clean(msh.field(6)),
		Version:           clean(msh.field(12)),
		Raw:               string(raw),
	}

	var obr *segment
	var pid *segment
	var qrd *segment
	for _, line := range lines[1:] {
		seg := p.segment(line)
		switch seg.name {
		case "PID":
			if pid == nil {
				pid = &seg
			}
		case "OBR":
			if obr == nil {
				obr = &seg
			}
		case "QRD":
			if qrd == nil {
				qrd = &seg
			}
		case "OBX":
			obs := parseOBX(seg)
			if obs.Value != "" || obs.TestCode != "" {
				msg.Observations = append(msg.Observations, obs)
			}
		}
	}

	if qrd != nil {
		// QRD-8 is the "who subject filter" and carries the tube barcode
		// for the Mindray host-query flow.
		msg.QueryBarcode = clean(qrd.field(8))
	}

	if pid != nil {
		msg.PatientID = firstComponent(pid.field(3))
		msg.PatientName = formatName(pid)
	}
	if obr != nil {
		msg.SampleID = firstNonEmpty(
			firstComponent(obr.field(3)),
			firstComponent(obr.field(2)),
			firstComponent(pidField(pid, 3)),
		)
		msg.TestCode = obr.component(4, 0)
		msg.TestName = obr.component(4, 1)
	} else if pid != nil {
		msg.SampleID = firstComponent(pid.field(3))
	}
	if msg.TestCode == "" && len(msg.Observations) > 0 {
		msg.TestCode = msg.Observations[0].TestCode
		msg.TestName = msg.Observations[0].TestName
	}

	switch {
	case isQuery(msg.Type):
		msg.Kind = "QUERY"
	case isQC(msg.SampleID, msg.PatientID, msg.Type):
		msg.Kind = "QC"
	case strings.HasPrefix(strings.ToUpper(msg.Type), "ORU") || len(msg.Observations) > 0:
		msg.Kind = "RESULT"
	default:
		msg.Kind = "UNKNOWN"
	}
	return msg, nil
}

func newParser(line string) (*parser, error) {
	if len(line) < 4 || !strings.HasPrefix(line, "MSH") {
		return nil, errors.New("hl7: malformed MSH segment")
	}
	fieldSep := line[3]
	rest := line[4:]
	end := strings.IndexByte(rest, fieldSep)
	if end < 0 {
		return nil, errors.New("hl7: MSH-2 encoding characters are missing")
	}
	encoding := rest[:end]
	p := &parser{fieldSep: fieldSep, compSep: '^', repSep: '~', escSep: '\\', subSep: '&'}
	if len(encoding) > 0 {
		p.compSep = encoding[0]
	}
	if len(encoding) > 1 {
		p.repSep = encoding[1]
	}
	if len(encoding) > 2 {
		p.escSep = encoding[2]
	}
	if len(encoding) > 3 {
		p.subSep = encoding[3]
	}
	return p, nil
}

func (p *parser) segment(line string) segment {
	return segment{name: strings.TrimSpace(strings.SplitN(line, string(p.fieldSep), 2)[0]),
		parts: strings.Split(line, string(p.fieldSep)), p: p}
}

func (s segment) field(n int) string {
	if s.name == "MSH" {
		if n == 1 {
			return string(s.p.fieldSep)
		}
		n--
	}
	if n < 0 || n >= len(s.parts) {
		return ""
	}
	return s.parts[n]
}

func (s segment) component(fieldN, componentN int) string {
	field := s.field(fieldN)
	if field == "" {
		return ""
	}
	parts := strings.Split(field, string(s.p.compSep))
	if componentN < 0 || componentN >= len(parts) {
		return ""
	}
	return clean(parts[componentN])
}

func (s segment) firstRep(fieldN int) string {
	field := s.field(fieldN)
	if field == "" {
		return ""
	}
	return strings.Split(field, string(s.p.repSep))[0]
}

func parseOBX(seg segment) Observation {
	value := seg.firstRep(5)
	obs := Observation{
		TestCode:       seg.component(3, 0),
		TestName:       seg.component(3, 1),
		Value:          clean(value),
		Unit:           clean(seg.field(6)),
		ReferenceRange: clean(seg.field(7)),
		AbnormalFlag:   strings.ToUpper(clean(seg.field(8))),
		ResultStatus:   strings.ToUpper(clean(seg.field(11))),
		ObservedAt:     values.Timestamp(clean(seg.field(14))),
		Raw:            strings.Join(seg.parts, string(seg.p.fieldSep)),
	}
	obs.NumericValue = values.Numeric(obs.Value)
	obs.ReferenceLow, obs.ReferenceHigh = values.Range(obs.ReferenceRange)
	if obs.TestName == "" {
		obs.TestName = obs.TestCode
	}
	return obs
}

func pidField(pid *segment, n int) string {
	if pid == nil {
		return ""
	}
	return pid.field(n)
}

func formatName(pid *segment) string {
	family := pid.component(5, 0)
	given := pid.component(5, 1)
	switch {
	case given != "" && family != "":
		return given + " " + family
	case given != "":
		return given
	default:
		return family
	}
}

func firstComponent(raw string) string {
	if raw == "" {
		return ""
	}
	return clean(strings.Split(raw, "^")[0])
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func isQuery(messageType string) bool {
	upper := strings.ToUpper(messageType)
	return strings.HasPrefix(upper, "QRY") || strings.HasPrefix(upper, "QBP")
}

func isQC(sampleID, patientID, messageType string) bool {
	upperType := strings.ToUpper(messageType)
	if strings.Contains(upperType, "QC") {
		return true
	}
	upperSample := strings.ToUpper(strings.TrimSpace(sampleID))
	for _, prefix := range []string{"QC", "CTRL", "CONTROL", "CAL", "CALIB", "BLANK", "WATER"} {
		if strings.HasPrefix(upperSample, prefix) {
			return true
		}
	}
	return strings.TrimSpace(patientID) == "" && strings.TrimSpace(sampleID) == ""
}

func normalize(raw []byte) string {
	text := string(raw)
	text = strings.ReplaceAll(text, "\r\n", "\r")
	text = strings.ReplaceAll(text, "\n", "\r")
	replacer := strings.NewReplacer("\x00", "", "\x0b", "", "\x1c", "",
		"\x03", "", "\x04", "", "\x05", "", "\x06", "", "\x15", "", "\x17", "")
	return replacer.Replace(text)
}

func splitSegments(text string) []string {
	out := []string{}
	for _, line := range strings.Split(text, "\r") {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func clean(raw string) string {
	return strings.TrimSpace(unescape(raw))
}

func unescape(raw string) string {
	replacer := strings.NewReplacer(
		`\F\`, `|`,
		`\S\`, `^`,
		`\R\`, `~`,
		`\E\`, `\`,
		`\T\`, `&`,
	)
	return replacer.Replace(raw)
}

func escapeField(raw string) string {
	replacer := strings.NewReplacer(
		`\`, `\E\`,
		`|`, `\F\`,
		`^`, `\S\`,
		`~`, `\R\`,
		`&`, `\T\`,
		"\r", " ",
		"\n", " ",
	)
	return replacer.Replace(raw)
}

// BuildACK returns an HL7 ACK payload (without MLLP framing) for an inbound
// message. Accepted messages get MSA|AA; rejected messages get MSA|AE with a
// short, separator-safe reason.
func BuildACK(original Message, accepted bool, text string) []byte {
	now := time.Now().UTC()
	controlID := "ACK-" + uuid.NewString()
	sendingApp := firstNonEmpty(original.ReceivingApp, "LIS")
	sendingFacility := original.ReceivingFacility
	receivingApp := original.SendingApp
	receivingFacility := original.SendingFacility
	version := firstNonEmpty(original.Version, "2.3.1")
	originalControlID := firstNonEmpty(original.ControlID, "UNKNOWN")

	msh := fmt.Sprintf("MSH|^~\\&|%s|%s|%s|%s|%s||ACK|%s|P|%s",
		escapeField(sendingApp), escapeField(sendingFacility),
		escapeField(receivingApp), escapeField(receivingFacility),
		now.Format("20060102150405"), escapeField(controlID), escapeField(version))
	msa := fmt.Sprintf("MSA|AA|%s", escapeField(originalControlID))
	if !accepted {
		msa = fmt.Sprintf("MSA|AE|%s|%s", escapeField(originalControlID), escapeField(text))
	}
	return []byte(msh + "\r" + msa + "\r")
}
