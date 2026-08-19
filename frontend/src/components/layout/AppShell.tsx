import { useState, type ReactNode } from 'react'
import type { Principal } from '../../auth/types'
import { Sidebar } from './Sidebar'
import { TopBar } from './TopBar'

interface AppShellProps {
  children: ReactNode
  principal: Principal
  onLogout: () => Promise<void>
}

export function AppShell({ children, principal, onLogout }: AppShellProps) {
  const [collapsed, setCollapsed] = useState(false)
  const [mobileOpen, setMobileOpen] = useState(false)
  const primaryOrganisation = principal.organisations[0]

  return (
    <div className="flex h-screen overflow-hidden bg-slate-950 text-slate-200">
      <Sidebar
        collapsed={collapsed}
        mobileOpen={mobileOpen}
        onCloseMobile={() => setMobileOpen(false)}
        organisation={primaryOrganisation}
      />
      <div className="flex min-w-0 flex-1 flex-col">
        <TopBar
          collapsed={collapsed}
          onLogout={onLogout}
          onOpenMobileNavigation={() => setMobileOpen(true)}
          onToggleSidebar={() => setCollapsed((value) => !value)}
          organisation={primaryOrganisation}
          user={principal.user}
        />
        <main className="min-h-0 flex-1 overflow-y-auto bg-slate-950">{children}</main>
      </div>
    </div>
  )
}
