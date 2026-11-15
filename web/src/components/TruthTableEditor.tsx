import React, { useState, useEffect, useRef } from 'react'
import { Plus, Search, Trash2, Edit3, Database, Save, X, ShieldAlert, Upload, Download, CheckCircle2 } from 'lucide-react'
import { Fact, Entity, ForbiddenPattern } from '../types'

export const TruthTableEditor: React.FC = () => {
  const [activeTab, setActiveTab] = useState<'facts' | 'forbidden'>('facts')
  const [facts, setFacts] = useState<Fact[]>([])
  const [entities, setEntities] = useState<Entity[]>([])
  const [forbidden, setForbidden] = useState<ForbiddenPattern[]>([])
  const [search, setSearch] = useState('')
  const [loading, setLoading] = useState(true)
  const [importStatus, setImportStatus] = useState<string | null>(null)

  // Fact modal state
  const [editingFact, setEditingFact] = useState<Fact | null>(null)
  const [isAddingFact, setIsAddingFact] = useState(false)
  const [factForm, setFactForm] = useState<Partial<Fact>>({
    id: '',
    entity_id: '',
    attribute: '',
    claim_type: 'duration',
    expected_value: '',
    numeric_value: 0,
    unit: '',
    tolerance: 0,
    policy_action: 'rewrite',
    description: '',
    source_doc: '',
  })

  // Forbidden modal state
  const [editingForbidden, setEditingForbidden] = useState<ForbiddenPattern | null>(null)
  const [isAddingForbidden, setIsAddingForbidden] = useState(false)
  const [forbiddenForm, setForbiddenForm] = useState<Partial<ForbiddenPattern>>({
    id: '',
    name: '',
    pattern: '',
    category: 'commercial',
    policy_action: 'strict_halt',
    replacement: '',
    reason: '',
    source_doc: '',
  })

  const fileInputRef = useRef<HTMLInputElement>(null)

  const loadData = async () => {
    try {
      setLoading(true)
      const [factsRes, entitiesRes, forbiddenRes] = await Promise.all([
        fetch('/api/facts'),
        fetch('/api/entities'),
        fetch('/api/forbidden'),
      ])
      if (factsRes.ok) setFacts(await factsRes.json())
      if (entitiesRes.ok) setEntities(await entitiesRes.json())
      if (forbiddenRes.ok) setForbidden(await forbiddenRes.json())
    } catch (err) {
      console.error('Failed to load truth registry:', err)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadData()
  }, [])

  // Save fact
  const handleSaveFact = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      const url = editingFact ? `/api/facts/${editingFact.id}` : '/api/facts'
      const method = editingFact ? 'PUT' : 'POST'
      const res = await fetch(url, {
        method,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(factForm),
      })
      if (res.ok) {
        setIsAddingFact(false)
        setEditingFact(null)
        loadData()
      }
    } catch (err) {
      console.error('Save fact error:', err)
    }
  }

  // Delete fact
  const handleDeleteFact = async (id: string) => {
    if (!confirm('Are you sure you want to delete this authoritative fact?')) return
    try {
      const res = await fetch(`/api/facts/${id}`, { method: 'DELETE' })
      if (res.ok) loadData()
    } catch (err) {
      console.error('Delete fact error:', err)
    }
  }

  // Save forbidden rule
  const handleSaveForbidden = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      const res = await fetch('/api/forbidden', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(forbiddenForm),
      })
      if (res.ok) {
        setIsAddingForbidden(false)
        setEditingForbidden(null)
        loadData()
      }
    } catch (err) {
      console.error('Save forbidden rule error:', err)
    }
  }

  // Delete forbidden rule
  const handleDeleteForbidden = async (id: string) => {
    if (!confirm('Are you sure you want to delete this forbidden commitment rule?')) return
    try {
      const res = await fetch(`/api/forbidden/${id}`, { method: 'DELETE' })
      if (res.ok) loadData()
    } catch (err) {
      console.error('Delete forbidden rule error:', err)
    }
  }

  // Drag and drop file ingestion
  const handleFileUpload = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return

    const reader = new FileReader()
    reader.onload = async (event) => {
      try {
        const text = event.target?.result as string
        const json = JSON.parse(text)

        if (json.facts && Array.isArray(json.facts)) {
          for (const f of json.facts) {
            await fetch('/api/facts', {
              method: 'POST',
              headers: { 'Content-Type': 'application/json' },
              body: JSON.stringify(f),
            })
          }
        }
        if (json.forbidden_patterns && Array.isArray(json.forbidden_patterns)) {
          for (const fp of json.forbidden_patterns) {
            await fetch('/api/forbidden', {
              method: 'POST',
              headers: { 'Content-Type': 'application/json' },
              body: JSON.stringify(fp),
            })
          }
        }

        setImportStatus(`Imported ${file.name} successfully`)
        setTimeout(() => setImportStatus(null), 3000)
        loadData()
      } catch (err) {
        console.error('Import error:', err)
        alert('Invalid JSON file format')
      }
    }
    reader.readAsText(file)
  }

  const filteredFacts = facts.filter((f) => {
    const q = search.toLowerCase()
    return (
      f.entity_id.toLowerCase().includes(q) ||
      f.attribute.toLowerCase().includes(q) ||
      f.expected_value.toLowerCase().includes(q)
    )
  })

  const filteredForbidden = forbidden.filter((fp) => {
    const q = search.toLowerCase()
    return (
      fp.name.toLowerCase().includes(q) ||
      fp.pattern.toLowerCase().includes(q) ||
      fp.reason.toLowerCase().includes(q)
    )
  })

  return (
    <div className="bg-surface border border-border rounded-xl p-6">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 mb-6">
        <div>
          <h2 className="text-lg font-semibold text-white flex items-center gap-2">
            <Database className="w-5 h-5 text-blue-400" />
            Ground Truth Knowledge Registry
          </h2>
          <p className="text-xs text-slate-400 mt-0.5">
            Authoritative enterprise facts and negative commitment rules stored in-memory in SQLite with sub-millisecond lookup latency.
          </p>
        </div>
        <div className="flex items-center gap-3">
          <input
            type="file"
            ref={fileInputRef}
            onChange={handleFileUpload}
            accept=".json,.csv"
            className="hidden"
          />
          <button
            onClick={() => fileInputRef.current?.click()}
            className="flex items-center gap-1.5 px-3 py-1.5 bg-surface-elevated hover:bg-slate-700 text-slate-300 border border-border text-xs font-medium rounded-lg transition-colors"
          >
            <Upload className="w-3.5 h-3.5" />
            <span>Import JSON</span>
          </button>

          <div className="relative">
            <Search className="w-4 h-4 text-slate-400 absolute left-3 top-2.5" />
            <input
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Search registry..."
              className="bg-surface-elevated border border-border rounded-lg pl-9 pr-3 py-1.5 text-xs text-white placeholder-slate-500 focus:outline-none focus:border-blue-500 w-48"
            />
          </div>

          <button
            onClick={() => {
              if (activeTab === 'facts') {
                setEditingFact(null)
                setFactForm({
                  id: `fact-${Date.now().toString(36)}`,
                  entity_id: entities[0]?.id || 'custom.entity',
                  attribute: '',
                  claim_type: 'duration',
                  expected_value: '',
                  numeric_value: 0,
                  unit: '',
                  tolerance: 0,
                  policy_action: 'rewrite',
                  description: '',
                  source_doc: '',
                })
                setIsAddingFact(true)
              } else {
                setEditingForbidden(null)
                setForbiddenForm({
                  id: `fp-${Date.now().toString(36)}`,
                  name: '',
                  pattern: '',
                  category: 'commercial',
                  policy_action: 'strict_halt',
                  replacement: '',
                  reason: '',
                  source_doc: '',
                })
                setIsAddingForbidden(true)
              }
            }}
            className="flex items-center gap-1.5 px-3 py-1.5 bg-blue-600 hover:bg-blue-500 text-white text-xs font-medium rounded-lg transition-colors"
          >
            <Plus className="w-3.5 h-3.5" />
            <span>{activeTab === 'facts' ? 'Add Fact' : 'Add Forbidden Rule'}</span>
          </button>
        </div>
      </div>

      {importStatus && (
        <div className="mb-4 p-2.5 rounded-lg bg-emerald-500/10 border border-emerald-500/20 text-emerald-400 text-xs flex items-center gap-2">
          <CheckCircle2 className="w-4 h-4" />
          <span>{importStatus}</span>
        </div>
      )}

      {/* Sub-tabs */}
      <div className="flex border-b border-border mb-4">
        <button
          onClick={() => setActiveTab('facts')}
          className={`flex items-center gap-2 px-4 py-2.5 text-xs font-medium border-b-2 transition-colors ${
            activeTab === 'facts'
              ? 'border-blue-500 text-white'
              : 'border-transparent text-slate-400 hover:text-slate-200'
          }`}
        >
          <Database className="w-3.5 h-3.5" />
          <span>Authoritative Facts ({facts.length})</span>
        </button>
        <button
          onClick={() => setActiveTab('forbidden')}
          className={`flex items-center gap-2 px-4 py-2.5 text-xs font-medium border-b-2 transition-colors ${
            activeTab === 'forbidden'
              ? 'border-blue-500 text-white'
              : 'border-transparent text-slate-400 hover:text-slate-200'
          }`}
        >
          <ShieldAlert className="w-3.5 h-3.5 text-amber-400" />
          <span>Forbidden Commitments &amp; Invariants ({forbidden.length})</span>
        </button>
      </div>

      {/* Facts View */}
      {activeTab === 'facts' && (
        <div className="overflow-x-auto">
          <table className="w-full text-left text-xs text-slate-300">
            <thead className="bg-surface-elevated text-slate-400 uppercase tracking-wider text-[11px] border-b border-border">
              <tr>
                <th className="px-4 py-3">Entity &amp; Attribute</th>
                <th className="px-4 py-3">Claim Type</th>
                <th className="px-4 py-3">Authoritative Value</th>
                <th className="px-4 py-3">Action Policy</th>
                <th className="px-4 py-3">Source Reference</th>
                <th className="px-4 py-3 text-right">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {loading ? (
                <tr>
                  <td colSpan={6} className="px-4 py-8 text-center text-slate-500">
                    Loading facts from SQLite registry...
                  </td>
                </tr>
              ) : filteredFacts.length === 0 ? (
                <tr>
                  <td colSpan={6} className="px-4 py-8 text-center text-slate-500">
                    No ground truth facts matched your search.
                  </td>
                </tr>
              ) : (
                filteredFacts.map((fact) => (
                  <tr key={fact.id} className="hover:bg-slate-800/40 transition-colors">
                    <td className="px-4 py-3 font-mono">
                      <div className="font-semibold text-white">{fact.entity_id}</div>
                      <div className="text-[11px] text-slate-400">{fact.attribute}</div>
                    </td>
                    <td className="px-4 py-3">
                      <span className="px-2 py-0.5 rounded bg-surface-elevated border border-border text-[11px] font-mono text-slate-300">
                        {fact.claim_type}
                      </span>
                    </td>
                    <td className="px-4 py-3">
                      <span className="text-emerald-400 font-semibold">{fact.expected_value}</span>
                      <div className="text-[11px] text-slate-400">
                        Numeric: {fact.numeric_value} {fact.unit}
                      </div>
                    </td>
                    <td className="px-4 py-3">
                      <span className={`px-2 py-0.5 rounded text-[11px] font-medium ${
                        fact.policy_action === 'rewrite'
                          ? 'bg-blue-500/10 text-blue-400 border border-blue-500/20'
                          : fact.policy_action === 'disclaimer'
                          ? 'bg-amber-500/10 text-amber-400 border border-amber-500/20'
                          : 'bg-red-500/10 text-red-400 border border-red-500/20'
                      }`}>
                        {fact.policy_action}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-slate-400 font-mono text-[11px]">
                      {fact.source_doc || 'N/A'}
                    </td>
                    <td className="px-4 py-3 text-right">
                      <div className="flex items-center justify-end gap-2">
                        <button
                          onClick={() => {
                            setEditingFact(fact)
                            setFactForm(fact)
                            setIsAddingFact(true)
                          }}
                          className="p-1 hover:bg-slate-700 text-slate-400 hover:text-white rounded transition-colors"
                        >
                          <Edit3 className="w-3.5 h-3.5" />
                        </button>
                        <button
                          onClick={() => handleDeleteFact(fact.id)}
                          className="p-1 hover:bg-red-500/20 text-slate-400 hover:text-red-400 rounded transition-colors"
                        >
                          <Trash2 className="w-3.5 h-3.5" />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      )}

      {/* Forbidden Rules View */}
      {activeTab === 'forbidden' && (
        <div className="overflow-x-auto">
          <table className="w-full text-left text-xs text-slate-300">
            <thead className="bg-surface-elevated text-slate-400 uppercase tracking-wider text-[11px] border-b border-border">
              <tr>
                <th className="px-4 py-3">Rule Name &amp; Category</th>
                <th className="px-4 py-3">Regex Pattern</th>
                <th className="px-4 py-3">Policy Action</th>
                <th className="px-4 py-3">Compliance Justification</th>
                <th className="px-4 py-3 text-right">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {filteredForbidden.length === 0 ? (
                <tr>
                  <td colSpan={5} className="px-4 py-8 text-center text-slate-500">
                    No forbidden commitment rules defined.
                  </td>
                </tr>
              ) : (
                filteredForbidden.map((fp) => (
                  <tr key={fp.id} className="hover:bg-slate-800/40 transition-colors">
                    <td className="px-4 py-3">
                      <div className="font-semibold text-white">{fp.name}</div>
                      <span className="text-[10px] uppercase font-mono px-1.5 py-0.5 rounded bg-surface-elevated border border-border text-slate-400">
                        {fp.category}
                      </span>
                    </td>
                    <td className="px-4 py-3 font-mono text-[11px] text-amber-400">
                      {fp.pattern}
                    </td>
                    <td className="px-4 py-3">
                      <span className="px-2 py-0.5 rounded bg-red-500/10 text-red-400 border border-red-500/20 text-[11px] font-medium">
                        {fp.policy_action}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-slate-300 text-xs max-w-xs">
                      {fp.reason}
                    </td>
                    <td className="px-4 py-3 text-right">
                      <button
                        onClick={() => handleDeleteForbidden(fp.id)}
                        className="p-1 hover:bg-red-500/20 text-slate-400 hover:text-red-400 rounded transition-colors"
                      >
                        <Trash2 className="w-3.5 h-3.5" />
                      </button>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      )}

      {/* Add Fact Modal */}
      {isAddingFact && (
        <div className="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center p-4 z-50">
          <div className="bg-surface border border-border rounded-xl w-full max-w-lg p-6 shadow-2xl">
            <div className="flex items-center justify-between mb-4 pb-3 border-b border-border">
              <h3 className="text-base font-semibold text-white">
                {editingFact ? 'Edit Authoritative Fact' : 'Add Ground Truth Fact'}
              </h3>
              <button onClick={() => setIsAddingFact(false)} className="text-slate-400 hover:text-white">
                <X className="w-4 h-4" />
              </button>
            </div>
            <form onSubmit={handleSaveFact} className="space-y-4">
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-xs font-medium text-slate-400 mb-1">Entity ID</label>
                  <input
                    type="text"
                    required
                    value={factForm.entity_id}
                    onChange={(e) => setFactForm({ ...factForm, entity_id: e.target.value })}
                    className="w-full bg-surface-elevated border border-border rounded-lg px-3 py-1.5 text-xs text-white focus:outline-none focus:border-blue-500"
                  />
                </div>
                <div>
                  <label className="block text-xs font-medium text-slate-400 mb-1">Attribute</label>
                  <input
                    type="text"
                    required
                    value={factForm.attribute}
                    onChange={(e) => setFactForm({ ...factForm, attribute: e.target.value })}
                    className="w-full bg-surface-elevated border border-border rounded-lg px-3 py-1.5 text-xs text-white focus:outline-none focus:border-blue-500"
                  />
                </div>
              </div>
              <div className="grid grid-cols-3 gap-3">
                <div>
                  <label className="block text-xs font-medium text-slate-400 mb-1">Claim Type</label>
                  <select
                    value={factForm.claim_type}
                    onChange={(e) => setFactForm({ ...factForm, claim_type: e.target.value })}
                    className="w-full bg-surface-elevated border border-border rounded-lg px-3 py-1.5 text-xs text-white focus:outline-none focus:border-blue-500"
                  >
                    <option value="duration">duration</option>
                    <option value="currency">currency</option>
                    <option value="percentage">percentage</option>
                    <option value="text">text</option>
                  </select>
                </div>
                <div>
                  <label className="block text-xs font-medium text-slate-400 mb-1">Expected Value</label>
                  <input
                    type="text"
                    required
                    value={factForm.expected_value}
                    onChange={(e) => setFactForm({ ...factForm, expected_value: e.target.value })}
                    className="w-full bg-surface-elevated border border-border rounded-lg px-3 py-1.5 text-xs text-white focus:outline-none focus:border-blue-500"
                  />
                </div>
                <div>
                  <label className="block text-xs font-medium text-slate-400 mb-1">Numeric Float</label>
                  <input
                    type="number"
                    step="any"
                    value={factForm.numeric_value}
                    onChange={(e) => setFactForm({ ...factForm, numeric_value: parseFloat(e.target.value) || 0 })}
                    className="w-full bg-surface-elevated border border-border rounded-lg px-3 py-1.5 text-xs text-white focus:outline-none focus:border-blue-500"
                  />
                </div>
              </div>
              <div className="flex justify-end gap-2 pt-2 border-t border-border">
                <button
                  type="button"
                  onClick={() => setIsAddingFact(false)}
                  className="px-4 py-2 bg-slate-800 hover:bg-slate-700 text-slate-300 text-xs font-medium rounded-lg"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="px-4 py-2 bg-blue-600 hover:bg-blue-500 text-white text-xs font-medium rounded-lg"
                >
                  Save Fact
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Add Forbidden Modal */}
      {isAddingForbidden && (
        <div className="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center p-4 z-50">
          <div className="bg-surface border border-border rounded-xl w-full max-w-lg p-6 shadow-2xl">
            <div className="flex items-center justify-between mb-4 pb-3 border-b border-border">
              <h3 className="text-base font-semibold text-white">Add Forbidden Commitment Rule</h3>
              <button onClick={() => setIsAddingForbidden(false)} className="text-slate-400 hover:text-white">
                <X className="w-4 h-4" />
              </button>
            </div>
            <form onSubmit={handleSaveForbidden} className="space-y-4">
              <div>
                <label className="block text-xs font-medium text-slate-400 mb-1">Rule Name</label>
                <input
                  type="text"
                  required
                  value={forbiddenForm.name}
                  onChange={(e) => setForbiddenForm({ ...forbiddenForm, name: e.target.value })}
                  placeholder="e.g. Unauthorized Lifetime Guarantee"
                  className="w-full bg-surface-elevated border border-border rounded-lg px-3 py-1.5 text-xs text-white focus:outline-none focus:border-blue-500"
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-slate-400 mb-1">Regex Pattern</label>
                <input
                  type="text"
                  required
                  value={forbiddenForm.pattern}
                  onChange={(e) => setForbiddenForm({ ...forbiddenForm, pattern: e.target.value })}
                  placeholder="(?i)\b(lifetime\s+warranty)\b"
                  className="w-full bg-surface-elevated border border-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-blue-500"
                />
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-xs font-medium text-slate-400 mb-1">Policy Action</label>
                  <select
                    value={forbiddenForm.policy_action}
                    onChange={(e) => setForbiddenForm({ ...forbiddenForm, policy_action: e.target.value })}
                    className="w-full bg-surface-elevated border border-border rounded-lg px-3 py-1.5 text-xs text-white focus:outline-none focus:border-blue-500"
                  >
                    <option value="strict_halt">strict_halt (Terminate Stream)</option>
                    <option value="rewrite">rewrite (In-place substitute)</option>
                    <option value="disclaimer">disclaimer (Append notice)</option>
                  </select>
                </div>
                <div>
                  <label className="block text-xs font-medium text-slate-400 mb-1">Category</label>
                  <select
                    value={forbiddenForm.category}
                    onChange={(e) => setForbiddenForm({ ...forbiddenForm, category: e.target.value })}
                    className="w-full bg-surface-elevated border border-border rounded-lg px-3 py-1.5 text-xs text-white focus:outline-none focus:border-blue-500"
                  >
                    <option value="commercial">commercial</option>
                    <option value="legal">legal</option>
                    <option value="medical">medical</option>
                    <option value="fintech">fintech</option>
                  </select>
                </div>
              </div>
              <div>
                <label className="block text-xs font-medium text-slate-400 mb-1">Reason / Legal Context</label>
                <textarea
                  rows={2}
                  required
                  value={forbiddenForm.reason}
                  onChange={(e) => setForbiddenForm({ ...forbiddenForm, reason: e.target.value })}
                  placeholder="Compliance and legal exposure reason..."
                  className="w-full bg-surface-elevated border border-border rounded-lg px-3 py-1.5 text-xs text-white focus:outline-none focus:border-blue-500"
                />
              </div>
              <div className="flex justify-end gap-2 pt-2 border-t border-border">
                <button
                  type="button"
                  onClick={() => setIsAddingForbidden(false)}
                  className="px-4 py-2 bg-slate-800 hover:bg-slate-700 text-slate-300 text-xs font-medium rounded-lg"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="px-4 py-2 bg-blue-600 hover:bg-blue-500 text-white text-xs font-medium rounded-lg"
                >
                  Save Forbidden Rule
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}
