// Package fixtures exposes the golden protocol frames used by the cloud,
// gateway and protocol tests. Keeping them embedded means every consumer tests
// against exactly the same bytes without copying testdata between repositories.
package fixtures

import _ "embed"

//go:embed hl7/query.hl7
var HL7Query []byte

//go:embed hl7/order-response.hl7
var HL7OrderResponse []byte

//go:embed astm/query.astm
var ASTMQuery []byte

//go:embed astm/order-response.astm
var ASTMOrderResponse []byte
