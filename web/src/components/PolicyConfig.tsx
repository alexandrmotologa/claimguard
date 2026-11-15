import React, { useState, useEffect } from 'react'
import { Sliders, CheckCircle2, Save } from 'lucide-react'
import { PolicyConfigData } from '../types'

export const PolicyConfig: React.FC = () => {
  const [policy, setPolicy] = useState<PolicyConfigData>({
    default_action: 'rewrite',
    disclaimer_template: ' [Correction: Official enterprise policy is {{.ExpectedValue}}, not {{.AssertedValue}}]',
    strict_halt_message: 'Stream terminated by compliance filter: factual contradiction detected.',
    numerical_tolerance_percent: 0.0,
  })
  const [loading, setLoading] = useState(true)
  const [savedSuccess, setSavedSuccess] = useState(false)

  useEffect(() => {
    fetch('/api/policy')
      .then((res) => res.json())
      .then((data) => {
        setPolicy(data)
        setLoading(false)
      })
      .catch((err) => {
        console.error('Failed to load policy:', err)
        setLoading(false)
      })
  }, [])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      const res = await fetch('/api/policy', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(policy),
      })
      if (res.ok) {
        setSavedSuccess(true)
        setTimeout(() => setSavedSuccess(false), 3000)
      }
    } catch (err) {
      console.error('Failed to update policy:', err)
    }
  }

  return (
    <div className="bg-surface border border-border rounded-xl p-6">
      <div className="flex items-center justify-between mb-4">
        <div>
          <h2 className="text-lg font-semibold text-white flex items-center gap-2">
            <Sliders className="w-5 h-5 text-blue-400" />
            Runtime Action Policy Configuration
          </h2>
          <p className="text-xs text-slate-400 mt-0.5">
            Configure how the proxy intercepts contradictions in real time. Changes take effect immediately without proxy restart.
          </p>
        </div>
        {savedSuccess && (
          <span className="flex items-center gap-1.5 text-xs text-emerald-400 font-medium bg-emerald-400/10 px-3 py-1 rounded-full border border-emerald-400/20">
            <CheckCircle2 className="w-3.5 h-3.5" />
            Policy Updated
          </span>
        )}
      </div>

      <form onSubmit={handleSubmit} className="space-y-4">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {/* Default Action */}
          <div>
            <label className="block text-xs font-medium text-slate-300 mb-1.5">
              Default Interception Action
            </label>
            <select
              value={policy.default_action}
              onChange={(e) => setPolicy({ ...policy, default_action: e.target.value })}
              className="w-full bg-surface-elevated border border-border rounded-lg px-3 py-2 text-xs text-white focus:outline-none focus:border-blue-500"
            >
              <option value="rewrite">rewrite - In-place token patch before client delivery</option>
              <option value="disclaimer">disclaimer - Append factual disclaimer notice to sentence</option>
              <option value="strict_halt">strict_halt - Immediately terminate stream with compliance refusal</option>
              <option value="flag_only">flag_only - Stream unmodified text but log audit incident</option>
            </select>
          </div>

          {/* Numerical Tolerance */}
          <div>
            <label className="block text-xs font-medium text-slate-300 mb-1.5">
              Numerical Tolerance ({(policy.numerical_tolerance_percent * 100).toFixed(1)}%)
            </label>
            <input
              type="range"
              min="0"
              max="0.1"
              step="0.005"
              value={policy.numerical_tolerance_percent}
              onChange={(e) => setPolicy({ ...policy, numerical_tolerance_percent: parseFloat(e.target.value) })}
              className="w-full h-1.5 bg-slate-700 rounded-lg appearance-none cursor-pointer accent-blue-500"
            />
            <div className="flex justify-between text-[11px] text-slate-400 mt-1">
              <span>0.0% (Exact match)</span>
              <span>5.0%</span>
              <span>10.0% (Relaxed)</span>
            </div>
          </div>
        </div>

        {/* Disclaimer Template */}
        <div>
          <label className="block text-xs font-medium text-slate-300 mb-1.5">
            Disclaimer Template
          </label>
          <input
            type="text"
            value={policy.disclaimer_template}
            onChange={(e) => setPolicy({ ...policy, disclaimer_template: e.target.value })}
            className="w-full bg-surface-elevated border border-border rounded-lg px-3 py-2 text-xs text-white font-mono focus:outline-none focus:border-blue-500"
          />
          <p className="text-[11px] text-slate-400 mt-1">
            Available template tokens: <code className="text-blue-400">{'{{.ExpectedValue}}'}</code>, <code className="text-blue-400">{'{{.AssertedValue}}'}</code>, <code className="text-blue-400">{'{{.EntityName}}'}</code>
          </p>
        </div>

        {/* Strict Halt Message */}
        <div>
          <label className="block text-xs font-medium text-slate-300 mb-1.5">
            Strict Halt Abort Message
          </label>
          <input
            type="text"
            value={policy.strict_halt_message}
            onChange={(e) => setPolicy({ ...policy, strict_halt_message: e.target.value })}
            className="w-full bg-surface-elevated border border-border rounded-lg px-3 py-2 text-xs text-white font-mono focus:outline-none focus:border-blue-500"
          />
        </div>

        <div className="flex justify-end pt-2">
          <button
            type="submit"
            className="flex items-center gap-1.5 px-4 py-2 bg-blue-600 hover:bg-blue-500 text-white text-xs font-medium rounded-lg transition-colors shadow-sm"
          >
            <Save className="w-3.5 h-3.5" />
            <span>Apply Policy Changes</span>
          </button>
        </div>
      </form>
    </div>
  )
}
