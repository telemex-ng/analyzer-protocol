package hl7

import (
	"strings"
	"testing"
)

const sampleResult = "MSH|^~\\&|BS240|LAB|LIS|CLOUD|20261108120000||ORU^R01|MSG-1|P|2.3.1\r" +
	"PID|1||PAT123||Doe^Jane\r" +
	"OBR|1|REQ|ACC|CBC^Complete blood count|||20261108120000\r" +
	"OBX|1|NM|HB^Haemoglobin||9.4|g/dL|12.0-16.0|L|||F|||20261108120000\r"

func TestParseResult(t *testing.T) {
	msg, err := Parse([]byte(sampleResult))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if msg.Kind != "RESULT" || msg.Type != "ORU^R01" {
		t.Fatalf("kind/type = %s/%s", msg.Kind, msg.Type)
	}
	if msg.ControlID != "MSG-1" {
		t.Fatalf("control id = %q", msg.ControlID)
	}
	if msg.SampleID != "ACC" {
		t.Fatalf("sample id = %q", msg.SampleID)
	}
	if msg.PatientID != "PAT123" || msg.PatientName != "Jane Doe" {
		t.Fatalf("patient = %q / %q", msg.PatientID, msg.PatientName)
	}
	if msg.TestCode != "CBC" || msg.TestName != "Complete blood count" {
		t.Fatalf("order test = %q / %q", msg.TestCode, msg.TestName)
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
	if obs.ObservedAt == nil {
		t.Fatal("observed at is nil")
	}
}

func TestParseEscapedValue(t *testing.T) {
	raw := "MSH|^~\\&|A|B|C|D|20261108120000||ORU^R01|MSG-2|P|2.3.1\r" +
		"OBR|1|REQ|ACC|CBC^CBC\r" +
		"OBX|1|ST|NOTE^Note||Hello\\F\\World|||N|||F\r"
	msg, err := Parse([]byte(raw))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got := msg.Observations[0].Value; got != "Hello|World" {
		t.Fatalf("value = %q", got)
	}
	if msg.Observations[0].NumericValue != nil {
		t.Fatal("text value must not have a numeric value")
	}
}

func TestParseQC(t *testing.T) {
	raw := "MSH|^~\\&|A|B|C|D|20261108120000||ORU^R01|MSG-3|P|2.3.1\r" +
		"OBR|1|QC-1|QC-1|QC^Quality control\r" +
		"OBX|1|NM|GLU^Glucose||5.2|mmol/L|3.9-5.5|N|||F\r"
	msg, err := Parse([]byte(raw))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if msg.Kind != "QC" {
		t.Fatalf("kind = %q, want QC", msg.Kind)
	}
}

func TestParseQuery(t *testing.T) {
	raw := "MSH|^~\\&|A|B|C|D|20261108120000||QRY^Q02|MSG-4|P|2.3.1\r" +
		"QRD|20261108120000|R|I|Q1|||1^RD|ACC|DEM\r"
	msg, err := Parse([]byte(raw))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if msg.Kind != "QUERY" {
		t.Fatalf("kind = %q, want QUERY", msg.Kind)
	}
}

func TestParseRejectsNonMSH(t *testing.T) {
	if _, err := Parse([]byte("PID|1||X\r")); err == nil {
		t.Fatal("expected an error")
	}
}

func TestBuildACK(t *testing.T) {
	original, err := Parse([]byte(sampleResult))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	ack := string(BuildACK(original, true, ""))
	if !strings.Contains(ack, "MSH|") || !strings.Contains(ack, "MSA|AA|MSG-1") {
		t.Fatalf("ack = %q", ack)
	}
	nak := string(BuildACK(original, false, "spool failed"))
	if !strings.Contains(nak, "MSA|AE|MSG-1|spool failed") {
		t.Fatalf("nak = %q", nak)
	}
}
