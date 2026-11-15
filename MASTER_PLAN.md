# claimguard: Architecture and Engineering Master Plan

Real-time semantic claim verification and anti-hallucination streaming reverse proxy for high-liability LLM deployments.
Target: Single-binary Go reverse proxy (< 25MB RAM, wire-speed streaming) with in-memory SQLite ground truth registry and embedded React 19 verification studio.

## 1. Problem statement and motivation

### 1.1 The liability of factual hallucinations in enterprise AI
When organizations deploy customer-facing generative AI across regulated or commercial domains such as fintech advisory, insurance claims processing, healthcare triage, and e-commerce support, factual hallucinations produce direct financial liability and regulatory penalties:
1. Unsanctioned commercial commitments: Chatbots frequently fabricate discount codes, promise non-existent return windows (for example claiming 60-day full refunds with zero return fees when policy is 14 days), or misquote product pricing tiers.
2. Statutory and regulatory non-compliance: In financial services, providing incorrect interest rate caps or hallucinating regulatory exemptions violates consumer protection rules (such as CFPB and FTC guidelines).
3. Medical and insurance misinformation: Hallucinating deductible thresholds, copay maximums, or approved drug formularies causes patient harm and legal exposure.
4. Post-hoc verification latency: Traditional evaluation frameworks (such as Ragas or TruLens) analyze logs retrospectively after the customer has already received and acted upon the hallucinated answer.

### 1.2 The limits of prompt-based guardrails
System prompt instructions such as "Only state true facts from the database" or "Never make promises you cannot keep" fail under adversarial pressure or complex conversational contexts:
- Attention fading: In long conversations, language models prioritize recent conversational turns over system prompt constraints.
- Plausible fabrication: When uncertain, models prioritize conversational fluency over factual precision, generating falsehoods in an authoritative tone.

### 1.3 The claimguard approach
claimguard is an inline streaming reverse proxy written in Go that operates as a real-time safety airlock between upstream LLM providers and client end-users:
- Wire-speed sentence-level interception: Analyzes Server-Sent Events (SSE) token streams on the fly, segmenting text at sentence boundaries before releasing tokens to the client.
- Fast entity and numerical claim extraction: Uses zero-allocation token scanning and regular expression DFA to isolate verifiable assertions (pricing, dates, policy limits, percentage rates, statutory references).
- Sub-10ms ground truth cross-referencing: Validates extracted claims against a local ground truth registry (in-memory SQLite via modernc.org/sqlite) containing verified enterprise facts.
- Active stream rewriting and disclaimers: If an ungrounded or contradictory claim is detected, claimguard intercepts the stream, replaces the hallucinated sentence with verified truth, or appends a calibrated clarification notice before the client renders it.
- Embedded React 19 verification studio: Provides compliance officers and AI engineers with a real-time audit cockpit displaying verified versus intercepted claims, latency metrics, and audit logs.

## 2. System architecture

```
       +-------------------------------------------------------------+
       |                  End-User Client Application                |
       |                (Web Chat, Mobile App, CRM)                  |
       +-------------------------------------------------------------+
                                      |
                     HTTP / SSE Request (Chat Completions)
                                      v
       +-------------------------------------------------------------+
       |                  claimguard Streaming Proxy                 |
       |                   (High-Performance Go CLI)                |
       |                                                             |
       |  +---------------------+        +------------------------+  |
       |  | Upstream Provider   |------->| Streaming Sentence     |  |
       |  | Client (OpenAI/Anth)|        | Boundary Buffer        |  |
       |  +---------------------+        +------------------------+  |
       |                                              |              |
       |                                              v              |
       |  +-------------------------------------------------------+  |
       |  |          Entity & Numerical Claim Extractor           |  |
       |  |   Extract: Currency ($), Dates, Percentages (%),      |  |
       |  |   Policy Limits, Product Names, Numerical Windows    |  |
       |  +-------------------------------------------------------+  |
       |                               |                             |
       |                               v                             |
       |  +-------------------------------------------------------+  |
       |  |            Ground Truth Verification Engine           |  |
       |  |  Query: In-Memory SQLite Registry                     |  |
       |  |  Compare: Claim Value vs. Authoritative Enterprise Data|  |
       |  +-------------------------------------------------------+  |
       |               /                               \             |
       |       Claim Verified                    Claim Contradicted  |
       |             /                                   \           |
       |            v                                     v          |
       |   +------------------+                 +-----------------+  |
       |   | Release Sentence |                 | Active Rewrite  |  |
       |   | Stream to Client |                 | or Safe Notice  |  |
       |   +------------------+                 +-----------------+  |
       |            \                                     /          |
       |             +-----------------+-----------------+           |
       |                               |                             |
       |                               v                             |
       |  +-------------------------------------------------------+  |
       |  | Cryptographic Audit Event Logger (SHA-256 Checksums)  |  |
       |  +-------------------------------------------------------+  |
       +-------------------------------|-----------------------------+
                                       |
                                       v
       +-------------------------------------------------------------+
       |                 Audited & Grounded Stream Output            |
       |  1. Safe Response Delivered to Client                       |
       |  2. Tamper-Evident Compliance Audit Event Logged to DB      |
       |  3. Embedded React 19 Studio (Local HTTP :9090)             |
       +-------------------------------------------------------------+
```

