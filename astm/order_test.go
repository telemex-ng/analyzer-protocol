package astm

import (
	"strings"
	"testing"
	"time"
)

func TestParseHostQuery(t *testing.T) {
	msg, err := Parse([]byte("Q|1|^010000000000016||||||O\r"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if msg.Kind != "QUERY" || msg.QueryBarcode != "010000000000016" {
		t.Fatalf("message = %+v", msg)
	}
}

func TestBuildOrderResponse(t *testing.T) {
	now := time.Date(2026, 11, 8, 14, 0, 0, 0, time.UTC)
	payload := string(BuildOrderResponse(OrderPatient{
		ID: "PAT-001", Name: "Tommy", Sex: "M",
		BirthDate: time.Date(1962, 8, 24, 0, 0, 0, 0, time.UTC),
	}, []OrderSample{{
		Sequence: 1, SampleID: "SPC-0123456789AB", Barcode: "010000000000016",
		SampleType: "SERUM", Tests: []OrderTest{{Code: "2", Name: "TBil"}},
	}}, now))
	for _, want := range []string{
		"H|\\^&", "P|1||PAT-001", "O|1|SPC-0123456789AB|010000000000016",
		"1^TBil^1^1", "L|1|N",
	} {
		if !strings.Contains(payload, want) {
			t.Fatalf("payload missing %q:\n%s", want, payload)
		}
	}
}
