import type { HTMLAttributes, ReactNode } from 'react'

function classes(...values: Array<string | undefined>) {
  return values.filter(Boolean).join(' ')
}

export function Card({ className, ...props }: HTMLAttributes<HTMLDivElement>) {
  return (
    <section
      className={classes(
        'overflow-hidden rounded-xl border border-slate-800 bg-slate-900/70 shadow-sm shadow-black/10',
        className,
      )}
      {...props}
    />
  )
}

export function CardHeader({ className, ...props }: HTMLAttributes<HTMLDivElement>) {
  return <div className={classes('p-5', className)} {...props} />
}

export function CardTitle({ children }: { children: ReactNode }) {
  return <h2 className="text-sm font-semibold text-slate-200">{children}</h2>
}

export function CardContent({ className, ...props }: HTMLAttributes<HTMLDivElement>) {
  return <div className={classes('border-t border-slate-800', className)} {...props} />
}
