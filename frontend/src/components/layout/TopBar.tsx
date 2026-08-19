import { Icon } from '../icons'

interface TopBarProps {
  collapsed: boolean
  onOpenMobileNavigation: () => void
  onToggleSidebar: () => void
}

export function TopBar({ collapsed, onOpenMobileNavigation, onToggleSidebar }: TopBarProps) {
  return (
    <header className="topbar">
      <button
        aria-label={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
        className="icon-button desktop-sidebar-toggle"
        onClick={onToggleSidebar}
        type="button"
      >
        <Icon name={collapsed ? 'chevron-right' : 'chevron-left'} />
      </button>

      <button
        aria-label="Open navigation"
        className="icon-button mobile-menu-button"
        onClick={onOpenMobileNavigation}
        type="button"
      >
        <Icon name="menu" />
      </button>

      <button
        className="global-search"
        title="Global search will cover projects, flags, members and documentation"
        type="button"
      >
        <Icon name="search" size={17} />
        <span>Search FlagStack...</span>
        <kbd>⌘K</kbd>
      </button>

      <div className="topbar-actions">
        <button aria-label="Notifications" className="icon-button" type="button">
          <Icon name="bell" />
        </button>

        <details className="account-menu">
          <summary>
            <span className="account-avatar">A</span>
            <span className="account-copy">
              <strong>Local admin</strong>
              <small>Development</small>
            </span>
          </summary>
          <div className="account-popover">
            <div className="account-popover-heading">
              <strong>Local admin</strong>
              <span>Authentication and account controls will replace this development identity in a focused follow-up.</span>
            </div>
          </div>
        </details>
      </div>
    </header>
  )
}
