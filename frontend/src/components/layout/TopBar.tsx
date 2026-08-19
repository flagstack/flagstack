import { Icon } from '../icons'

interface TopBarProps {
  collapsed: boolean
  onOpenMobileNavigation: () => void
  onToggleSidebar: () => void
}

export function TopBar({ collapsed, onOpenMobileNavigation, onToggleSidebar }: TopBarProps) {
  return (
    <header className="sticky top-0 z-40 flex h-16 shrink-0 items-center gap-4 border-b border-slate-800 bg-slate-900/95 px-4 backdrop-blur lg:px-6">
      <button
        aria-label={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
        className="hidden h-9 w-9 items-center justify-center rounded-lg text-slate-400 transition hover:bg-slate-800 hover:text-white focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-flagstack-400 lg:flex"
        onClick={onToggleSidebar}
        type="button"
      >
        <Icon name={collapsed ? 'chevron-right' : 'chevron-left'} />
      </button>

      <button
        aria-label="Open navigation"
        className="flex h-9 w-9 items-center justify-center rounded-lg text-slate-400 transition hover:bg-slate-800 hover:text-white focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-flagstack-400 lg:hidden"
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
        <span className="truncate">Search FlagStack...</span>
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

        <div className="flex items-center gap-2 rounded-lg p-1.5">
          <span className="flex h-8 w-8 items-center justify-center rounded-lg bg-gradient-to-br from-flagstack-600 to-indigo-600 text-xs font-black text-white ring-1 ring-flagstack-400/20">
            A
          </span>
          <span className="hidden min-w-0 text-left xl:block">
            <span className="block max-w-40 truncate text-sm font-medium text-slate-200">Local admin</span>
            <span className="block max-w-40 truncate text-[11px] text-slate-500">Development</span>
          </span>
        </div>
      </div>
    </header>
  )
}
