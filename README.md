# Telemex analyzer protocol

Shared Go module for the analyzer wire protocol used by both the Telemex cloud
API and the on-prem analyzer gateway.

This repository is the single source of truth for:

- HL7 v2 parsing and host-query/order-response builders
- ASTM E1394 parsing and host-query/order-response builders
- MLLP framing
- analyzer value, range and flag parsing
- the order-resolution contract types and wire-contract version
- golden protocol fixtures shared by both consumers

It deliberately contains no database code, no cloud authentication, no
listeners, no spool and no business rules.

## Import

```go
import (
    "github.com/telemex-ng/analyzer-protocol/contract"
    "github.com/telemex-ng/analyzer-protocol/hl7"
    "github.com/telemex-ng/analyzer-protocol/astm"
    "github.com/telemex-ng/analyzer-protocol/mllp"
    "github.com/telemex-ng/analyzer-protocol/values"
)
```

Consumers pin a released version:

```go
require github.com/telemex-ng/analyzer-protocol v0.1.0
```

The repository is private. Developers and CI need read access and:

```bash
export GOPRIVATE=github.com/telemex-ng/*
```

## Versions

There are two version numbers:

- **module version** — the Git tag, for example `v1.2.0`
- **wire-contract version** — `contract.Version`, currently `1`

A parser fix bumps the module patch version. A wire-format change bumps
`contract.Version` and requires a new module major version.

## Test

```bash
go test ./... -count=1
```

`fixtures` embeds the golden HL7 and ASTM frames so the cloud, the gateway and
this module all test against exactly the same bytes.
