<p align="center">
  <img src="docs/images/logo.png?raw=true" alt="claimguard Logo" width="130" style="border-radius: 24px;" />
</p>

<h1 align="center">claimguard</h1>

<p align="center">
  <strong>Wire-speed semantic claim verification and anti-hallucination streaming reverse proxy for high-liability LLM applications.</strong>
</p>

<p align="center">
  <a href="https://github.com/alexandrmotologa/claimguard/actions"><img src="https://img.shields.io/badge/build-passing-brightgreen?style=flat-square" alt="Build" /></a>
  <a href="https://golang.org"><img src="https://img.shields.io/badge/go-1.22%2B-blue?style=flat-square&logo=go" alt="Go" /></a>
  <a href="https://react.dev"><img src="https://img.shields.io/badge/studio-React%2019-61dafb?style=flat-square&logo=react" alt="React 19" /></a>
  <a href="#observability-and-metrics"><img src="https://img.shields.io/badge/latency-%3C%200.5ms-emerald?style=flat-square" alt="Latency" /></a>
  <a href="#compliance-and-governance"><img src="https://img.shields.io/badge/compliance-EU%20AI%20Act%20Art.%2014%2F50-purple?style=flat-square" alt="Compliance" /></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-slate?style=flat-square" alt="License" /></a>
</p>

<p align="center">
  <img src="docs/images/claimguard_demo.gif?raw=true" alt="claimguard Live Stream Interception Demo" width="920" style="border-radius: 12px; border: 1px solid #1e293b;" />
</p>

## Overview

claimguard is an inline streaming reverse proxy written in Go that intercepts LLM token streams over Server-Sent Events (SSE). It segments text into sentence boundaries in real time, extracts verifiable assertions (prices, refund windows, interest rates, policy caps, dates), checks them against an in-memory SQLite ground truth registry, and either corrects false claims or appends audit disclaimers before the tokens reach the end user.

The proxy runs as a single binary with an embedded React 19 verification studio. It operates with under 25MB of RAM overhead and adds less than 0.5 milliseconds of latency to streaming responses.

## Key capabilities

- **Streaming sentence boundary buffer**: Buffers incoming tokens only until a sentence boundary (`.`, `?`, `!`, `\n`) is reached, minimizing latency impact.
- **Zero-allocation claim extractor**: Uses finite state scanning to isolate currencies, percentages, temporal windows, and metric bounds without garbage collection pressure.
- **Ground truth registry**: Backed by in-memory SQLite using `modernc.org/sqlite`, providing sub-millisecond fact lookup times.
- **Dynamic action policies**:
  - `rewrite`: Replaces hallucinated figures and statements in-flight before the client receives them.
  - `disclaimer`: Appends an immediate factual clarification notice right after the contradictory sentence.
  - `strict_halt`: Terminates the stream with an explicit compliance refusal when a critical fact violation occurs.
  - `flag_only`: Delivers the original stream while recording a tamper-evident audit event.
- **Cryptographic compliance ledger**: Every verified and intercepted claim is hashed with SHA-256 and logged to a local SQLite database for compliance auditing (EU AI Act Article 14 and 50).
- **Embedded React 19 studio**: Includes a live stream auditor, interactive prompt test console, visual hallucination gauge, ground truth editor, and audit log viewer.
- **Built-in simulation mode**: Includes a mock LLM engine with controllable hallucination scenarios for automated testing and offline evaluations without API keys.

## Architecture

```
Client (Web / Mobile / API)
       |
       | HTTP / SSE Request (/v1/chat/completions)
       v
claimguard Streaming Proxy (:8080)
  ├── 1. Upstream Provider Client (OpenAI, Anthropic, Gemini, Ollama, Simulator)
  ├── 2. Sliding Sentence Buffer (sentence-by-sentence window)
  ├── 3. Claim Extractor (currency, percentages, durations, product tiers)
  ├── 4. Ground Truth Registry (in-memory SQLite facts)
  ├── 5. Stream Rewriter (rewrite / disclaimer / strict halt)
  └── 6. Cryptographic Audit Logger (SHA-256 event log)
       |
       +---> Audited SSE Stream to Client
       +---> Live Event Feed (:9090/api/events/live)
       +---> Embedded React 19 Studio (:9090)
```

## User interface and verification studio

claimguard embeds a React 19 single-page verification studio directly into the compiled Go binary. The studio operates with full responsiveness across desktop, tablet, and mobile displays.

### Audit cockpit

The audit cockpit provides a real-time view of intercepted token streams, live hallucination metrics, added latency, and violation distributions across enterprise domains.

<p align="center">
  <img src="docs/images/dashboard_cockpit.png?raw=true" alt="claimguard Audit Cockpit" width="920" style="border-radius: 10px; border: 1px solid #1e293b;" />
</p>

### Word-level diff inspection

Inspect in-place token substitutions side-by-side or as inline diffs with full rule provenance and expected values.

<p align="center">
  <img src="docs/images/stream_auditor_diff.png?raw=true" alt="claimguard Word-Level Diff Inspection" width="920" style="border-radius: 10px; border: 1px solid #1e293b;" />
</p>

### Cryptographic compliance ledger

Tamper-evident SHA-256 audit log tracking every LLM assertion, ground truth comparison, applied policy action, and execution latency for regulatory compliance.

