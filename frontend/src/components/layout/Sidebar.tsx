import { Link, NavLink } from 'react-router'
import type { OrganisationMembership } from '../../auth/types'
import { Icon, type IconName } from '../icons'

interface NavigationItem {
  icon: IconName
  label: string
  to: string
  end?: boolean
}

interface NavigationGroup {
  label: string
  items: NavigationItem[]
}

interface SidebarProps {
  collapsed: boolean
  mobileOpen: boolean
  onCloseMobile: () => void
  organisation?: OrganisationMembership
}

const navigation: NavigationGroup[] = [
  {
    label: 'Overview',
    items: [{ label: 'Dashboard', to: '/', icon: 'dashboard', end: true }],
  },
  {
    label: 'Feature management',
    items: [{ label: 'Projects', to: '/projects', icon: 'project' }],
  },
]

function SidebarContent({
  collapsed,
  organisation,
  onNavigate,
  showCloseButton = false,
}: {
  collapsed: boolean
  organisation?: OrganisationMembership
  onNavigate?: () => void
  showCloseButton?: boolean
}) {
  return (
    <>
      <div className="flex h-16 shrink-0 items-center border-b border-slate-800 px-4">
        <Link
          className="flex min-w-0 flex-1 items-center gap-3 overflow-hidden text-inherit no-underline"
          onClick={onNavigate}
          to="/"
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
        </Link>
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
              {group.items.map((item) => (
                <NavLink
                  className={({ isActive }) =>
                    `group flex h-10 items-center rounded-lg text-sm font-medium transition ${
                      collapsed ? 'justify-center px-0' : 'gap-3 px-3'
                    } ${
                      isActive
                        ? 'bg-slate-800 text-white shadow-inner shadow-black/20'
                        : 'text-slate-400 hover:bg-slate-900 hover:text-slate-100'
                    }`
                  }
                  end={item.end}
                  key={item.label}
                  onClick={onNavigate}
                  title={collapsed ? item.label : undefined}
                  to={item.to}
                >
                  {({ isActive }) => (
                    <>
                      <Icon
                        className={`shrink-0 ${
                          isActive ? 'text-flagstack-400' : 'text-slate-500 group-hover:text-slate-300'
                        }`}
                        name={item.icon}
                      />
                      {!collapsed ? <span className="truncate">{item.label}</span> : null}
                    </>
                  )}
                </NavLink>
              ))}
            </div>
          </div>
        ))}
      </nav>

      <div className="border-t border-slate-800 p-4">
        {!collapsed ? (
          <div className="rounded-lg border border-slate-800 bg-slate-900/60 p-3">
            <div className="flex items-center gap-2 text-xs font-medium text-slate-300">
              <span className="h-1.5 w-1.5 rounded-full bg-emerald-400 ring-4 ring-emerald-400/10" />
              <span className="truncate">{organisation?.name ?? 'No organisation'}</span>
            </div>
            <p className="mt-1 truncate text-[11px] leading-4 text-slate-600">
              {organisation ? `${organisation.role} · ${organisation.slug}` : 'No membership available'}
            </p>
          </div>
        ) : (
          <span
            className="mx-auto block h-1.5 w-1.5 rounded-full bg-emerald-400 ring-4 ring-emerald-400/10"
            title={organisation?.name ?? 'No organisation'}
          />
        )}
      </div>
    </>
  )
}

export function Sidebar({ collapsed, mobileOpen, onCloseMobile, organisation }: SidebarProps) {
  return (
    <>
      <aside
        className={`hidden shrink-0 border-r border-slate-800 bg-slate-950 transition-[width] duration-200 lg:flex lg:flex-col ${
          collapsed ? 'w-20' : 'w-72'
        }`}
      >
        <SidebarContent collapsed={collapsed} organisation={organisation} />
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
            <SidebarContent organisation={organisation} collapsed={false} onNavigate={onCloseMobile} showCloseButton />
          </aside>
        </div>
      ) : null}
    </>
  )
}
