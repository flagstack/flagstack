import type { ReactNode } from 'react'

interface AuthFrameProps {
  eyebrow: string
  title: string
  description: string
  children: ReactNode
  footer?: ReactNode
}

export function AuthFrame({ eyebrow, title, description, children, footer }: AuthFrameProps) {
  return (
    <main className="flex min-h-screen items-center justify-center bg-slate-950 px-4 py-10 text-slate-200">
      <div className="w-full max-w-md">
        <div className="mb-6 flex items-center gap-3">
          <span className="flex h-10 w-10 items-center justify-center rounded-xl bg-gradient-to-br from-flagstack-600 to-indigo-600 text-sm font-black tracking-tight text-white shadow-lg shadow-violet-950/30 ring-1 ring-flagstack-400/20">
            FS
          </span>
          <div>
            <strong className="block text-sm font-semibold text-white">FlagStack</strong>
            <span className="text-xs text-slate-500">Feature management</span>
          </div>
        </div>

        <section className="overflow-hidden rounded-2xl border border-slate-800 bg-slate-900/70 shadow-2xl shadow-black/20">
          <div className="border-b border-slate-800 px-6 py-5">
            <p className="text-[10px] font-bold tracking-[0.18em] text-flagstack-400 uppercase">{eyebrow}</p>
            <h1 className="mt-2 text-xl font-semibold tracking-tight text-white">{title}</h1>
            <p className="mt-2 text-sm leading-6 text-slate-500">{description}</p>
          </div>
          <div className="px-6 py-6">{children}</div>
          {footer ? <div className="border-t border-slate-800 px-6 py-4 text-xs text-slate-600">{footer}</div> : null}
        </section>
      </div>
    </main>
  )
}

export const authInputClassName =
  'mt-1.5 h-10 w-full rounded-lg border border-slate-700 bg-slate-950 px-3 text-sm text-slate-200 outline-none transition placeholder:text-slate-700 focus:border-flagstack-500 focus:ring-2 focus:ring-flagstack-500/15'

export const authLabelClassName = 'block text-xs font-medium text-slate-300'
