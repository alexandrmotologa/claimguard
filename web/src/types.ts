export interface Entity {
  id: string
  name: string
  category: string
  aliases: string[]
}

export interface Fact {
  id: string
  entity_id: string
  attribute: string
  claim_type: string
  expected_value: string
  numeric_value: number
  unit: string
  tolerance: number
  policy_action: string
  description: string
  source_doc: string
  updated_at?: string
}

export interface AuditEvent {
  id: string
  timestamp: string
  request_id: string
  prompt_preview: string
  original_sentence: string
  corrected_sentence: string
  entity_id: string
  attribute: string
  asserted_value: string
  authoritative_value: string
  policy_action: string
  status: 'VERIFIED' | 'CONTRADICTED' | 'UNGROUNDED'
  latency_ms: number
  payload_hash: string
}

export interface MetricsSnapshot {
  uptime_seconds: number
  total_requests: number
  total_tokens: number
  total_sentences: number
  total_claims_verified: number
  total_contradictions: number
  interception_rate_pct: number
  average_latency_ms: number
  interventions_by_action: Record<string, number>
  violations_by_entity: Record<string, number>
}

export interface PolicyConfigData {
  default_action: string
  disclaimer_template: string
  strict_halt_message: string
  numerical_tolerance_percent: number
}

export interface ExtractedClaim {
  raw_sentence: string
  entity_id: string
  entity_name: string
  attribute: string
  claim_type: string
  asserted_value: string
  numeric_value: number
  unit: string
}

export interface VerificationResult {
  status: 'VERIFIED' | 'CONTRADICTED' | 'UNGROUNDED'
  claim: ExtractedClaim
  authoritative_fact?: Fact
  delta: number
  applied_action: string
  audit_message: string
  corrected_sentence?: string
}

export interface DryRunResponse {
  original_sentence: string
  claims: ExtractedClaim[]
  results: VerificationResult[]
  transformed_sentence: string
  primary_action: string
  should_halt: boolean
}

export interface ForbiddenPattern {
  id: string
  name: string
  pattern: string
  category: string
  policy_action: string
  replacement?: string
  reason: string
  source_doc?: string
  updated_at?: string
}

export interface BenchmarkScenarioResult {
  scenario_id: string
  domain: string
  prompt: string
  target_entity: string
  raw_output: string
  guarded_output: string
  was_intercepted: boolean
  action_taken: string
  status: 'PASSED' | 'FAILED'
  latency_ms: number
}

export interface BenchmarkReport {
  timestamp: string
  suite_name: string
  total_scenarios: number
  passed_scenarios: number
  compliance_accuracy_score: number
  average_latency_ms: number
  certification_hash: string
  scenarios: BenchmarkScenarioResult[]
}
