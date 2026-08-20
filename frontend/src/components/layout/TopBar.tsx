import { useState } from 'react'
import type { OrganisationMembership, UserIdentity } from '../../auth/types'
import { Icon } from '../icons'

interface TopBarProps {
  collapsed: boolean
  organisation?: OrganisationMembership
  user: UserIdentity
  onLogout: () => Promise<void>
  onOpenMobileNavigation: () => void
  onToggleSidebar: () => void
}

function initials(name: string, email: string): string {
  const source = name.trim() || email.split('@')[0] || 'U'
  const words = source.split(/\s+/).filter(Boolean)
  return words.slice(0, 2).map((word) => word[0]?.toUpperCase()).join('') || 'U'
}

export function TopBar({
  collapsed,
  organisation,
  user,
  onLogout,
  onOpenMobileNavigation,
  onToggleSidebar,
}: TopBarProps) {
  const [logoutError, setLogoutError] = useState<string>()
  const [loggingOut, setLoggingOut] = useState(false)

  async function handleLogout() {
    setLoggingOut(true)
    setLogoutError(undefined)
    try {
      await onLogout()
    } catch {
      setLogoutError('Sign out failed. Please try again.')
      setLoggingOut(false)
    }
  }

  return (
    <header className="sticky top-0 z-40 flex h-16 shrink-0 items-center gap-4 border-b border-slate-800 bg-slate-900/95 px-4 backdrop-blur lg:px-6">
      <button
        aria-label={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
        className="hidden h-9 w-9 items-center justify-center rounded-lg text-slate-400 transition hover:bg-slate-800 hover:text-white focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-switchonyourcode-400 lg:flex"
        onClick={onToggleSidebar}
        type="button"
      >
        <Icon name={collapsed ? 'chevron-right' : 'chevron-left'} />
      </button>

      <button
        aria-label="Open navigation"
        className="flex h-9 w-9 items-center justify-center rounded-lg text-slate-400 transition hover:bg-slate-800 hover:text-white focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-switchonyourcode-400 lg:hidden"
        onClick={onOpenMobileNavigation}
        type="button"
      >
        <Icon name="menu" />
      </button>

      <button
        className="group flex h-9 min-w-0 flex-1 cursor-not-allowed items-center gap-2 rounded-lg border border-slate-800 bg-slate-950/70 px-3 text-left text-sm text-slate-500 opacity-70 md:max-w-xl"
        disabled
        title="Global search will cover projects, flags, members and documentation"
        type="button"
      >
        <Icon name="search" size={17} />
        <span className="truncate">Search SwitchOnYourCode...</span>
        <kbd className="ml-auto hidden rounded border border-slate-700 bg-slate-900 px-1.5 py-0.5 text-[10px] text-slate-600 sm:block">
          ⌘K
        </kbd>
      </button>

      <div className="ml-auto flex items-center gap-2">
        <button
          aria-label="Notifications"
          className="flex h-9 w-9 cursor-not-allowed items-center justify-center rounded-lg text-slate-500 opacity-70"
          disabled
          type="button"
        >
          <Icon name="bell" />
        </button>

        <details className="group relative">
          <summary className="flex cursor-pointer list-none items-center gap-2 rounded-lg p-1.5 transition hover:bg-slate-800 [&::-webkit-details-marker]:hidden">
            <span className="flex h-8 w-8 items-center justify-center rounded-lg bg-gradient-to-br from-switchonyourcode-600 to-indigo-600 text-xs font-black text-white ring-1 ring-switchonyourcode-400/20">
              {initials(user.display_name, user.email)}
            </span>
            <span className="hidden min-w-0 text-left xl:block">
              <span className="block max-w-40 truncate text-sm font-medium text-slate-200">{user.display_name}</span>
              <span className="block max-w-40 truncate text-[11px] text-slate-500">
                {organisation ? `${organisation.name} · ${organisation.role}` : user.email}
              </span>
            </span>
            <Icon className="hidden text-slate-600 xl:block" name="chevron-down" size={14} />
          </summary>

          <div className="absolute top-[calc(100%+0.5rem)] right-0 hidden w-64 overflow-hidden rounded-xl border border-slate-800 bg-slate-950 shadow-2xl shadow-black/40 group-open:block">
            <div className="border-b border-slate-800 px-4 py-3">
              <strong className="block truncate text-sm font-medium text-slate-200">{user.display_name}</strong>
              <span className="mt-0.5 block truncate text-xs text-slate-500">{user.email}</span>
            </div>
            {organisation ? (
              <div className="border-b border-slate-800 px-4 py-3">
                <span className="block truncate text-xs font-medium text-slate-300">{organisation.name}</span>
                <span className="mt-0.5 block text-[11px] text-slate-600">{organisation.role} · {organisation.slug}</span>
              </div>
            ) : null}
            <div className="p-2">
              <button
                className="flex w-full items-center gap-2 rounded-lg px-3 py-2 text-left text-xs font-medium text-slate-400 transition hover:bg-slate-900 hover:text-white disabled:cursor-not-allowed disabled:opacity-60"
                disabled={loggingOut}
                onClick={() => void handleLogout()}
                type="button"
              >
                <Icon name="logout" size={15} />
                {loggingOut ? 'Signing out…' : 'Sign out'}
              </button>
              {logoutError ? <p className="px-3 pt-2 pb-1 text-[11px] leading-4 text-red-400">{logoutError}</p> : null}
            </div>
          </div>
        </details>
      </div>
    </header>
  )
}
