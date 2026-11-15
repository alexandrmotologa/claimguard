# EU AI Act compliance mapping

This document describes how claimguard maps to the requirements of the European Union Artificial Intelligence Act (Regulation EU 2024/1689), specifically focusing on Articles 14, 15, and 50 for high-liability and customer-facing generative AI systems.

## Relevant regulatory articles

### Article 14: Human oversight and technical safeguards
High-risk AI systems must be designed with appropriate technical interfaces and operational guardrails to prevent or minimize risks to health, safety, or fundamental rights.

claimguard provides:
- Real-time intervention: Intercepts factual departures before incorrect assertions are committed to end users.
- Deterministic overrides: Business policies in the ground truth registry take precedence over non-deterministic probabilistic outputs.
- Operator controls: Compliance officers can switch action modes (`rewrite`, `disclaimer`, `strict_halt`) dynamically through the verification studio or REST API.

### Article 15: Accuracy, robustness, and cybersecurity
AI systems must achieve an appropriate level of accuracy and resilience against errors, inconsistencies, and adversarial inputs.

claimguard provides:
- Numerical accuracy checks: Prevents hallucinated numbers in contracts, interest rates, refund windows, and loan limits.
- Tamper-evident logging: Audit entries include SHA-256 integrity hashes to record the exact transformations applied to model outputs.
- Fail-safe options: When configured with `strict_halt`, high-severity discrepancies immediately abort the session with an audit record.

### Article 50: Transparency obligations for providers and deployers
Deployers of generative AI systems must ensure that outputs are clearly identified as machine-generated and accompanied by appropriate disclosures when necessary.

claimguard provides:
- Automated disclaimer injection: Under the `disclaimer` policy, corrected claims are visibly flagged with explicit corrections.
- Audit export: Generates structured CSV and JSON audit exports for internal compliance reviews and external regulatory audits.

## Operational implementation checklist

1. Ingest official enterprise facts into the ground truth registry (`claimguard ingest --file ground_truth.json`).
2. Set default action policy to `rewrite` or `disclaimer` depending on domain disclosure requirements.
3. Establish daily or weekly truth table sync routines to update active pricing, terms, and regulatory formulas.
4. Export audit logs regularly (`claimguard audit export --format json --output audit_export.json`) for corporate compliance archiving.
