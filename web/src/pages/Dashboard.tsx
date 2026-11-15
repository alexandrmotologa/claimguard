import React, { useState, useEffect } from 'react'
import { Shield, Activity, Database, FileText, Sliders, Radio, ShieldCheck } from 'lucide-react'
import { HallucinationGauge } from '../components/HallucinationGauge'
import { StreamAuditor } from '../components/StreamAuditor'
import { TruthTableEditor } from '../components/TruthTableEditor'
import { ClaimFrequency } from '../components/ClaimFrequency'
import { ComplianceLog } from '../components/ComplianceLog'
import { PolicyConfig } from '../components/PolicyConfig'
import { BenchmarkRunner } from '../components/BenchmarkRunner'
import { MetricsSnapshot, AuditEvent } from '../types'

export const Dashboard: React.FC = () => {
  const [activeTab, setActiveTab] = useState<'cockpit' | 'registry' | 'audit' | 'policy' | 'benchmark'>('cockpit')
  const [metrics, setMetrics] = useState<MetricsSnapshot | null>(null)
  const [liveEvents, setLiveEvents] = useState<AuditEvent[]>([])
  const [isConnected, setIsConnected] = useState(false)

  // Fetch initial telemetry metrics
  const fetchMetrics = async () => {
    try {
      const res = await fetch('/api/metrics')
      if (res.ok) {
        setMetrics(await res.json())
      }
    } catch (err) {
      console.error('Failed to fetch metrics:', err)
    }
  }

  useEffect(() => {
    fetchMetrics()
    const interval = setInterval(fetchMetrics, 3000)
    return () => clearInterval(interval)
  }, [])

  // Listen to live SSE events from `/api/events/live`
  useEffect(() => {
    const eventSource = new EventSource('/api/events/live')

    eventSource.onopen = () => {
      setIsConnected(true)
    }

    eventSource.addEventListener('AUDIT_INTERCEPTION', (e: MessageEvent) => {
      try {
        const payload = JSON.parse(e.data)
        const auditEvent: AuditEvent = payload.data
        setLiveEvents((prev) => [auditEvent, ...prev.slice(0, 49)])
        fetchMetrics()
      } catch (err) {
        console.error('Error parsing live audit event:', err)
      }
    })

    eventSource.onerror = () => {
      setIsConnected(false)
    }

    return () => {
      eventSource.close()
    }
  }, [])

  return (
    <div className="min-h-screen bg-background text-slate-100 flex flex-col">
      {/* Top Navbar */}
      <header className="border-b border-border bg-surface/80 backdrop-blur sticky top-0 z-40">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 h-16 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="w-9 h-9 rounded-lg bg-blue-600/20 border border-blue-500/30 flex items-center justify-center text-blue-400">
              <Shield className="w-5 h-5" />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <span className="font-bold text-white tracking-tight text-base">claimguard</span>
                <span className="text-[10px] uppercase font-semibold tracking-wider px-2 py-0.5 rounded-full bg-blue-500/10 text-blue-400 border border-blue-500/20">
                  AI Governance Airlock
                </span>
              </div>
              <p className="text-[11px] text-slate-400">
                Wire-Speed Semantic Claim Verification &amp; Interception Proxy
              </p>
            </div>
          </div>

          {/* Right Navigation & Status */}
          <div className="flex items-center gap-4">
            <div className="flex items-center gap-2 px-3 py-1.5 rounded-full bg-surface-elevated border border-border text-xs">
              <span className={`w-2 h-2 rounded-full ${isConnected ? 'bg-emerald-400 animate-pulse' : 'bg-red-400'}`} />
              <span className="text-slate-300 font-medium">
                {isConnected ? 'Proxy Live (:8080)' : 'Connecting...'}
              </span>
            </div>
          </div>
        </div>
      </header>

      {/* Navigation Sub-bar */}
      <nav className="border-b border-border bg-surface/40 overflow-x-auto scrollbar-none">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 flex items-center gap-1 min-w-max">
          <button
            onClick={() => setActiveTab('cockpit')}
            className={`flex items-center gap-2 px-4 py-3 text-xs font-medium border-b-2 transition-colors whitespace-nowrap ${
              activeTab === 'cockpit'
                ? 'border-blue-500 text-white'
                : 'border-transparent text-slate-400 hover:text-slate-200'
            }`}
          >
            <Activity className="w-3.5 h-3.5" />
            <span>Audit Cockpit</span>
          </button>
          <button
            onClick={() => setActiveTab('registry')}
            className={`flex items-center gap-2 px-4 py-3 text-xs font-medium border-b-2 transition-colors whitespace-nowrap ${
              activeTab === 'registry'
                ? 'border-blue-500 text-white'
                : 'border-transparent text-slate-400 hover:text-slate-200'
            }`}
          >
            <Database className="w-3.5 h-3.5" />
            <span>Ground Truth Registry</span>
          </button>
          <button
            onClick={() => setActiveTab('audit')}
            className={`flex items-center gap-2 px-4 py-3 text-xs font-medium border-b-2 transition-colors whitespace-nowrap ${
              activeTab === 'audit'
                ? 'border-blue-500 text-white'
                : 'border-transparent text-slate-400 hover:text-slate-200'
            }`}
          >
            <FileText className="w-3.5 h-3.5" />
            <span>Compliance Ledger</span>
          </button>
          <button
            onClick={() => setActiveTab('policy')}
            className={`flex items-center gap-2 px-4 py-3 text-xs font-medium border-b-2 transition-colors whitespace-nowrap ${
              activeTab === 'policy'
                ? 'border-blue-500 text-white'
                : 'border-transparent text-slate-400 hover:text-slate-200'
            }`}
          >
            <Sliders className="w-3.5 h-3.5" />
            <span>Action Policy</span>
          </button>
          <button
            onClick={() => setActiveTab('benchmark')}
            className={`flex items-center gap-2 px-4 py-3 text-xs font-medium border-b-2 transition-colors whitespace-nowrap ${
              activeTab === 'benchmark'
                ? 'border-blue-500 text-white'
                : 'border-transparent text-slate-400 hover:text-slate-200'
            }`}
          >
            <ShieldCheck className="w-3.5 h-3.5 text-emerald-400" />
            <span>Red-Team Benchmark</span>
          </button>
        </div>
      </nav>

      {/* Main Content Area */}
      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8 flex-1 w-full space-y-6">
        {activeTab === 'cockpit' && (
          <>
            <HallucinationGauge metrics={metrics} />
            <StreamAuditor />
            <ClaimFrequency metrics={metrics} />
          </>
        )}

        {activeTab === 'registry' && <TruthTableEditor />}

        {activeTab === 'audit' && <ComplianceLog liveAuditEvents={liveEvents} />}

        {activeTab === 'policy' && <PolicyConfig />}

        {activeTab === 'benchmark' && <BenchmarkRunner />}
      </main>

      {/* Footer */}
      <footer className="border-t border-border py-4 text-center text-xs text-slate-500">
        claimguard &middot; EU AI Act Article 14/50 Technical Safeguard Proxy &middot; <span className="font-mono">modernc.org/sqlite</span> &middot; Under 25MB RAM
      </footer>
    </div>
  )
}
