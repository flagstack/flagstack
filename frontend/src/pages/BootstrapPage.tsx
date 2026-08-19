import { useState, type FormEvent } from 'react'
import type { BootstrapPayload, Principal } from '../auth/types'
import { AuthFrame, authInputClassName, authLabelClassName } from '../components/auth/AuthFrame'
import { APIError, apiRequest } from '../lib/api'

interface BootstrapPageProps {
  onAuthenticated: (principal: Principal) => void
}

function slugify(value: string): string {
  return value
    .toLowerCase()
    .trim()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
    .slice(0, 63)
}

export function BootstrapPage({ onAuthenticated }: BootstrapPageProps) {
  const [form, setForm] = useState<BootstrapPayload>({
    email: '',
    display_name: '',
    password: '',
    organisation_name: '',
    organisation_slug: '',
  })
  const [slugEdited, setSlugEdited] = useState(false)
  const [error, setError] = useState<string>()
  const [submitting, setSubmitting] = useState(false)

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setSubmitting(true)
    setError(undefined)

    try {
      const principal = await apiRequest<Principal>('/api/v1/bootstrap', {
        method: 'POST',
        body: JSON.stringify(form),
      })
      onAuthenticated(principal)
    } catch (requestError) {
      setError(requestError instanceof APIError ? requestError.message : 'Initial setup could not be completed.')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <AuthFrame
      description="Create the first owner account and organisation for this FlagStack installation. This setup route closes permanently after the first account is created."
      eyebrow="First run"
      footer="Local authentication is the self-hosted baseline. Additional identity providers can be added later."
      title="Set up FlagStack"
    >
      <form className="space-y-4" onSubmit={handleSubmit}>
        <div className="grid gap-4 sm:grid-cols-2">
          <label className={authLabelClassName}>
            Your name
            <input
              autoComplete="name"
              autoFocus
              className={authInputClassName}
              maxLength={120}
              onChange={(event) => setForm((current) => ({ ...current, display_name: event.target.value }))}
              required
              value={form.display_name}
            />
          </label>

          <label className={authLabelClassName}>
            Email
            <input
              autoComplete="email"
              className={authInputClassName}
              onChange={(event) => setForm((current) => ({ ...current, email: event.target.value }))}
              required
              type="email"
              value={form.email}
            />
          </label>
        </div>

        <label className={authLabelClassName}>
          Organisation name
          <input
            className={authInputClassName}
            maxLength={160}
            onChange={(event) => {
              const organisationName = event.target.value
              setForm((current) => ({
                ...current,
                organisation_name: organisationName,
                organisation_slug: slugEdited ? current.organisation_slug : slugify(organisationName),
              }))
            }}
            required
            value={form.organisation_name}
          />
        </label>

        <label className={authLabelClassName}>
          Organisation slug
          <input
            className={authInputClassName}
            maxLength={63}
            onChange={(event) => {
              setSlugEdited(true)
              setForm((current) => ({ ...current, organisation_slug: event.target.value.toLowerCase() }))
            }}
            pattern="[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?"
            required
            value={form.organisation_slug}
          />
          <span className="mt-1.5 block text-[11px] text-slate-600">Used in stable organisation URLs and API paths.</span>
        </label>

        <label className={authLabelClassName}>
          Password
          <input
            autoComplete="new-password"
            className={authInputClassName}
            minLength={12}
            onChange={(event) => setForm((current) => ({ ...current, password: event.target.value }))}
            required
            type="password"
            value={form.password}
          />
          <span className="mt-1.5 block text-[11px] text-slate-600">Use at least 12 characters.</span>
        </label>

        {error ? (
          <div className="rounded-lg border border-red-900/60 bg-red-950/30 px-3 py-2.5 text-xs leading-5 text-red-300">{error}</div>
        ) : null}

        <button
          className="flex h-10 w-full items-center justify-center rounded-lg bg-flagstack-600 px-4 text-sm font-semibold text-white transition hover:bg-flagstack-500 disabled:cursor-not-allowed disabled:opacity-60"
          disabled={submitting}
          type="submit"
        >
          {submitting ? 'Creating workspace…' : 'Create FlagStack workspace'}
        </button>
      </form>
    </AuthFrame>
  )
}
