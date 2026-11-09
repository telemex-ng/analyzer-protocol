package hl7

import (
	"fmt"
	"strings"
	"time"
)

// OrderTest is one test the analyzer should run.
type OrderTest struct {
	Code string
	Name string
	Unit string
}

// OrderPatient is the demographic subset the analyzer needs.
type OrderPatient struct {
	HospitalNumber string
	Bed            string
	Name           string
	BirthDate      time.Time
	Sex            string
	BloodType      string
	PatientType    string
	PayType        string
}

// OrderSample is one tube/order returned by a host query.
type OrderSample struct {
	Barcode     string
	SampleID    string
	SampleType  string
	Sender      string
	Department  string
	SendingTime time.Time
	Emergency   bool
	Tests       []OrderTest
}

// BuildQueryAck returns the QCK^Q02 acknowledgement for a host query. When
// found is false the analyzer receives QAK|SR|NF and must not expect a DSR.
func BuildQueryAck(controlID string, found bool, now time.Time) []byte {
	status := "NF"
	if found {
		status = "OK"
	}
	lines := []string{
		msh("QCK^Q02", controlID, now),
		"MSA|AA|" + controlID + "|Message accepted||0",
		"ERR|0",
		"QAK|SR|" + status,
	}
	return []byte(strings.Join(lines, "\r") + "\r")
}

// BuildOrderResponse returns a DSR^Q03 carrying the resolved order. Positions
// follow the Mindray BS-360E/240Pro/240E LIS interface manual, which is the
// provisional source for the BS-240.
func BuildOrderResponse(controlID string, patient OrderPatient, sample OrderSample, now time.Time) []byte {
	lines := []string{
		msh("DSR^Q03", controlID, now),
		"MSA|AA|" + controlID + "|Message accepted||0",
		"ERR|0",
		"QAK|SR|OK",
		"QRD|" + now.Format("20060102150405") + "|R|D|1|||RD|" + sample.Barcode + "|OTH|||T|",
		"QRF|LIS|" + now.Format("20060102150405") + "|" + now.Format("20060102150405") + "||RCT|COR|ALL||",
	}
	values := make([]string, 32)
	values[1] = patient.HospitalNumber
	values[2] = patient.Bed
	values[3] = patient.Name
	if !patient.BirthDate.IsZero() {
		values[4] = patient.BirthDate.Format("20060102150405")
	}
	values[5] = patient.Sex
	values[6] = patient.BloodType
	values[15] = patient.PatientType
	values[17] = patient.PayType
	values[21] = sample.Barcode
	values[22] = sample.SampleID
	if !sample.SendingTime.IsZero() {
		values[23] = sample.SendingTime.Format("20060102150405")
	}
	if sample.Emergency {
		values[24] = "Y"
	} else {
		values[24] = "N"
	}
	values[26] = sample.SampleType
	values[27] = sample.Sender
	values[28] = sample.Department
	for i, test := range sample.Tests {
		if 29+i > 31 {
			break
		}
		values[29+i] = test.Code
	}
	for i := 1; i <= 31; i++ {
		lines = append(lines, fmt.Sprintf("DSP|%d||%s", i, escape(values[i])))
	}
	lines = append(lines, "DSC|")
	return []byte(strings.Join(lines, "\r") + "\r")
}

func msh(messageType, controlID string, now time.Time) string {
	return "MSH|^~\\&|LIS|HMIS|ANALYZER|LAB|" +
		now.Format("20060102150405") + "||" + messageType + "|" +
		controlID + "|P|2.3.1|||ASCII|"
}

func escape(value string) string {
	replacer := strings.NewReplacer(
		`\`, `\E\`,
		`|`, `\F\`,
		`^`, `\S\`,
		`~`, `\R\`,
		`&`, `\T\`,
		"\r", " ",
		"\n", " ",
	)
	return replacer.Replace(value)
}
