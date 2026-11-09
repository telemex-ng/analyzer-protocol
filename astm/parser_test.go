package astm

import (
	"bufio"
	"bytes"
	"strings"
	"testing"
)

const sampleResult = "H|\\^&|||Sysmex|||||||P|E1394-97|20261108120000\r" +
	"P|1||PAT123||Doe^Jane\r" +
	"O|1|ACC||CBC^Complete blood count||||||||||||||||||F\r" +
	"R|1|HB^Haemoglobin|9.4|g/dL|12.0-16.0|L||F||||20261108120000\r" +
	"L|1|N\r"

func TestParseResult(t *testing.T) {
	msg, err := Parse([]byte(sampleResult))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if msg.Kind != "RESULT" {
		t.Fatalf("kind = %q", msg.Kind)
	}
	if msg.SampleID != "ACC" {
		t.Fatalf("sample id = %q", msg.SampleID)
	}
	if msg.PatientID != "PAT123" || msg.PatientName != "Jane Doe" {
		t.Fatalf("patient = %q / %q", msg.PatientID, msg.PatientName)
	}
	if msg.TestCode != "CBC" {
		t.Fatalf("order test = %q", msg.TestCode)
	}
	if len(msg.Observations) != 1 {
		t.Fatalf("observations = %d", len(msg.Observations))
	}
	obs := msg.Observations[0]
	if obs.TestCode != "HB" || obs.TestName != "Haemoglobin" {
		t.Fatalf("obs test = %q / %q", obs.TestCode, obs.TestName)
	}
	if obs.NumericValue == nil || *obs.NumericValue != 9.4 {
		t.Fatalf("numeric value = %v", obs.NumericValue)
	}
	if obs.Unit != "g/dL" || obs.AbnormalFlag != "L" || obs.ResultStatus != "F" {
		t.Fatalf("unit/flag/status = %q / %q / %q", obs.Unit, obs.AbnormalFlag, obs.ResultStatus)
	}
	if obs.ReferenceLow == nil || *obs.ReferenceLow != 12 ||
		obs.ReferenceHigh == nil || *obs.ReferenceHigh != 16 {
		t.Fatalf("reference = %v - %v", obs.ReferenceLow, obs.ReferenceHigh)
	}
	if msg.ControlID == "" || !strings.HasPrefix(msg.ControlID, "astm-") {
		t.Fatalf("control id = %q", msg.ControlID)
	}
}

func TestParseQC(t *testing.T) {
	raw := "H|\\^&|||Sysmex\r" +
		"Q|1|^QC1||ALL||||||||O\r" +
		"R|1|GLU^Glucose|5.2|mmol/L|3.9-5.5|N||F\r" +
		"L|1|N\r"
	msg, err := Parse([]byte(raw))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if msg.Kind != "QC" {
		t.Fatalf("kind = %q, want QC", msg.Kind)
	}
}

func TestReadFrame(t *testing.T) {
	frame := append([]byte{STX, ENQ}, []byte(sampleResult)...)
	frame = append(frame, ETX)
	got, err := ReadFrame(bufio.NewReader(bytes.NewReader(frame)), DefaultMax)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.HasPrefix(string(got), "H|") || !strings.Contains(string(got), "L|1|N") {
		t.Fatalf("frame = %q", got)
	}
	if strings.ContainsAny(string(got), "\x02\x03\x05") {
		t.Fatalf("frame contains control bytes: %q", got)
	}
}

func TestBuildACK(t *testing.T) {
	if got := BuildACK(true); len(got) != 1 || got[0] != ACK {
		t.Fatalf("ack = %v", got)
	}
	if got := BuildACK(false); len(got) != 1 || got[0] != NAK {
		t.Fatalf("nak = %v", got)
	}
}
