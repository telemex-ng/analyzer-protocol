package hl7

import (
	"strings"
	"testing"
	"time"
)

func TestParseHostQuery(t *testing.T) {
	raw := "MSH|^~\\&|BS240|LAB|||20261108120000||QRY^Q02|1|P|2.3.1|||ASCII|\r" +
		"QRD|20261108120000|R|D|1|||RD|010000000000016|OTH|||T|\r"
	msg, err := Parse([]byte(raw))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if msg.Kind != "QUERY" || msg.QueryBarcode != "010000000000016" {
		t.Fatalf("message = %+v", msg)
	}
}

func TestBuildQueryAck(t *testing.T) {
	found := string(BuildQueryAck("1", true, time.Now()))
	if !strings.Contains(found, "QCK^Q02") || !strings.Contains(found, "QAK|SR|OK") {
		t.Fatalf("found ack = %q", found)
	}
	missing := string(BuildQueryAck("1", false, time.Now()))
	if !strings.Contains(missing, "QAK|SR|NF") {
		t.Fatalf("missing ack = %q", missing)
	}
}

func TestBuildOrderResponse(t *testing.T) {
	now := time.Date(2026, 11, 8, 14, 0, 0, 0, time.UTC)
	payload := string(BuildOrderResponse("1", OrderPatient{
		HospitalNumber: "PAT-001", Name: "Tommy", Sex: "M",
		BirthDate: time.Date(1962, 8, 24, 0, 0, 0, 0, time.UTC),
	}, OrderSample{
		Barcode: "010000000000016", SampleID: "SPC-0123456789AB",
		SampleType: "SERUM", Tests: []OrderTest{{Code: "2", Name: "TBil"}},
	}, now))
	for _, want := range []string{
		"DSR^Q03", "QAK|SR|OK", "DSP|21||010000000000016",
		"DSP|22||SPC-0123456789AB", "DSP|29||2", "DSC|",
	} {
		if !strings.Contains(payload, want) {
			t.Fatalf("payload missing %q:\n%s", want, payload)
		}
	}
}
