import type { ReactNode } from 'react'
import { Card } from './Card'

type Accent = 'blue' | 'green' | 'orange' | 'red' | 'violet'

interface StatCardProps {
  accent?: Accent
  helper?: string
  icon?: ReactNode
  label: string
  value: string
}

export function StatCard({ accent = 'violet', helper, icon, label, value }: StatCardProps) {
  return (
    <Card className="stat-card">
      <div className="stat-copy">
        <p className="stat-label">{label}</p>
        <p className="stat-value">{value}</p>
        {helper ? <p className="stat-helper">{helper}</p> : null}
      </div>
      {icon ? <div className={`stat-icon stat-icon--${accent}`}>{icon}</div> : null}
    </Card>
  )
}
