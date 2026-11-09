package contract

import "time"

// ContractVersion is the order-resolution contract this gateway speaks.
const Version = 1

// OrderResolveRequest is the barcode lookup sent to the cloud.
type OrderResolveRequest struct {
	ConnectionCode string    `json:"connection_code"`
	Barcode        string    `json:"barcode"`
	Protocol       string    `json:"protocol"`
	QueryMessageID string    `json:"query_message_id"`
	RequestedAt    time.Time `json:"requested_at"`
}

// OrderResolution mirrors the cloud contract without importing cloud code.
type OrderResolution struct {
	Found           bool            `json:"found"`
	ContractVersion int             `json:"contract_version"`
	ExpiresAt       *time.Time      `json:"expires_at,omitempty"`
	Reason          string          `json:"reason,omitempty"`
	Container       *OrderContainer `json:"container,omitempty"`
	Patient         *OrderPatient   `json:"patient,omitempty"`
	Request         *OrderRequest   `json:"request,omitempty"`
	Tests           []OrderTest     `json:"tests,omitempty"`
	Warnings        []string        `json:"warnings,omitempty"`
}

type OrderContainer struct {
	Barcode       string `json:"barcode"`
	ContainerType string `json:"container_type"`
	Label         string `json:"label"`
	Status        string `json:"status"`
}

type OrderPatient struct {
	HospitalNumber string `json:"hospital_number"`
	Name           string `json:"name"`
	Sex            string `json:"sex"`
	BirthDate      string `json:"birth_date"`
	PatientType    string `json:"patient_type"`
	Bed            string `json:"bed"`
}

type OrderRequest struct {
	RequestID        string `json:"request_id"`
	RequestReference string `json:"request_reference"`
	Priority         string `json:"priority"`
}

type OrderTest struct {
	SpecimenID       string `json:"specimen_id"`
	AccessionNumber  string `json:"accession_number"`
	RequestItemID    string `json:"request_item_id"`
	TestCode         string `json:"test_code"`
	TestName         string `json:"test_name"`
	AnalyzerTestCode string `json:"analyzer_test_code"`
	AnalyzerTestName string `json:"analyzer_test_name"`
	SampleType       string `json:"sample_type"`
	Unit             string `json:"unit"`
	Status           string `json:"status"`
}
