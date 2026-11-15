import React, { useState, useEffect } from 'react'
import { Plus, Search, Trash2, Edit3, Database, Save, X } from 'lucide-react'
import { Fact, Entity } from '../types'

export const TruthTableEditor: React.FC = () => {
  const [facts, setFacts] = useState<Fact[]>([])
  const [entities, setEntities] = useState<Entity[]>([])
  const [search, setSearch] = useState('')
  const [loading, setLoading] = useState(true)
  const [editingFact, setEditingFact] = useState<Fact | null>(null)
  const [isAdding, setIsAdding] = useState(false)

  const [formData, setFormData] = useState<Partial<Fact>>({
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

  const loadData = async () => {
    try {
      setLoading(true)
      const [factsRes, entitiesRes] = await Promise.all([
        fetch('/api/facts'),
        fetch('/api/entities'),
      ])
      if (factsRes.ok && entitiesRes.ok) {
        setFacts(await factsRes.json())
        setEntities(await entitiesRes.json())
      }
    } catch (err) {
      console.error('Failed to load truth registry:', err)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadData()
  }, [])

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      const url = editingFact ? `/api/facts/${editingFact.id}` : '/api/facts'
      const method = editingFact ? 'PUT' : 'POST'

      const res = await fetch(url, {
        method,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(formData),
      })

      if (res.ok) {
        setIsAdding(false)
        setEditingFact(null)
        loadData()
      }
    } catch (err) {
      console.error('Save fact error:', err)
    }
  }

  const handleDelete = async (id: string) => {
    if (!confirm('Are you sure you want to delete this authoritative fact?')) return
    try {
      const res = await fetch(`/api/facts/${id}`, { method: 'DELETE' })
      if (res.ok) {
        loadData()
      }
    } catch (err) {
      console.error('Delete fact error:', err)
    }
  }

  const startEdit = (fact: Fact) => {
    setEditingFact(fact)
    setFormData(fact)
    setIsAdding(true)
  }

  const startAdd = () => {
    setEditingFact(null)
    setFormData({
      id: `fact-${Date.now().toString(36)}`,
      entity_id: entities[0]?.id || '',
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
    setIsAdding(true)
  }

  const filteredFacts = facts.filter((f) => {
    const q = search.toLowerCase()
    return (
      f.entity_id.toLowerCase().includes(q) ||
      f.attribute.toLowerCase().includes(q) ||
      f.expected_value.toLowerCase().includes(q) ||
      f.description.toLowerCase().includes(q)
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
            Authoritative enterprise facts stored in-memory in SQLite with sub-millisecond lookup latency.
          </p>
        </div>
        <div className="flex items-center gap-3">
          <div className="relative">
            <Search className="w-4 h-4 text-slate-400 absolute left-3 top-2.5" />
            <input
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Search facts..."
              className="bg-surface-elevated border border-border rounded-lg pl-9 pr-3 py-1.5 text-xs text-white placeholder-slate-500 focus:outline-none focus:border-blue-500 w-48"
            />
          </div>
          <button
            onClick={startAdd}
            className="flex items-center gap-1.5 px-3 py-1.5 bg-blue-600 hover:bg-blue-500 text-white text-xs font-medium rounded-lg transition-colors"
          >
            <Plus className="w-3.5 h-3.5" />
            <span>Add Fact</span>
          </button>
        </div>
      </div>

      {/* Facts Table */}
      <div className="overflow-x-auto">
        <table className="w-full text-left text-xs text-slate-300">
          <thead className="bg-surface-elevated text-slate-400 uppercase tracking-wider text-[11px] border-b border-border">
            <tr>
              <th className="px-4 py-3">Entity &amp; Attribute</th>
              <th className="px-4 py-3">Claim Type</th>
              <th className="px-4 py-3">Authoritative Value</th>
              <th className="px-4 py-3">Action Policy</th>
              <th className="px-4 py-3">Compliance Source</th>
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
                  No ground truth facts matched your query.
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
                        onClick={() => startEdit(fact)}
                        className="p-1 hover:bg-slate-700 text-slate-400 hover:text-white rounded transition-colors"
                        title="Edit Fact"
                      >
                        <Edit3 className="w-3.5 h-3.5" />
                      </button>
                      <button
                        onClick={() => handleDelete(fact.id)}
                        className="p-1 hover:bg-red-500/20 text-slate-400 hover:text-red-400 rounded transition-colors"
                        title="Delete Fact"
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

      {/* Add / Edit Modal */}
      {isAdding && (
        <div className="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center p-4 z-50">
          <div className="bg-surface border border-border rounded-xl w-full max-w-lg p-6 shadow-2xl">
            <div className="flex items-center justify-between mb-4 pb-3 border-b border-border">
              <h3 className="text-base font-semibold text-white">
                {editingFact ? 'Edit Authoritative Fact' : 'Add Ground Truth Fact'}
              </h3>
              <button
                onClick={() => setIsAdding(false)}
                className="text-slate-400 hover:text-white"
              >
                <X className="w-4 h-4" />
              </button>
            </div>

            <form onSubmit={handleSave} className="space-y-4">
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-xs font-medium text-slate-400 mb-1">Entity ID</label>
                  <input
                    type="text"
                    required
                    value={formData.entity_id}
                    onChange={(e) => setFormData({ ...formData, entity_id: e.target.value })}
                    className="w-full bg-surface-elevated border border-border rounded-lg px-3 py-1.5 text-xs text-white focus:outline-none focus:border-blue-500"
                    placeholder="e.g. policy.refund"
                  />
                </div>
                <div>
                  <label className="block text-xs font-medium text-slate-400 mb-1">Attribute</label>
                  <input
                    type="text"
                    required
                    value={formData.attribute}
                    onChange={(e) => setFormData({ ...formData, attribute: e.target.value })}
                    className="w-full bg-surface-elevated border border-border rounded-lg px-3 py-1.5 text-xs text-white focus:outline-none focus:border-blue-500"
                    placeholder="e.g. window_days"
                  />
                </div>
              </div>

              <div className="grid grid-cols-3 gap-3">
                <div>
                  <label className="block text-xs font-medium text-slate-400 mb-1">Claim Type</label>
                  <select
                    value={formData.claim_type}
                    onChange={(e) => setFormData({ ...formData, claim_type: e.target.value })}
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
                    value={formData.expected_value}
                    onChange={(e) => setFormData({ ...formData, expected_value: e.target.value })}
                    className="w-full bg-surface-elevated border border-border rounded-lg px-3 py-1.5 text-xs text-white focus:outline-none focus:border-blue-500"
                    placeholder="e.g. 14 days"
                  />
                </div>
                <div>
                  <label className="block text-xs font-medium text-slate-400 mb-1">Numeric Float</label>
                  <input
                    type="number"
                    step="any"
                    value={formData.numeric_value}
                    onChange={(e) => setFormData({ ...formData, numeric_value: parseFloat(e.target.value) || 0 })}
                    className="w-full bg-surface-elevated border border-border rounded-lg px-3 py-1.5 text-xs text-white focus:outline-none focus:border-blue-500"
                  />
                </div>
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-xs font-medium text-slate-400 mb-1">Policy Action</label>
                  <select
                    value={formData.policy_action}
                    onChange={(e) => setFormData({ ...formData, policy_action: e.target.value })}
                    className="w-full bg-surface-elevated border border-border rounded-lg px-3 py-1.5 text-xs text-white focus:outline-none focus:border-blue-500"
                  >
                    <option value="rewrite">rewrite</option>
                    <option value="disclaimer">disclaimer</option>
                    <option value="strict_halt">strict_halt</option>
                    <option value="flag_only">flag_only</option>
                  </select>
                </div>
                <div>
                  <label className="block text-xs font-medium text-slate-400 mb-1">Compliance Source</label>
                  <input
                    type="text"
                    value={formData.source_doc}
                    onChange={(e) => setFormData({ ...formData, source_doc: e.target.value })}
                    className="w-full bg-surface-elevated border border-border rounded-lg px-3 py-1.5 text-xs text-white focus:outline-none focus:border-blue-500"
                    placeholder="e.g. DOC-LEGAL-2026"
                  />
                </div>
              </div>

              <div>
                <label className="block text-xs font-medium text-slate-400 mb-1">Description</label>
                <textarea
                  rows={2}
                  value={formData.description}
                  onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                  className="w-full bg-surface-elevated border border-border rounded-lg px-3 py-1.5 text-xs text-white focus:outline-none focus:border-blue-500"
                  placeholder="Official enterprise policy background..."
                />
              </div>

              <div className="flex justify-end gap-2 pt-2 border-t border-border">
                <button
                  type="button"
                  onClick={() => setIsAdding(false)}
                  className="px-4 py-2 bg-slate-800 hover:bg-slate-700 text-slate-300 text-xs font-medium rounded-lg transition-colors"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="flex items-center gap-1.5 px-4 py-2 bg-blue-600 hover:bg-blue-500 text-white text-xs font-medium rounded-lg transition-colors shadow-sm"
                >
                  <Save className="w-3.5 h-3.5" />
                  <span>Save Fact</span>
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}
