import React from 'react'
import { BarChart3, AlertTriangle } from 'lucide-react'
import { MetricsSnapshot } from '../types'

interface Props {
  metrics: MetricsSnapshot | null
}

export const ClaimFrequency: React.FC<Props> = ({ metrics }) => {
  const actions = metrics?.interventions_by_action || {}
  const violations = metrics?.violations_by_entity || {}

  const actionKeys = Object.keys(actions)
  const totalInterventions = actionKeys.reduce((acc, k) => acc + (actions[k] || 0), 0)

  const entityKeys = Object.keys(violations).sort((a, b) => (violations[b] || 0) - (violations[a] || 0))

  return (
    <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
      {/* Policy Actions Breakdown */}
      <div className="bg-surface border border-border rounded-xl p-5">
        <h3 className="text-sm font-semibold text-white flex items-center gap-2 mb-4">
          <BarChart3 className="w-4 h-4 text-blue-400" />
          Active Interventions by Policy Action
        </h3>
        <div className="space-y-3">
          {actionKeys.length === 0 ? (
            <p className="text-xs text-slate-500 py-4 text-center">No interventions recorded yet.</p>
          ) : (
            actionKeys.map((action) => {
              const count = actions[action] || 0
              const pct = totalInterventions > 0 ? (count / totalInterventions) * 100 : 0
              const colorClass =
                action === 'rewrite'
                  ? 'bg-blue-500'
                  : action === 'disclaimer'
                  ? 'bg-amber-500'
                  : action === 'strict_halt'
                  ? 'bg-red-500'
                  : 'bg-purple-500'

              return (
                <div key={action}>
                  <div className="flex justify-between text-xs mb-1">
                    <span className="text-slate-300 font-mono capitalize">{action}</span>
                    <span className="text-slate-400 font-medium">
                      {count} ({pct.toFixed(0)}%)
                    </span>
                  </div>
                  <div className="w-full bg-surface-elevated rounded-full h-2">
                    <div
                      className={`h-2 rounded-full ${colorClass} transition-all duration-500`}
                      style={{ width: `${pct}%` }}
                    />
                  </div>
                </div>
              )
            })
          )}
        </div>
      </div>

      {/* Top Hallucinated Entities */}
      <div className="bg-surface border border-border rounded-xl p-5">
        <h3 className="text-sm font-semibold text-white flex items-center gap-2 mb-4">
          <AlertTriangle className="w-4 h-4 text-amber-400" />
          High-Liability Violation Frequency
        </h3>
        <div className="space-y-3">
          {entityKeys.length === 0 ? (
            <p className="text-xs text-slate-500 py-4 text-center">Zero entity contradictions recorded.</p>
          ) : (
            entityKeys.slice(0, 5).map((entityID) => {
              const count = violations[entityID] || 0
              const maxCount = violations[entityKeys[0]] || 1
              const pct = (count / maxCount) * 100

              return (
                <div key={entityID}>
                  <div className="flex justify-between text-xs mb-1">
                    <span className="text-slate-200 font-mono">{entityID}</span>
                    <span className="text-amber-400 font-semibold">{count} violations</span>
                  </div>
                  <div className="w-full bg-surface-elevated rounded-full h-2">
                    <div
                      className="h-2 rounded-full bg-amber-500 transition-all duration-500"
                      style={{ width: `${pct}%` }}
                    />
                  </div>
                </div>
              )
            })
          )}
        </div>
      </div>
    </div>
  )
}
