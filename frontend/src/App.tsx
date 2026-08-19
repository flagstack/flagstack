import { useCallback, useEffect, useState, type ReactNode } from 'react'
import type { BootstrapStatus, Principal } from './auth/types'
import { AppShell } from './components/layout/AppShell'
import { APIError, apiRequest } from './lib/api'
import { BootstrapPage } from './pages/BootstrapPage'
import { DashboardPage } from './pages/DashboardPage'
import { LoginPage } from './pages/LoginPage'

type AppState =
  | { status: 'loading' }
  | { status: 'bootstrap' }
  | { status: 'login' }
  | { status: 'authenticated'; principal: Principal }
  | { status: 'error'; message: string }

export function App() {
  const [state, setState] = useState<AppState>({ status: 'loading' })

  const initialise = useCallback(async () => {
    setState({ status: 'loading' })
    try {
      const bootstrap = await apiRequest<BootstrapStatus>('/api/v1/bootstrap')
      if (bootstrap.required) {
        setState({ status: 'bootstrap' })
        return
      }

      try {
        const principal = await apiRequest<Principal>('/api/v1/auth/me')
        setState({ status: 'authenticated', principal })
      } catch (error) {
        if (error instanceof APIError && error.status === 401) {
          setState({ status: 'login' })
          return
        }
        throw error
      }
    } catch (error) {
      setState({
        status: 'error',
        message: error instanceof Error ? error.message : 'FlagStack could not initialise.',
      })
    }
  }, [])

  useEffect(() => {
    void initialise()
  }, [initialise])

  if (state.status === 'loading') {
    return <StartupScreen message="Loading FlagStack…" />
  }

  if (state.status === 'error') {
    return (
      <StartupScreen
        action={
          <button
            className="mt-4 rounded-lg bg-flagstack-600 px-3 py-2 text-xs font-semibold text-white transition hover:bg-flagstack-500"
            onClick={() => void initialise()}
            type="button"
          >
            Try again
          </button>
        }
        message={state.message}
      />
    )
  }

  if (state.status === 'bootstrap') {
    return <BootstrapPage onAuthenticated={(principal) => setState({ status: 'authenticated', principal })} />
  }

  if (state.status === 'login') {
    return <LoginPage onAuthenticated={(principal) => setState({ status: 'authenticated', principal })} />
  }

  async function logout() {
    await apiRequest<void>('/api/v1/auth/logout', { method: 'POST' })
    setState({ status: 'login' })
  }

  return (
    <AppShell onLogout={logout} principal={state.principal}>
      <DashboardPage />
    </AppShell>
  )
}

function StartupScreen({ message, action }: { message: string; action?: ReactNode }) {
  return (
    <main className="flex min-h-screen items-center justify-center bg-slate-950 px-4 text-center text-slate-200">
      <div>
        <span className="mx-auto flex h-10 w-10 items-center justify-center rounded-xl bg-gradient-to-br from-flagstack-600 to-indigo-600 text-sm font-black text-white ring-1 ring-flagstack-400/20">
          FS
        </span>
        <p className="mt-4 text-sm text-slate-500">{message}</p>
        {action}
      </div>
    </main>
  )
}
