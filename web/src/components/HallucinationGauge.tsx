import React from 'react'
import { ShieldCheck, Zap, AlertTriangle, Activity } from 'lucide-react'
import { MetricsSnapshot } from '../types'

interface Props {
  metrics: MetricsSnapshot | null
}

export const HallucinationGauge: React.FC<Props> = ({ metrics }) => {
  const rate = metrics?.interception_rate_pct ?? 0
  const totalVerified = metrics?.total_claims_verified ?? 0
  const totalContradictions = metrics?.total_contradictions ?? 0
  const avgLatency = metrics?.average_latency_ms ?? 0
  const totalRequests = metrics?.total_requests ?? 0

  // Calculate SVG circular stroke
  const radius = 64
  const circumference = 2 * Math.PI * radius
  const strokeDashoffset = circumference - (Math.min(rate, 100) / 100) * circumference

  return (
    <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
      {/* Interception Rate Dial */}
      <div className="bg-surface border border-border rounded-xl p-5 flex items-center justify-between">
        <div>
          <div className="text-xs uppercase tracking-wider text-slate-400 font-medium mb-1">
            Interception Rate
          </div>
          <div className="text-2xl font-bold text-white tracking-tight">
            {rate.toFixed(1)}%
          </div>
          <div className="text-xs text-slate-400 mt-1">
            {totalContradictions} intercepted of {totalVerified} claims
          </div>
        </div>
        <div className="relative w-20 h-20 flex items-center justify-center">
          <svg className="w-20 h-20 transform -rotate-90">
            <circle
              cx="40"
              cy="40"
              r={radius / 2}
              stroke="#243248"
              strokeWidth="6"
              fill="transparent"
            />
            <circle
              cx="40"
              cy="40"
              r={radius / 2}
              stroke={rate > 0 ? '#f59e0b' : '#10b981'}
              strokeWidth="6"
              strokeDasharray={circumference / 2}
              strokeDashoffset={strokeDashoffset / 2}
              strokeLinecap="round"
              fill="transparent"
              className="transition-all duration-700 ease-out"
            />
          </svg>
          <div className="absolute text-xs font-semibold text-slate-200">
            {rate.toFixed(0)}%
          </div>
        </div>
      </div>

      {/* Latency Impact */}
      <div className="bg-surface border border-border rounded-xl p-5">
        <div className="flex items-center justify-between text-slate-400 mb-2">
          <span className="text-xs uppercase tracking-wider font-medium">Added Latency</span>
          <Zap className="w-4 h-4 text-emerald-400" />
        </div>
        <div className="text-2xl font-bold text-emerald-400 tracking-tight">
          {avgLatency.toFixed(2)} ms
        </div>
        <div className="flex items-center gap-1.5 text-xs text-slate-400 mt-1">
          <span className="inline-block w-2 h-2 rounded-full bg-emerald-400"></span>
          <span>Target &lt; 15.0 ms (Wire-speed)</span>
        </div>
      </div>

      {/* Verified Claims */}
      <div className="bg-surface border border-border rounded-xl p-5">
        <div className="flex items-center justify-between text-slate-400 mb-2">
          <span className="text-xs uppercase tracking-wider font-medium">Verified Claims</span>
          <ShieldCheck className="w-4 h-4 text-blue-400" />
        </div>
        <div className="text-2xl font-bold text-white tracking-tight">
          {totalVerified.toLocaleString()}
        </div>
        <div className="text-xs text-slate-400 mt-1">
          Across {metrics?.total_sentences ?? 0} analyzed sentences
        </div>
      </div>

      {/* Active Interventions */}
      <div className="bg-surface border border-border rounded-xl p-5">
        <div className="flex items-center justify-between text-slate-400 mb-2">
          <span className="text-xs uppercase tracking-wider font-medium">Total Requests</span>
          <Activity className="w-4 h-4 text-purple-400" />
        </div>
        <div className="text-2xl font-bold text-white tracking-tight">
          {totalRequests.toLocaleString()}
        </div>
        <div className="text-xs text-slate-400 mt-1">
          {metrics?.total_tokens ?? 0} tokens scanned
        </div>
      </div>
    </div>
  )
}
