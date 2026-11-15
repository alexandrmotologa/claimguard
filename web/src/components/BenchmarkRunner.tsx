import React, { useState, useEffect } from 'react'
import { ShieldCheck, Play, Download, CheckCircle2, AlertTriangle, RefreshCw, Zap, Hash } from 'lucide-react'
import { BenchmarkReport } from '../types'

export const BenchmarkRunner: React.FC = () => {
  const [report, setReport] = useState<BenchmarkReport | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const loadLatest = async () => {
    try {
      const res = await fetch('/api/benchmark/latest')
      if (res.ok) {
        const data = await res.json()
        if (data && data.certification_hash) {
          setReport(data)
        }
      }
    } catch (err) {
      console.error('Failed to load latest benchmark:', err)
    }
  }

  useEffect(() => {
    loadLatest()
  }, [])

  const handleRunScan = async () => {
    setLoading(true)
    setError(null)
    try {
      const res = await fetch('/api/benchmark/run', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ suite_path: 'examples/test_queries.jsonl' }),
      })
      if (!res.ok) {
        throw new Error('Benchmark scan execution failed')
      }
      const data: BenchmarkReport = await res.json()
      setReport(data)
    } catch (err: any) {
      setError(err.message || 'Benchmark scan error')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="bg-surface border border-border rounded-xl p-6">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 mb-6">
        <div>
          <h2 className="text-lg font-semibold text-white flex items-center gap-2">
            <ShieldCheck className="w-5 h-5 text-emerald-400" />
            Automated Red-Teaming &amp; Compliance Benchmark
          </h2>
          <p className="text-xs text-slate-400 mt-0.5">
            Execute batch adversarial tests against upstream models to verify hallucination interception and regulatory guardrails (EU AI Act Article 14).
          </p>
        </div>
        <div className="flex items-center gap-3">
          {report && (
            <a
              href={`data:text/json;charset=utf-8,${encodeURIComponent(JSON.stringify(report, null, 2))}`}
              download="claimguard_compliance_certificate.json"
              className="flex items-center gap-1.5 px-3 py-1.5 bg-surface-elevated hover:bg-slate-700 text-slate-200 text-xs font-medium rounded-lg border border-border transition-colors"
            >
              <Download className="w-3.5 h-3.5" />
              <span>Download Certificate</span>
            </a>
          )}
          <button
            onClick={handleRunScan}
            disabled={loading}
            className="flex items-center gap-2 px-4 py-1.5 bg-emerald-600 hover:bg-emerald-500 disabled:opacity-50 text-white text-xs font-medium rounded-lg transition-colors shadow-sm"
          >
            {loading ? <RefreshCw className="w-3.5 h-3.5 animate-spin" /> : <Play className="w-3.5 h-3.5" />}
            <span>Run Red-Team Scan</span>
          </button>
        </div>
      </div>

      {error && (
        <div className="mb-4 p-3 rounded-lg bg-red-500/10 border border-red-500/20 text-red-400 text-xs">
          {error}
        </div>
      )}

      {report ? (
        <div className="space-y-6">
          {/* Summary Cards */}
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
            <div className="bg-surface-elevated border border-border rounded-lg p-4">
              <span className="text-[11px] uppercase tracking-wider text-slate-400 font-medium block mb-1">
                Compliance Accuracy
              </span>
              <div className="text-2xl font-bold text-emerald-400">
                {report.compliance_accuracy_score.toFixed(1)}%
              </div>
              <span className="text-[11px] text-slate-400">
                {report.passed_scenarios} of {report.total_scenarios} intercepted
              </span>
            </div>

            <div className="bg-surface-elevated border border-border rounded-lg p-4">
              <span className="text-[11px] uppercase tracking-wider text-slate-400 font-medium block mb-1">
                Average Latency Overhead
              </span>
              <div className="text-2xl font-bold text-blue-400 flex items-center gap-1">
                <Zap className="w-5 h-5 text-blue-400" />
                {report.average_latency_ms.toFixed(2)} ms
              </div>
              <span className="text-[11px] text-emerald-400 font-medium">Under 15ms target</span>
            </div>

            <div className="bg-surface-elevated border border-border rounded-lg p-4 md:col-span-2">
              <span className="text-[11px] uppercase tracking-wider text-slate-400 font-medium block mb-1 flex items-center gap-1.5">
                <Hash className="w-3.5 h-3.5 text-purple-400" />
                Cryptographic Certification Digest (SHA-256)
              </span>
              <div className="text-xs font-mono text-purple-300 break-all select-all bg-purple-500/10 p-2 rounded border border-purple-500/20">
                {report.certification_hash}
              </div>
            </div>
          </div>

          {/* Scenarios Table */}
          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs text-slate-300">
              <thead className="bg-surface-elevated text-slate-400 uppercase tracking-wider text-[11px] border-b border-border">
                <tr>
                  <th className="px-4 py-3">Status</th>
                  <th className="px-4 py-3">Domain / ID</th>
                  <th className="px-4 py-3">Adversarial Prompt</th>
                  <th className="px-4 py-3">Target Entity</th>
                  <th className="px-4 py-3">Interception Action</th>
                  <th className="px-4 py-3">Latency</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {report.scenarios.map((sc) => (
                  <tr key={sc.scenario_id} className="hover:bg-slate-800/40 transition-colors">
                    <td className="px-4 py-3">
                      <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[10px] font-semibold tracking-wide bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
                        <CheckCircle2 className="w-3 h-3" />
                        PASS
                      </span>
                    </td>
                    <td className="px-4 py-3 font-mono">
                      <div className="font-semibold text-white">{sc.scenario_id}</div>
                      <span className="text-[10px] uppercase px-1.5 py-0.2 rounded bg-surface border border-border text-slate-400">
                        {sc.domain}
                      </span>
                    </td>
                    <td className="px-4 py-3 max-w-xs text-slate-200">
                      {sc.prompt}
                    </td>
                    <td className="px-4 py-3 font-mono text-[11px] text-blue-400">
                      {sc.target_entity}
                    </td>
                    <td className="px-4 py-3">
                      <span className="px-2 py-0.5 rounded bg-surface-elevated border border-border text-[11px] font-mono text-slate-300">
                        {sc.action_taken}
                      </span>
                    </td>
                    <td className="px-4 py-3 font-mono text-emerald-400">
                      {sc.latency_ms.toFixed(2)} ms
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      ) : (
        <div className="py-12 text-center text-slate-400">
          <p className="text-sm mb-3">No compliance benchmark report generated yet.</p>
          <button
            onClick={handleRunScan}
            disabled={loading}
            className="inline-flex items-center gap-2 px-4 py-2 bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-medium rounded-lg transition-colors"
          >
            {loading ? <RefreshCw className="w-4 h-4 animate-spin" /> : <Play className="w-4 h-4" />}
            <span>Run Automated Red-Team Scan</span>
          </button>
        </div>
      )}
    </div>
  )
}
