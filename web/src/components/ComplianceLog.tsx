import React, { useState, useEffect } from 'react'
import { FileText, Download, ShieldCheck, Search, Filter } from 'lucide-react'
import { AuditEvent } from '../types'

interface Props {
  liveAuditEvents?: AuditEvent[]
}

export const ComplianceLog: React.FC<Props> = ({ liveAuditEvents = [] }) => {
  const [events, setEvents] = useState<AuditEvent[]>([])
  const [loading, setLoading] = useState(true)
  const [statusFilter, setStatusFilter] = useState('')
  const [search, setSearch] = useState('')

  const loadAuditEvents = async () => {
    try {
      setLoading(true)
      let url = '/api/audit?limit=50'
      if (statusFilter) url += `&status=${statusFilter}`

      const res = await fetch(url)
      if (res.ok) {
        const data = await res.json()
        setEvents(data.events || [])
      }
    } catch (err) {
      console.error('Failed to load audit events:', err)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadAuditEvents()
  }, [statusFilter])

  // Merge live events from SSE stream
  useEffect(() => {
    if (liveAuditEvents.length > 0) {
      setEvents((prev) => {
        const newEvents = [...liveAuditEvents, ...prev]
        // deduplicate by id
        const seen = new Set()
        return newEvents.filter((item) => {
          if (seen.has(item.id)) return false
          seen.add(item.id)
          return true
        })
      })
    }
  }, [liveAuditEvents])

  const filtered = events.filter((e) => {
    const q = search.toLowerCase()
    return (
      e.request_id.toLowerCase().includes(q) ||
      e.entity_id.toLowerCase().includes(q) ||
      e.original_sentence.toLowerCase().includes(q) ||
      e.asserted_value.toLowerCase().includes(q)
    )
  })

  return (
    <div className="bg-surface border border-border rounded-xl p-6">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 mb-6">
        <div>
          <h2 className="text-lg font-semibold text-white flex items-center gap-2">
            <FileText className="w-5 h-5 text-purple-400" />
            Compliance Audit Ledger
          </h2>
          <p className="text-xs text-slate-400 mt-0.5">
            Tamper-evident SHA-256 audit log for EU AI Act Article 14 and Article 50 regulatory accountability.
          </p>
        </div>
        <div className="flex items-center gap-3">
          {/* Status filter */}
          <div className="relative">
            <Filter className="w-3.5 h-3.5 text-slate-400 absolute left-2.5 top-2.5" />
            <select
              value={statusFilter}
              onChange={(e) => setStatusFilter(e.target.value)}
              className="bg-surface-elevated border border-border rounded-lg pl-8 pr-3 py-1.5 text-xs text-white focus:outline-none focus:border-blue-500"
            >
              <option value="">All Statuses</option>
              <option value="CONTRADICTED">Contradicted</option>
              <option value="VERIFIED">Verified</option>
              <option value="UNGROUNDED">Ungrounded</option>
            </select>
          </div>

          {/* Search */}
          <div className="relative">
            <Search className="w-3.5 h-3.5 text-slate-400 absolute left-2.5 top-2.5" />
            <input
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Search audit records..."
              className="bg-surface-elevated border border-border rounded-lg pl-8 pr-3 py-1.5 text-xs text-white placeholder-slate-500 focus:outline-none focus:border-blue-500 w-44"
            />
          </div>

          {/* Export buttons */}
          <a
            href="/api/audit/export?format=csv"
            download
            className="flex items-center gap-1.5 px-3 py-1.5 bg-surface-elevated hover:bg-slate-700 text-slate-200 text-xs font-medium rounded-lg border border-border transition-colors"
          >
            <Download className="w-3.5 h-3.5" />
            <span>CSV</span>
          </a>
          <a
            href="/api/audit/export?format=json"
            download
            className="flex items-center gap-1.5 px-3 py-1.5 bg-surface-elevated hover:bg-slate-700 text-slate-200 text-xs font-medium rounded-lg border border-border transition-colors"
          >
            <Download className="w-3.5 h-3.5" />
            <span>JSON</span>
          </a>
        </div>
      </div>

      {/* Table */}
      <div className="overflow-x-auto">
        <table className="w-full text-left text-xs text-slate-300">
          <thead className="bg-surface-elevated text-slate-400 uppercase tracking-wider text-[11px] border-b border-border">
            <tr>
              <th className="px-4 py-3">Timestamp / Request</th>
              <th className="px-4 py-3">Status</th>
              <th className="px-4 py-3">Asserted vs Authoritative</th>
              <th className="px-4 py-3">Action Applied</th>
              <th className="px-4 py-3">Latency</th>
              <th className="px-4 py-3">SHA-256 Checksum</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-border font-mono">
            {loading ? (
              <tr>
                <td colSpan={6} className="px-4 py-8 text-center text-slate-500">
                  Loading compliance audit events...
                </td>
              </tr>
            ) : filtered.length === 0 ? (
              <tr>
                <td colSpan={6} className="px-4 py-8 text-center text-slate-500">
                  No compliance audit events recorded.
                </td>
              </tr>
            ) : (
              filtered.map((evt) => (
                <tr key={evt.id} className="hover:bg-slate-800/40 transition-colors">
                  <td className="px-4 py-3 text-[11px]">
                    <div className="text-white font-medium">
                      {new Date(evt.timestamp).toLocaleTimeString()}
                    </div>
                    <div className="text-slate-400">{evt.request_id}</div>
                  </td>
                  <td className="px-4 py-3">
                    <span
                      className={`px-2 py-0.5 rounded text-[10px] font-semibold tracking-wide ${
                        evt.status === 'CONTRADICTED'
                          ? 'bg-amber-500/10 text-amber-400 border border-amber-500/20'
                          : evt.status === 'VERIFIED'
                          ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20'
                          : 'bg-purple-500/10 text-purple-400 border border-purple-500/20'
                      }`}
                    >
                      {evt.status}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-xs">
                    <div className="text-slate-200">
                      <span className="line-through text-red-400 mr-2">{evt.asserted_value}</span>
                      <span className="text-emerald-400 font-semibold">{evt.authoritative_value || 'None'}</span>
                    </div>
                    <div className="text-[11px] text-slate-400 font-sans truncate max-w-xs">
                      {evt.original_sentence}
                    </div>
                  </td>
                  <td className="px-4 py-3">
                    <span className="px-2 py-0.5 rounded bg-surface-elevated border border-border text-[11px] text-slate-300">
                      {evt.policy_action}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-emerald-400 font-medium">
                    {evt.latency_ms.toFixed(2)} ms
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-1.5 text-slate-400 text-[11px]">
                      <ShieldCheck className="w-3.5 h-3.5 text-emerald-400" />
                      <span className="truncate w-24" title={evt.payload_hash}>
                        {evt.payload_hash.slice(0, 10)}...
                      </span>
                    </div>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  )
}
