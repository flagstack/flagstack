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

const accentClasses: Record<Accent, string> = {
  blue: 'bg-blue-500/10 text-blue-400',
  green: 'bg-emerald-500/10 text-emerald-400',
  orange: 'bg-amber-500/10 text-amber-400',
  red: 'bg-red-500/10 text-red-400',
  violet: 'bg-flagstack-500/10 text-flagstack-400',
}

export function StatCard({ accent = 'violet', helper, icon, label, value }: StatCardProps) {
  return (
    <Card className="flex min-h-28 items-start justify-between gap-4 p-5">
      <div className="min-w-0">
        <p className="text-xs font-medium text-slate-500">{label}</p>
        <p className="mt-2 text-2xl font-semibold tracking-tight text-white">{value}</p>
        {helper ? <p className="mt-1 truncate text-xs text-slate-600">{helper}</p> : null}
      </div>
      {icon ? (
        <div className={`flex h-9 w-9 shrink-0 items-center justify-center rounded-lg ${accentClasses[accent]}`}>
          {icon}
        </div>
      ) : null}
    </Card>
  )
}
