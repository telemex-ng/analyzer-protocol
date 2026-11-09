package astm

import (
	"fmt"
	"strings"
	"time"
)

// OrderTest is one assay in an O record.
type OrderTest struct {
	Code string
	Name string
}

// OrderPatient is the demographic subset the analyzer needs.
type OrderPatient struct {
	ID        string
	Name      string
	BirthDate time.Time
	Sex       string
}

// OrderSample is one tube/order returned for a Q record.
type OrderSample struct {
	Sequence    int
	SampleID    string
	Barcode     string
	SampleType  string
	Priority    string
	RequestedAt time.Time
	CollectedAt time.Time
	Sender      string
	Department  string
	Tests       []OrderTest
}

// BuildOrderResponse returns an H/P/O/L order download. Multiple samples are
// returned as H, P, O, P, O, ..., L with at most ten P records.
func BuildOrderResponse(patient OrderPatient, orders []OrderSample, now time.Time) []byte {
	lines := []string{
		"H|\\^&|||LIS^01.00^HMIS|||||||P|1394-97|" + now.Format("20060102150405"),
	}
	for _, order := range orders {
		lines = append(lines, buildPatient(patient))
		lines = append(lines, buildOrder(order))
	}
	lines = append(lines, "L|1|N")
	return []byte(strings.Join(lines, "\r") + "\r")
}

func buildPatient(patient OrderPatient) string {
	birth := ""
	if !patient.BirthDate.IsZero() {
		birth = patient.BirthDate.Format("20060102")
	}
	parts := []string{
		"P", "1", "", patient.ID, "", patient.Name, "", birth, patient.Sex,
		"", "A", "", "", "", "", "", "", "", "", "01", "", "", "", "A1", "002",
	}
	return strings.Join(parts, "|")
}

func buildOrder(order OrderSample) string {
	assays := make([]string, 0, len(order.Tests)*4)
	for i, test := range order.Tests {
		name := test.Name
		if name == "" {
			name = test.Code
		}
		assays = append(assays,
			fmt.Sprintf("%d", i+1), name, "1", "1")
	}
	sampleID := order.SampleID
	if sampleID == "" {
		sampleID = "1^1^1"
	}
	barcode := order.Barcode
	if barcode == "" {
		barcode = order.SampleID
	}
	priority := order.Priority
	if priority == "" {
		priority = "R"
	}
	requested := ""
	if !order.RequestedAt.IsZero() {
		requested = order.RequestedAt.Format("20060102150405")
	}
	collected := ""
	if !order.CollectedAt.IsZero() {
		collected = order.CollectedAt.Format("20060102150405")
	}
	parts := []string{
		"O",
		fmt.Sprintf("%d", order.Sequence),
		sampleID,
		barcode,
		strings.Join(assays, "^"),
		priority,
		requested,
		collected,
		"", "", order.Sender, "", "", "", order.SampleType, "", order.Department,
		"1", "", "", "", "", "", "F",
	}
	return strings.Join(parts, "|")
}
