# claimguard

Real-time semantic claim verification and anti-hallucination streaming reverse proxy for high-liability LLM applications.

## Overview

claimguard is an inline streaming reverse proxy written in Go that intercepts LLM token streams over Server-Sent Events (SSE). It segments text into sentence boundaries in real time, extracts verifiable assertions (prices, refund windows, interest rates, policy caps, dates), checks them against an in-memory SQLite ground truth registry, and either corrects false claims or appends audit disclaimers before the tokens reach the end user.

The proxy runs as a single binary with an embedded React 19 verification studio. It operates with under 25MB of RAM overhead and adds less than 15 milliseconds of latency to streaming responses.

## Key capabilities

- Streaming sentence boundary buffer: Buffers incoming tokens only until a sentence boundary (`.`, `?`, `!`, `\n`) is reached, minimizing latency impact.
- Zero-allocation claim extractor: Uses finite state scanning to isolate currencies, percentages, temporal windows, and metric bounds without garbage collection pressure.
- Ground truth registry: Backed by in-memory SQLite using `modernc.org/sqlite`, providing sub-millisecond fact lookup times.
- Dynamic action policies:
  - `rewrite`: Replaces hallucinated figures and statements in-flight before the client receives them.
  - `disclaimer`: Appends an immediate factual clarification notice right after the contradictory sentence.
  - `strict_halt`: Terminates the stream with an explicit compliance refusal when a critical fact violation occurs.
  - `flag_only`: Delivers the original stream while recording a tamper-evident audit event.
- Cryptographic compliance ledger: Every verified and intercepted claim is hashed with SHA-256 and logged to a local SQLite database for compliance auditing (EU AI Act Article 14 and 50).
- Embedded React 19 studio: Includes a live stream auditor, interactive prompt test console, visual hallucination gauge, ground truth editor, and audit log viewer.
- Built-in simulation mode: Includes a mock LLM engine with controllable hallucination scenarios for automated testing and offline evaluations without API keys.

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

## Quick start

### Prerequisites

- Go 1.22 or higher
- Node.js 20 or higher (for building the web studio)

### Installation

Clone the repository and build the binary:

```bash
git clone https://github.com/alexandrmotologa/claimguard.git
cd claimguard
make build
```

Alternatively, build with Go directly:

```bash
go build -o bin/claimguard ./cmd/claimguard
```

### Running with the built-in simulator

To test claimguard without an external LLM API key, run the built-in simulation mode:

```bash
./bin/claimguard serve --config examples/claimguard.yaml --mode demo
```

This starts:
- The proxy server on `http://localhost:8080`
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

The mock model attempts to state "We offer a 60-day full money-back guarantee with zero restocking fees."
claimguard intercepts the sentence and rewrites it to:
"Our standard policy provides a 14-day refund window with standard return conditions."

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
claimguard_average_latency_ms 0.2600
```

## Testing

Run unit tests and verification benchmarks:

```bash
go test -v -race ./...
go test -bench=. -benchmem ./internal/...
```

## License

MIT License. See [LICENSE](LICENSE) for details.
