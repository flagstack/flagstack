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
      <div className="sidebar-header">
        <a className="brand" href="#dashboard" onClick={onNavigate}>
          <span className="brand-mark">FS</span>
          {!collapsed ? (
            <span className="brand-copy">
              <strong>FlagStack</strong>
              <small>Feature management</small>
            </span>
          ) : null}
        </a>
        {showCloseButton ? (
          <button
            aria-label="Close navigation"
            className="icon-button sidebar-close"
            onClick={onNavigate}
            type="button"
          >
            <Icon name="close" />
          </button>
        ) : null}
      </div>

      <nav className="sidebar-nav" aria-label="Primary navigation">
        {navigation.map((group) => (
          <div className="nav-group" key={group.label}>
            {!collapsed ? <p className="nav-group-label">{group.label}</p> : null}
            <div className="nav-items">
              {group.items.map((item) => {
                const active = item.href === '#dashboard'
                return (
                  <a
                    aria-current={active ? 'page' : undefined}
                    className={`nav-item${active ? ' nav-item--active' : ''}`}
                    href={item.href}
                    key={item.label}
                    onClick={onNavigate}
                    title={collapsed ? item.label : undefined}
                  >
                    <Icon name={item.icon} />
                    {!collapsed ? <span>{item.label}</span> : null}
                  </a>
                )
              })}
            </div>
          </div>
        ))}
      </nav>

      <div className="sidebar-footer">
        {!collapsed ? (
          <div className="sidebar-meta">
            <div className="sidebar-meta-row">
              <span className="status-dot status-dot--healthy" />
              <span>Local development</span>
            </div>
            <p>Self-hosted core · Early preview</p>
          </div>
        ) : (
          <span className="status-dot status-dot--healthy sidebar-status-dot" title="Local development" />
        )}
      </div>
    </>
  )
}

export function Sidebar({ collapsed, mobileOpen, onCloseMobile }: SidebarProps) {
  return (
    <>
      <aside className={`sidebar${collapsed ? ' sidebar--collapsed' : ''}`}>
        <SidebarContent collapsed={collapsed} />
      </aside>

      {mobileOpen ? (
        <div className="mobile-navigation">
          <button
            aria-label="Close navigation"
            className="mobile-navigation-backdrop"
            onClick={onCloseMobile}
            type="button"
          />
          <aside className="mobile-sidebar">
            <SidebarContent
              collapsed={false}
              onNavigate={onCloseMobile}
              showCloseButton
            />
          </aside>
        </div>
      ) : null}
    </>
  )
}
