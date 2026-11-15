import React, { useState } from 'react'
import { Send, Sparkles, CheckCircle2, AlertOctagon, ArrowRight, RefreshCw, Columns2, AlignLeft } from 'lucide-react'
import { VerificationResult } from '../types'

const SAMPLE_PROMPTS = [
  { label: 'Return Policy', prompt: 'Can I return items within 60 days if I change my mind?' },
  { label: 'Enterprise Pricing', prompt: 'How much does the Enterprise tier cost per month?' },
  { label: 'Starter Plan', prompt: 'What is the cost of the starter plan?' },
  { label: 'Loan APR', prompt: 'What is the maximum interest rate APR you charge on personal loans?' },
  { label: 'Medical Deductible', prompt: 'What is my annual in-network deductible for medical claims?' },
  { label: 'SLA Uptime', prompt: 'What uptime percentage do you legally guarantee in the SLA contract?' },
]

interface GuardedSentenceEvent {
  raw_sentence: string
  guarded_sentence: string
  results: VerificationResult[]
  primary_action: string
  should_halt: boolean
}

export const StreamAuditor: React.FC = () => {
  const [prompt, setPrompt] = useState('Can I return items within 60 days if I change my mind?')
  const [loading, setLoading] = useState(false)
  const [rawOutput, setRawOutput] = useState<string>('')
  const [guardedOutput, setGuardedOutput] = useState<string>('')
  const [allResults, setAllResults] = useState<VerificationResult[]>([])
  const [isStreaming, setIsStreaming] = useState(false)
  const [viewMode, setViewMode] = useState<'split' | 'inline'>('split')

  const handleRunTest = async () => {
    if (!prompt.trim() || loading) return
    setLoading(true)
    setIsStreaming(true)
    setRawOutput('')
    setGuardedOutput('')
    setAllResults([])

    try {
      const response = await fetch('/api/stream/compare', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ prompt, model: 'gpt-4o' }),
      })

      if (!response.body) {
        throw new Error('No readable stream available')
      }

      const reader = response.body.getReader()
      const decoder = new TextDecoder()
      let streamBuffer = ''
      let currentEvent = ''
      let accumulatedGuarded = ''
      const collectedResults: VerificationResult[] = []

      while (true) {
        const { value, done } = await reader.read()
        if (done) break

        streamBuffer += decoder.decode(value, { stream: true })
        const lines = streamBuffer.split('\n')
        // Keep the last partial line in the buffer
        streamBuffer = lines.pop() || ''

        for (const line of lines) {
          const trimmed = line.trim()
          if (!trimmed) {
            currentEvent = ''
            continue
          }

          if (trimmed.startsWith('event: ')) {
            currentEvent = trimmed.replace('event: ', '').trim()
          } else if (trimmed.startsWith('data: ')) {
            const dataStr = trimmed.replace('data: ', '').trim()

            if (currentEvent === 'raw') {
              try {
                const parsed = JSON.parse(dataStr)
                if (parsed.delta) {
                  setRawOutput((prev) => prev + parsed.delta)
                }
              } catch {
                // Ignore chunk parse error
              }
            } else if (currentEvent === 'guarded') {
              try {
                const ev: GuardedSentenceEvent = JSON.parse(dataStr)
                accumulatedGuarded += (accumulatedGuarded ? ' ' : '') + ev.guarded_sentence
                setGuardedOutput(accumulatedGuarded)
                if (ev.results && ev.results.length > 0) {
                  collectedResults.push(...ev.results)
                  setAllResults([...collectedResults])
                }
              } catch {
                // Ignore chunk parse error
              }
            } else if (currentEvent === 'done') {
              setIsStreaming(false)
            }
          }
        }
      }
    } catch (err) {
      console.error('Stream audit failed:', err)
      setGuardedOutput('Error: Unable to connect to streaming proxy on localhost:8080.')
    } finally {
      setLoading(false)
      setIsStreaming(false)
    }
  }

  const renderInlineDiff = () => {
    if (!guardedOutput && !rawOutput) {
      return <span className="text-slate-500 italic">Run a test to inspect inline word-level diffs...</span>
    }

    if (allResults.length === 0) {
      return (
        <div className="p-3 rounded bg-surface border border-border text-slate-300">
          <div className="flex items-center gap-1.5 text-xs text-emerald-400 font-medium mb-2">
            <CheckCircle2 className="w-4 h-4" />
            <span>All Assertions Compliant &bull; Zero Hallucinations Detected</span>
          </div>
          <p className="text-sm font-sans text-slate-100">{guardedOutput || rawOutput}</p>
        </div>
      )
    }

    return (
      <div className="space-y-3 font-mono text-xs leading-relaxed">
        <div className="p-3 rounded bg-surface border border-border text-slate-300">
          <span className="text-slate-400 block text-[11px] uppercase font-semibold mb-2">
            Guarded Sentence with Highlighted Interventions:
          </span>
          {allResults.map((r, i) => (
            <div key={i} className="mb-2 flex flex-wrap items-center gap-2">
              <span className="line-through text-red-400 bg-red-500/10 px-1.5 py-0.5 rounded">
                {r.claim.asserted_value}
              </span>
              <ArrowRight className="w-3 h-3 text-slate-500 inline" />
              <span className="text-emerald-400 font-semibold bg-emerald-500/10 px-1.5 py-0.5 rounded">
                {r.authoritative_fact?.expected_value || r.applied_action}
              </span>
              <span className="text-slate-400 text-[11px]">
                ({r.claim.entity_id}.{r.claim.attribute} &bull; Rule: {r.applied_action})
              </span>
            </div>
          ))}
          <div className="pt-2 border-t border-border/60 text-slate-100 font-sans text-sm">
            {guardedOutput}
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="bg-surface border border-border rounded-xl p-4 sm:p-6">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 mb-4">
        <div>
          <h2 className="text-base sm:text-lg font-semibold text-white">Live Stream Auditor</h2>
          <p className="text-xs text-slate-400 mt-0.5">
            Intercept and inspect streaming LLM token assertions sentence-by-sentence before client delivery.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-3">
          {isStreaming && (
            <span className="flex items-center gap-1.5 text-xs text-amber-400 font-medium bg-amber-400/10 px-2.5 py-1 rounded-full border border-amber-400/20">
              <span className="w-2 h-2 rounded-full bg-amber-400 animate-ping" />
              Streaming &amp; Verifying...
            </span>
          )}

          {/* View Toggle */}
          <div className="flex items-center bg-surface-elevated border border-border rounded-lg p-0.5 text-xs">
            <button
              onClick={() => setViewMode('split')}
              className={`flex items-center gap-1.5 px-2.5 py-1 rounded-md transition-colors ${
                viewMode === 'split' ? 'bg-blue-600 text-white font-medium' : 'text-slate-400 hover:text-white'
              }`}
            >
              <Columns2 className="w-3.5 h-3.5" />
              <span>Side-by-Side</span>
            </button>
            <button
              onClick={() => setViewMode('inline')}
              className={`flex items-center gap-1.5 px-2.5 py-1 rounded-md transition-colors ${
                viewMode === 'inline' ? 'bg-blue-600 text-white font-medium' : 'text-slate-400 hover:text-white'
              }`}
            >
              <AlignLeft className="w-3.5 h-3.5" />
              <span>Inline Diff</span>
            </button>
          </div>
        </div>
      </div>

      {/* Preset Scenario Pills */}
      <div className="flex flex-wrap items-center gap-2 mb-4">
        <span className="text-xs text-slate-400 font-medium">Test Presets:</span>
        {SAMPLE_PROMPTS.map((p, idx) => (
          <button
            key={idx}
            onClick={() => setPrompt(p.prompt)}
            className="text-xs px-2.5 py-1 rounded-md bg-surface-elevated hover:bg-slate-700 text-slate-300 border border-border transition-colors"
          >
            {p.label}
          </button>
        ))}
      </div>

      {/* Prompt Input Box */}
      <div className="flex flex-col sm:flex-row gap-3 mb-6">
        <input
          type="text"
          value={prompt}
          onChange={(e) => setPrompt(e.target.value)}
          placeholder="Ask a question with commercial, pricing, or policy liability..."
          className="flex-1 bg-surface-elevated border border-border rounded-lg px-4 py-2.5 text-sm text-white placeholder-slate-500 focus:outline-none focus:border-blue-500"
          onKeyDown={(e) => e.key === 'Enter' && handleRunTest()}
        />
        <button
          onClick={handleRunTest}
          disabled={loading}
          className="flex items-center justify-center gap-2 px-5 py-2.5 bg-blue-600 hover:bg-blue-500 disabled:opacity-50 text-white text-sm font-medium rounded-lg transition-colors shadow-sm whitespace-nowrap"
        >
          {loading ? (
            <RefreshCw className="w-4 h-4 animate-spin" />
          ) : (
            <Send className="w-4 h-4" />
          )}
          <span>Send to Proxy</span>
        </button>
      </div>

      {/* View Modes */}
      {viewMode === 'split' ? (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          {/* Left: Raw LLM Output (Before Interception) */}
          <div className="bg-surface-elevated border border-border rounded-lg p-4">
            <div className="flex items-center justify-between pb-2 mb-3 border-b border-border">
              <span className="text-xs uppercase tracking-wider font-semibold text-amber-400 flex items-center gap-1.5">
                <AlertOctagon className="w-3.5 h-3.5" />
                Raw Model Generation (Upstream)
              </span>
              <span className="text-[11px] text-slate-400">Unfiltered tokens</span>
            </div>
            <div className="text-sm text-slate-300 leading-relaxed min-h-[120px] font-mono text-xs whitespace-pre-wrap">
              {rawOutput ? (
                <span className="text-slate-300">{rawOutput}</span>
              ) : (
                <span className="text-slate-500 italic">Click &quot;Send to Proxy&quot; to test stream interception.</span>
              )}
            </div>
          </div>

          {/* Right: Guarded Output (After claimguard Interception) */}
          <div className="bg-surface-elevated border border-border rounded-lg p-4">
            <div className="flex items-center justify-between pb-2 mb-3 border-b border-border">
              <span className="text-xs uppercase tracking-wider font-semibold text-emerald-400 flex items-center gap-1.5">
                <CheckCircle2 className="w-3.5 h-3.5" />
                claimguard Guarded Stream (Delivered)
              </span>
              <span className="text-[11px] text-emerald-400 font-medium">Sub-10ms active patch</span>
            </div>
            <div className="text-sm text-slate-100 leading-relaxed min-h-[120px] font-mono text-xs whitespace-pre-wrap">
              {guardedOutput ? (
                <span className="text-slate-100">{guardedOutput}</span>
              ) : (
                <span className="text-slate-500 italic">Awaiting guarded stream output...</span>
              )}
            </div>
          </div>
        </div>
      ) : (
        <div className="bg-surface-elevated border border-border rounded-lg p-4">
          <div className="flex items-center justify-between pb-2 mb-3 border-b border-border">
            <span className="text-xs uppercase tracking-wider font-semibold text-blue-400 flex items-center gap-1.5">
              <AlignLeft className="w-3.5 h-3.5" />
              Word-Level Diff Analysis
            </span>
            <span className="text-[11px] text-slate-400">In-place token substitutions</span>
          </div>
          {renderInlineDiff()}
        </div>
      )}

      {/* Semantic Claim Inspector Detail */}
      {allResults.length > 0 && (
        <div className="mt-4 p-4 bg-slate-900/80 border border-border rounded-lg">
          <div className="text-xs uppercase tracking-wider font-semibold text-blue-400 mb-2 flex items-center gap-1.5">
            <Sparkles className="w-3.5 h-3.5" />
            Active Verification Breakdown
          </div>
          <div className="space-y-2">
            {allResults.map((r, i) => (
              <div key={i} className="flex flex-col sm:flex-row sm:items-center justify-between gap-2 text-xs p-2.5 rounded bg-surface border border-border">
                <div className="flex items-center gap-2">
                  <span className={`px-2 py-0.5 rounded font-medium ${
                    r.status === 'VERIFIED'
                      ? 'bg-emerald-500/20 text-emerald-400 border border-emerald-500/30'
                      : 'bg-amber-500/20 text-amber-400 border border-amber-500/30'
                  }`}>
                    {r.status}
                  </span>
                  <span className="text-slate-300 font-mono">{r.claim.entity_id}.{r.claim.attribute}</span>
                </div>
                <div className="flex flex-wrap items-center gap-2 sm:gap-3 text-slate-300">
                  <span className="line-through text-red-400 bg-red-500/10 px-1 rounded">{r.claim.asserted_value}</span>
                  <ArrowRight className="w-3 h-3 text-slate-500" />
                  <span className="text-emerald-400 font-semibold bg-emerald-500/10 px-1 rounded">
                    {r.authoritative_fact?.expected_value || r.applied_action}
                  </span>
                  <span className="text-slate-400 text-[11px]">Action: <span className="text-white font-medium">{r.applied_action}</span></span>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
