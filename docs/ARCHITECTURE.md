# System architecture

This document details the internal design and streaming data path of claimguard.

## Data flow and lifecycle

The core lifecycle of a request through claimguard consists of six stages:

1. Request intake: The client initiates a standard OpenAI-compatible `/v1/chat/completions` request with `stream: true`.
2. Upstream forwarding: The proxy forwards the payload to the upstream LLM provider (or built-in simulator) and establishes an SSE response reader.
3. Sentence windowing: Incoming SSE chunks (`data: {"choices":[{"delta":{"content":"..."}}]}`) are parsed into text fragments and accumulated into a sliding sentence buffer until a terminator character (`.`, `?`, `!`, `\n`) or paragraph boundary is reached.
4. Claim extraction: The completed sentence is evaluated by the claim extractor. The extractor isolates entities (such as product names or policy tiers), attributes (such as pricing, return windows, SLAs, or interest rates), and numerical or categorical values.
5. Ground truth lookup and verification: The extracted entity-attribute pair is checked against the in-memory SQLite knowledge base. If an authoritative fact exists:
   - If the asserted value matches the authoritative value within configured tolerance, the sentence is marked verified.
   - If the asserted value differs from the authoritative fact, a violation is recorded.
6. Transformation and release:
   - Under `rewrite`: The extractor replaces the contradicted tokens with authoritative values and emits synthetic SSE chunks.
   - Under `disclaimer`: The original sentence is flushed followed by an injected disclaimer SSE chunk.
   - Under `strict_halt`: The SSE stream is immediately aborted with a refusal message and closed.
   - Under `flag_only`: The original sentence is flushed unchanged.
   - At stream completion, a tamper-evident audit record containing the SHA-256 hash of the transaction is written to the audit database.

## Sliding sentence buffer design

Traditional streaming buffers either emit token-by-token (making semantic verification impossible before delivery) or buffer entire responses (destroying the real-time user experience).

claimguard implements an intermediate sliding sentence buffer:
- The buffer holds only the active clause. Once a punctuation mark followed by whitespace or uppercase letter arrives, the clause is released.
- Typical sentence lengths in conversational responses range from 10 to 25 tokens. At standard LLM generation speeds (30 to 60 tokens per second), sentence accumulation takes between 150ms and 400ms.
- The incremental latency introduced by the verification logic itself is sub-millisecond, as SQLite lookups operate directly in memory without disk I/O.

## Ground truth knowledge model

The registry schema supports structured assertions with optional tolerances and aliases:

```sql
CREATE TABLE IF NOT EXISTS entities (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    category TEXT NOT NULL,
    aliases_json TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS facts (
    id TEXT PRIMARY KEY,
    entity_id TEXT NOT NULL REFERENCES entities(id),
    attribute TEXT NOT NULL,
    claim_type TEXT NOT NULL, -- currency, duration, percentage, text
    expected_value TEXT NOT NULL,
    numeric_value REAL,
    unit TEXT,
    tolerance REAL DEFAULT 0.0,
    policy_action TEXT,
    source_doc TEXT,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

Aliases enable fuzzy entity resolution. For instance, `starter`, `basic tier`, and `standard tier` resolve to the same underlying entity `tier.starter`.

## Cryptographic audit chain

Every intervention produces a tamper-evident audit log with:
- Request ID and timestamp
- Upstream model name
- Original sentence text
- Extracted claim tuple (entity, attribute, asserted value)
- Ground truth authoritative value
- Applied action (`rewrite`, `disclaimer`, `strict_halt`)
- Corrected sentence text
- SHA-256 hash of the entire event payload

This audit trail supports regulatory verification under EU AI Act Article 14 (human oversight and technical guardrails) and Article 50 (transparency).
