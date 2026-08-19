import { Icon, type IconName } from '../icons'

interface NavigationItem {
  icon: IconName
  label: string
  href: string
}

interface NavigationGroup {
  label: string
  items: NavigationItem[]
}

interface SidebarProps {
  collapsed: boolean
  mobileOpen: boolean
  onCloseMobile: () => void
}

const navigation: NavigationGroup[] = [
  {
    label: 'Overview',
    items: [{ label: 'Dashboard', href: '#dashboard', icon: 'dashboard' }],
  },
  {
    label: 'Feature management',
    items: [
      { label: 'Projects', href: '#projects', icon: 'project' },
      { label: 'Feature flags', href: '#feature-flags', icon: 'flag' },
      { label: 'Environments', href: '#environments', icon: 'environment' },
    ],
  },
]

function SidebarContent({
  collapsed,
  onNavigate,
  showCloseButton = false,
}: {
  collapsed: boolean
  onNavigate?: () => void
  showCloseButton?: boolean
}) {
  return (
    <>
      <div className="flex h-16 shrink-0 items-center border-b border-slate-800 px-4">
        <a
          className="flex min-w-0 flex-1 items-center gap-3 overflow-hidden text-inherit no-underline"
          href="#dashboard"
          onClick={onNavigate}
        >
          <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-gradient-to-br from-flagstack-600 to-indigo-600 text-xs font-black tracking-tight text-white shadow-lg shadow-violet-950/30 ring-1 ring-flagstack-400/20">
            FS
          </span>
          {!collapsed ? (
            <span className="min-w-0">
              <strong className="block truncate text-sm font-semibold text-white">FlagStack</strong>
              <small className="mt-0.5 block truncate text-xs text-slate-500">Feature management</small>
            </span>
          ) : null}
        </a>
        {showCloseButton ? (
          <button
            aria-label="Close navigation"
            className="ml-3 flex h-9 w-9 items-center justify-center rounded-lg text-slate-400 transition hover:bg-slate-800 hover:text-white focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-flagstack-400"
            onClick={onNavigate}
            type="button"
          >
            <Icon name="close" />
          </button>
        ) : null}
      </div>

      <nav className="flex-1 space-y-6 overflow-y-auto px-3 py-5" aria-label="Primary navigation">
        {navigation.map((group) => (
          <div key={group.label}>
            {!collapsed ? (
              <p className="mb-2 px-3 text-[10px] font-bold tracking-[0.18em] text-slate-600 uppercase">
                {group.label}
              </p>
            ) : null}
            <div className="space-y-1">
              {group.items.map((item) => {
                const active = item.href === '#dashboard'
                return (
                  <a
                    aria-current={active ? 'page' : undefined}
                    className={`group flex h-10 items-center rounded-lg text-sm font-medium transition ${
                      collapsed ? 'justify-center px-0' : 'gap-3 px-3'
                    } ${
                      active
                        ? 'bg-slate-800 text-white shadow-inner shadow-black/20'
                        : 'text-slate-400 hover:bg-slate-900 hover:text-slate-100'
                    }`}
                    href={item.href}
                    key={item.label}
                    onClick={onNavigate}
                    title={collapsed ? item.label : undefined}
                  >
                    <Icon
                      className={`shrink-0 ${
                        active ? 'text-flagstack-400' : 'text-slate-500 group-hover:text-slate-300'
                      }`}
                      name={item.icon}
                    />
                    {!collapsed ? <span className="truncate">{item.label}</span> : null}
                  </a>
                )
              })}
            </div>
          </div>
        ))}
      </nav>

      <div className="border-t border-slate-800 p-4">
        {!collapsed ? (
          <div className="rounded-lg border border-slate-800 bg-slate-900/60 p-3">
            <div className="flex items-center gap-2 text-xs font-medium text-slate-300">
              <span className="h-1.5 w-1.5 rounded-full bg-emerald-400 ring-4 ring-emerald-400/10" />
              Local development
            </div>
            <p className="mt-1 text-[11px] leading-4 text-slate-600">Self-hosted core · Early preview</p>
          </div>
        ) : (
          <span
            className="mx-auto block h-1.5 w-1.5 rounded-full bg-emerald-400 ring-4 ring-emerald-400/10"
            title="Local development"
          />
        )}
      </div>
    </>
  )
}

export function Sidebar({ collapsed, mobileOpen, onCloseMobile }: SidebarProps) {
  return (
    <>
      <aside
        className={`hidden shrink-0 border-r border-slate-800 bg-slate-950 transition-[width] duration-200 lg:flex lg:flex-col ${
          collapsed ? 'w-20' : 'w-72'
        }`}
      >
        <SidebarContent collapsed={collapsed} />
      </aside>

      {mobileOpen ? (
        <div className="fixed inset-0 z-50 lg:hidden">
          <button
            aria-label="Close navigation"
            className="absolute inset-0 bg-black/70 backdrop-blur-sm"
            onClick={onCloseMobile}
            type="button"
          />
          <aside className="relative flex h-full w-[min(88vw,20rem)] flex-col border-r border-slate-800 bg-slate-950 shadow-2xl shadow-black/50">
            <SidebarContent collapsed={false} onNavigate={onCloseMobile} showCloseButton />
          </aside>
        </div>
      ) : null}
    </>
  )
}