## 3. Real-time verification mechanics

| Processing stage | Technical mechanism |
| --- | --- |
| 1. Sentence sliding window | Buffers streaming tokens until end-of-sentence punctuation (`.`, `?`, `!`, `\n`) is reached; holds trailing tokens per clause. |
| 2. Claim entity extraction | Identifies numeric assertions, temporal windows, and policy clauses using token scanning and DFA patterns. |
| 3. Registry lookup | Queries indexed enterprise facts using key matching (for example `policy.refund` or `tier.enterprise`). |
| 4. Factual entailment scoring | Compares asserted value with authoritative value; computes deviation delta. |
| 5. Stream rewriter action | If delta is zero, flushes buffer directly. If delta is non-zero, emits corrected sentence, appends disclaimer, or halts. |

### 3.1 Verification formulation
Let S be a completed sentence containing extracted factual claim tuple K = (e, a, v), where e is the entity, a the attribute, and v the asserted value.

Let T(e, a) -> v* represent the authoritative value retrieved from the ground truth registry:

Action(S):
- If v = v* or K is empty: Emit(S)
- If v != v* and policy = rewrite: Rewrite(S, v*)
- If v != v* and policy = disclaimer: AppendCorrection(S, v*)
- If v != v* and policy = strict_halt: HaltAndRefuse()
- If v != v* and policy = flag_only: Emit(S) and RecordAudit(K, v*)

## 4. Repository structure

```
claimguard/
├── cmd/
│   └── claimguard/
│       └── main.go                 # CLI entry point: serve, verify, ingest, audit, demo
├── internal/
│   ├── config/
│   │   ├── config.go               # YAML configuration for upstream LLMs, ports, policies
│   │   └── schema.go               # Thresholds, fallback messages, truth sync schedules
│   ├── proxy/
│   │   ├── handler.go              # Transparent HTTP reverse proxy for /v1/chat/completions
│   │   ├── sse_window.go           # Sliding sentence-boundary window buffer for SSE streams
│   │   ├── upstream.go             # Upstream provider client (OpenAI, Anthropic, Gemini, Ollama)
│   │   ├── simulator.go            # Built-in LLM simulator for offline testing and live demos
│   │   └── benchmark_test.go       # Latency profiling benchmarks
│   ├── extractor/
│   │   ├── parser.go               # Sentence boundary detector and whitespace normalizer
│   │   ├── numerical.go            # Currencies, percentages, dates, metric extractor
│   │   └── entity.go               # Product, policy, and statutory entity matcher
│   ├── registry/
│   │   ├── models.go               # Entity and Fact schemas
│   │   ├── store.go                # In-memory SQLite store for authoritative enterprise facts
│   │   ├── loader.go               # Ingestion pipeline for CSV, JSON, and YAML truth tables
│   │   └── sync.go                 # Periodic background synchronizer with remote data sources
│   ├── verifier/
│   │   ├── engine.go               # Factual entailment and value comparator
│   │   └── tolerance.go            # Rounding and numerical tolerance evaluator
│   ├── rewriter/
│   │   ├── injector.go             # Active stream token re-encoder and patcher
│   │   └── templates.go            # Disclaimer and inline correction templates
│   ├── telemetry/
│   │   ├── audit.go                # Tamper-evident SQLite audit log recording intercepted claims
│   │   └── metrics.go              # Hallucination rate, precision, latency impact tracker
│   └── server/
│       ├── api.go                  # REST endpoints for registry management, metrics, audit logs
│       ├── embed.go                # Embedded React 19 verification studio assets
│       └── sse_live.go             # Live event broadcaster for real-time dashboard updates
├── web/
│   ├── src/
│   │   ├── App.tsx                 # React 19 root application
│   │   ├── components/
│   │   │   ├── HallucinationGauge.tsx # Real-time hallucination interception percentage dial
│   │   │   ├── StreamAuditor.tsx   # Side-by-side stream inspector (Original vs Corrected)
│   │   │   ├── TruthTableEditor.tsx# In-browser editor for enterprise ground truth facts
│   │   │   ├── ClaimFrequency.tsx  # Top hallucinated entities and attributes breakdown
│   │   │   ├── ComplianceLog.tsx   # Filterable audit table for compliance reporting
│   │   │   └── PolicyConfig.tsx    # Dynamic runtime policy controls
│   │   ├── pages/
│   │   │   └── Dashboard.tsx       # Main compliance monitoring dashboard
│   │   ├── types.ts                # TypeScript type definitions
│   │   ├── index.css               # Styling definitions
│   │   ├── main.tsx                # Entry point
│   │   └── dist.go                 # Go embed wrapper for dist bundle
│   ├── package.json
│   ├── tsconfig.json
│   └── vite.config.ts
├── examples/
│   ├── claimguard.yaml             # Example enterprise proxy configuration
│   ├── ground_truth.json           # Sample enterprise truth registry (pricing, return windows, SLAs)
│   └── test_queries.jsonl          # Adversarial prompts designed to test hallucination interception
├── docs/
│   ├── ARCHITECTURE.md             # Detailed engineering specifications
│   ├── COMPLIANCE_EU_AI_ACT.md     # EU AI Act Article 14 and 50 alignment guide
│   └── BENCHMARKS.md               # Latency overhead and throughput profiling results
├── Makefile
├── go.mod
├── go.sum
├── LICENSE
└── README.md
```