<p align="center">
  <img src="docs/images/compliance_ledger.png?raw=true" alt="claimguard Compliance Audit Ledger" width="920" style="border-radius: 10px; border: 1px solid #1e293b;" />
</p>

### Ground truth knowledge registry

Inspect, add, and modify authoritative enterprise facts, numerical bounds, and forbidden invariant rules in SQLite with zero cold-start latency.

<p align="center">
  <img src="docs/images/ground_truth_registry.png?raw=true" alt="claimguard Ground Truth Knowledge Registry" width="920" style="border-radius: 10px; border: 1px solid #1e293b;" />
</p>

### Automated red-team compliance benchmark

Run automated penetration test suites against upstream models to verify policy enforcement and generate signed compliance certificates.

<p align="center">
  <img src="docs/images/redteam_benchmark.png?raw=true" alt="claimguard Automated Red-Team Compliance Benchmark" width="920" style="border-radius: 10px; border: 1px solid #1e293b;" />
</p>

### Mobile responsiveness

The verification studio provides full operational control on mobile and tablet devices with touch-friendly navigation and responsive layout scales.

<p align="center">
  <img src="docs/images/mobile_cockpit.png?raw=true" alt="claimguard Mobile Viewport" width="380" style="border-radius: 16px; border: 1px solid #1e293b;" />
</p>

## Quick start

### Prerequisites

- Go 1.22 or higher
- Node.js 20 or higher (only needed if modifying the web studio)

### Installation

Clone the repository and build the binary:

```bash
git clone https://github.com/alexandrmotologa/claimguard.git
cd claimguard
go build -o bin/claimguard ./cmd/claimguard
```

### Running with the built-in simulator

To test claimguard without an external LLM API key, run the built-in simulation mode:

```bash
./bin/claimguard serve --config examples/claimguard.yaml --mode demo
```

This starts:
- The streaming reverse proxy on `http://localhost:8080`
- The React 19 verification studio on `http://localhost:9090`
- A mock upstream that generates intentional commercial hallucinations to demonstrate real-time interception

### Testing an intercepted request

Send a prompt asking for an unauthorized return window:

```bash
curl -N http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o",
    "stream": true,
    "messages": [
      {"role": "user", "content": "What is your refund policy window?"}
    ]
  }'
```

The upstream model attempts to state "We offer a 60-day return policy for all items, ensuring a full refund with zero restocking fees."
claimguard intercepts the sentence and rewrites it to:
"We offer a 14-day return policy for all items, ensuring a full refund with zero restocking fees."

## Configuration

Configuration is managed via a YAML file (`examples/claimguard.yaml`):

```yaml
server:
  proxy_port: 8080
  studio_port: 9090
  read_timeout_seconds: 30
  write_timeout_seconds: 60

upstream:
  provider: "simulator" # options: simulator, openai, anthropic, ollama, custom
  base_url: "https://api.openai.com/v1"
  api_key: "${OPENAI_API_KEY}"
  default_model: "gpt-4o"

policy:
  default_action: "rewrite" # options: rewrite, disclaimer, strict_halt, flag_only
  disclaimer_template: " [Correction: Official enterprise policy is {{.ExpectedValue}}, not {{.AssertedValue}}]"
  strict_halt_message: "Stream terminated by compliance filter: factual contradiction detected."
  numerical_tolerance_percent: 0.0

registry:
  db_path: ":memory:"
  initial_truth_file: "examples/ground_truth.json"

telemetry:
  audit_db_path: "claimguard_audit.db"
  retention_days: 90
```

## CLI reference

```text
claimguard - Real-time semantic claim verification proxy

Usage:
  claimguard [command]

Available Commands:
  serve       Start the streaming proxy server and verification studio
  verify      Test a single statement against the ground truth registry
  ingest      Load facts from JSON, YAML, or CSV into the registry
  audit       Query, verify, or export compliance audit records
  scan        Execute automated red-teaming compliance scan
  demo        Run a self-contained live demonstration in the terminal
  help        Help about any command
```

## Observability and metrics

claimguard exports standard Prometheus metrics on both `:8080/metrics` and `:9090/metrics`:

```text
# HELP claimguard_requests_total Total number of chat completion requests
# TYPE claimguard_requests_total counter
claimguard_requests_total 42

# HELP claimguard_contradictions_total Total intercepted factual contradictions
# TYPE claimguard_contradictions_total counter
claimguard_contradictions_total 7

# HELP claimguard_average_latency_ms Average sentence interception latency in milliseconds
# TYPE claimguard_average_latency_ms gauge
claimguard_average_latency_ms 0.4200
```

## Compliance and governance

claimguard serves as a technical control layer for regulatory frameworks:
- **EU AI Act Article 14 (Human Oversight & Technical Governance)**: Enforces real-time intervention boundaries before automated outputs reach consumers.
- **EU AI Act Article 50 (Transparency Obligations)**: Automatically injects factual disclaimers and flags AI-generated claims requiring correction.
- **Financial and Healthcare Liability (CFPB / HIPAA)**: Prevents numerical commitments exceeding regulatory caps (APR limits, deductible schedules, SLA minimums).

## Testing

Run unit tests and verification benchmarks:

```bash
go test -v -race ./...
go test "-bench=." -benchmem ./internal/...
```

## License

MIT License. See [LICENSE](LICENSE) for details.
