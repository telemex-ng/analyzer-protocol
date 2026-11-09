package fixtures_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/telemex-ng/analyzer-protocol/astm"
	"github.com/telemex-ng/analyzer-protocol/fixtures"
	"github.com/telemex-ng/analyzer-protocol/hl7"
)

func TestHL7QueryFixture(t *testing.T) {
	msg, err := hl7.Parse(fixtures.HL7Query)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if msg.Kind != "QUERY" {
		t.Fatalf("kind = %q", msg.Kind)
	}
	if msg.QueryBarcode != "010000000000016" {
		t.Fatalf("barcode = %q", msg.QueryBarcode)
	}
	if msg.ControlID != "1" {
		t.Fatalf("control id = %q", msg.ControlID)
	}
}

func TestHL7OrderResponseFixture(t *testing.T) {
	now := time.Date(2026, 11, 8, 14, 0, 0, 0, time.UTC)
	birth := time.Date(1962, 8, 24, 0, 0, 0, 0, time.UTC)
	got := hl7.BuildOrderResponse("1", hl7.OrderPatient{
		HospitalNumber: "PAT-001", Name: "Tommy", Sex: "M", BirthDate: birth,
		PatientType: "OUTPATIENT",
	}, hl7.OrderSample{
		Barcode: "010000000000016", SampleID: "SPC-0123456789AB",
		SampleType: "SERUM", Sender: "LIS", Department: "CHEM",
		SendingTime: now, Tests: []hl7.OrderTest{{Code: "2", Name: "TBil"}},
	}, now)
	if !bytes.Equal(got, fixtures.HL7OrderResponse) {
		t.Fatalf("order response mismatch\n got: %q\nwant: %q", got, fixtures.HL7OrderResponse)
	}
}

func TestASTMQueryFixture(t *testing.T) {
	msg, err := astm.Parse(fixtures.ASTMQuery)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if msg.Kind != "QUERY" {
		t.Fatalf("kind = %q", msg.Kind)
	}
	if msg.QueryBarcode != "010000000000016" {
		t.Fatalf("barcode = %q", msg.QueryBarcode)
	}
}

func TestASTMOrderResponseFixture(t *testing.T) {
	now := time.Date(2026, 11, 8, 14, 0, 0, 0, time.UTC)
	birth := time.Date(1962, 8, 24, 0, 0, 0, 0, time.UTC)
	got := astm.BuildOrderResponse(astm.OrderPatient{
		ID: "PAT-001", Name: "Tommy", Sex: "M", BirthDate: birth,
	}, []astm.OrderSample{{
		Sequence: 1, SampleID: "SPC-0123456789AB", Barcode: "010000000000016",
		SampleType: "SERUM", Priority: "R", Sender: "LIS", Department: "CHEM",
		Tests: []astm.OrderTest{{Code: "2", Name: "TBil"}},
	}}, now)
	if !bytes.Equal(got, fixtures.ASTMOrderResponse) {
		t.Fatalf("order response mismatch\n got: %q\nwant: %q", got, fixtures.ASTMOrderResponse)
	}
}
