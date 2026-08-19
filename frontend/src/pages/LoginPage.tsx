import { useState, type FormEvent } from 'react'
import type { LoginPayload, Principal } from '../auth/types'
import { AuthFrame, authInputClassName, authLabelClassName } from '../components/auth/AuthFrame'
import { APIError, apiRequest } from '../lib/api'

interface LoginPageProps {
  onAuthenticated: (principal: Principal) => void
}

export function LoginPage({ onAuthenticated }: LoginPageProps) {
  const [form, setForm] = useState<LoginPayload>({ email: '', password: '' })
  const [error, setError] = useState<string>()
  const [submitting, setSubmitting] = useState(false)

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setSubmitting(true)
    setError(undefined)

    try {
      const principal = await apiRequest<Principal>('/api/v1/auth/login', {
        method: 'POST',
        body: JSON.stringify(form),
      })
      onAuthenticated(principal)
    } catch (requestError) {
      setError(requestError instanceof APIError ? requestError.message : 'Sign in could not be completed.')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <AuthFrame
      description="Sign in to manage projects, environments and feature flags for your organisations."
      eyebrow="Authentication"
      footer="Session tokens are stored in an HttpOnly cookie and can be revoked by signing out."
      title="Sign in to FlagStack"
    >
      <form className="space-y-4" onSubmit={handleSubmit}>
        <label className={authLabelClassName}>
          Email
          <input
            autoComplete="email"
            autoFocus
            className={authInputClassName}
            onChange={(event) => setForm((current) => ({ ...current, email: event.target.value }))}
            required
            type="email"
            value={form.email}
          />
        </label>

        <label className={authLabelClassName}>
          Password
          <input
            autoComplete="current-password"
            className={authInputClassName}
            onChange={(event) => setForm((current) => ({ ...current, password: event.target.value }))}
            required
            type="password"
            value={form.password}
          />
        </label>

        {error ? (
          <div className="rounded-lg border border-red-900/60 bg-red-950/30 px-3 py-2.5 text-xs leading-5 text-red-300">{error}</div>
        ) : null}

        <button
          className="flex h-10 w-full items-center justify-center rounded-lg bg-flagstack-600 px-4 text-sm font-semibold text-white transition hover:bg-flagstack-500 disabled:cursor-not-allowed disabled:opacity-60"
          disabled={submitting}
          type="submit"
        >
          {submitting ? 'Signing in…' : 'Sign in'}
        </button>
      </form>
    </AuthFrame>
  )
}