## 5. Implementation roadmap and task tracking

### Phase 0: Project setup, repository creation and metadata
- [x] Initialize Git repository with proper .gitignore and MIT license.
- [x] Create public GitHub repository `alexandrmotologa/claimguard`.
- [x] Configure GitHub repository About description and 12 targeted topics across 4 dimensions.
- [x] Author humanized README and foundational documentation.

### Phase 1: Go engine setup and sliding SSE sentence buffer
- [x] Initialize Go module `github.com/alexandrmotologa/claimguard`.
- [x] Build configuration loader (`internal/config`).
- [x] Build streaming reverse proxy in `internal/proxy` intercepting SSE chunk responses.
- [x] Implement `sse_window.go` buffering tokens until complete sentence boundary punctuation is recognized, holding only a single sentence in memory.
- [x] Implement `simulator.go` providing a built-in mock LLM engine with controllable hallucination scenarios for offline execution and tests.

### Phase 2: Ground truth knowledge registry
- [x] Implement SQLite storage using pure Go `modernc.org/sqlite` for sub-millisecond lookups.
- [x] Define database schema for entities, fact attributes, aliases, and source documents.
- [x] Build loader in `internal/registry/loader.go` parsing JSON, YAML, and CSV enterprise truth tables.
- [x] Implement search and match methods with alias resolution.

### Phase 3: Zero-allocation claim extractor
- [x] Implement sentence boundary detector and whitespace normalizer in `internal/extractor/parser.go`.
- [x] Implement `internal/extractor/numerical.go` extracting currencies ($X.XX), percentages, dates, and quantitative values.
- [x] Implement entity and attribute matcher in `internal/extractor/entity.go` resolving product tiers, policies, and terms.

### Phase 4: Real-time verifier and active stream rewriter
- [x] Build `internal/verifier/engine.go` validating extracted claims against the ground truth registry.
- [x] Implement numerical and unit tolerance checking in `internal/verifier/tolerance.go`.
- [x] Implement `internal/rewriter` supporting active policies:
  - `rewrite`: Replaces the hallucinated numeric/factual token sequence in-flight.
  - `disclaimer`: Emits original sentence followed by an immediate corrective disclaimer.
  - `strict_halt`: Terminates stream with a safe refusal when a violation occurs.
  - `flag_only`: Delivers original text while recording an audit incident.

### Phase 5: Compliance audit logging and telemetry
- [x] Log every interception event into SQLite with SHA-256 request checksums, prompt context, and rule details.
- [x] Compute telemetry metrics: total requests, verified claims, interception rate, and latency overhead.
- [x] Build export endpoints for audit logs (JSON/CSV) for regulatory audits.

### Phase 6: REST API and live SSE event broadcaster
- [x] Implement REST handlers in `internal/server/api.go` for registry CRUD, telemetry, and log queries.
- [x] Implement live SSE event broadcaster in `internal/server/sse_live.go` for real-time dashboard updates.

### Phase 7: Embedded React 19 verification studio
- [x] Initialize React 19 project in `web/` with Vite, TypeScript, and Tailwind CSS.
- [x] Build Live Stream Auditor component for interactive testing and live side-by-side stream diffs.
- [x] Build HallucinationGauge and metrics dashboard.
- [x] Build TruthTableEditor for live in-browser fact management.
- [x] Build ComplianceLog table with filters, search, and export.
- [x] Build PolicyConfig component for runtime adjustments.
- [x] Integrate React static assets into Go binary via `embed.FS` served on `:9090`.

### Phase 8: Verification, benchmarking, and documentation
- [x] Write comprehensive unit tests for tokenizer, extractor, registry, and rewriter.
- [x] Run race condition detection tests (`go test -race ./...`).
- [x] Run benchmarks verifying added latency stays under 15ms on streaming connections (~0.1ms measured).
- [x] Complete technical documentation in `docs/` and verify humanizer guidelines throughout.
