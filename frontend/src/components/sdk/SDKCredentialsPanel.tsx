import { useState, type FormEvent } from 'react'
import type { OrganisationMembership } from '../../auth/types'
import type { Environment } from '../../environment/types'
import { APIError, apiRequest } from '../../lib/api'
import type { CreatedSDKCredential, SDKCredential, SDKCredentialKind } from '../../sdkconfig/types'
import { Icon } from '../icons'
import { Card, CardContent, CardHeader, CardTitle } from '../ui/Card'
import { SDKQuickstart } from './SDKQuickstart'

interface SDKCredentialsPanelProps {
  canManage: boolean
  credentials: SDKCredential[]
  environments: Environment[]
  onCreated: (credential: SDKCredential) => void
  onRevoked: (credential: SDKCredential) => void
  organisation: OrganisationMembership
  projectID: string
}

export function SDKCredentialsPanel({
  canManage,
  credentials,
  environments,
  onCreated,
  onRevoked,
  organisation,
  projectID,
}: SDKCredentialsPanelProps) {
  const [creating, setCreating] = useState(false)
  const [name, setName] = useState('')
  const [kind, setKind] = useState<SDKCredentialKind>('server')
  const [environmentID, setEnvironmentID] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [revokingID, setRevokingID] = useState<string>()
  const [error, setError] = useState<string>()
  const [revealedKey, setRevealedKey] = useState<{ key: string; kind: SDKCredentialKind }>()

  const selectedEnvironmentID = environmentID || environments[0]?.id || ''

  async function createCredential(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!selectedEnvironmentID) {
      return
    }
    setSubmitting(true)
    setError(undefined)
    try {
      const created = await apiRequest<CreatedSDKCredential>(
        `/api/v1/organisations/${encodeURIComponent(organisation.slug)}/projects/${encodeURIComponent(projectID)}/environments/${encodeURIComponent(selectedEnvironmentID)}/sdk-credentials`,
        { method: 'POST', body: JSON.stringify({ name, kind }) },
      )
      onCreated(created.credential)
      setRevealedKey({ key: created.key, kind: created.credential.kind })
      setName('')
      setCreating(false)
    } catch (requestError) {
      setError(requestError instanceof APIError ? requestError.message : 'SDK credential could not be created.')
    } finally {
      setSubmitting(false)
    }
  }

  async function revokeCredential(credential: SDKCredential) {
    setRevokingID(credential.id)
    setError(undefined)
    try {
      const revoked = await apiRequest<SDKCredential>(
        `/api/v1/organisations/${encodeURIComponent(organisation.slug)}/projects/${encodeURIComponent(projectID)}/sdk-credentials/${encodeURIComponent(credential.id)}/revoke`,
        { method: 'POST' },
      )
      onRevoked(revoked)
    } catch (requestError) {
      setError(requestError instanceof APIError ? requestError.message : 'SDK credential could not be revoked.')
    } finally {
      setRevokingID(undefined)
    }
  }

  return (
    <Card className="mt-4">
      <CardHeader className="flex items-start justify-between gap-4 border-b border-slate-800">
        <div>
          <CardTitle>SDK credentials</CardTitle>
          <p className="mt-1 text-xs text-slate-500">
            Environment-scoped keys for downloading configuration and evaluating flags locally.
          </p>
        </div>
        {canManage && environments.length > 0 ? (
          <button
            className="rounded-lg bg-switchonyourcode-600 px-3 py-1.5 text-xs font-semibold text-white transition hover:bg-switchonyourcode-500"
            onClick={() => setCreating((value) => !value)}
            type="button"
          >
            {creating ? 'Close' : 'Create SDK key'}
          </button>
        ) : null}
      </CardHeader>

      {revealedKey ? (
        <div className="border-b border-slate-800 bg-amber-950/20 px-5 py-4">
          <div className="flex items-start gap-3">
            <Icon className="mt-0.5 text-amber-400" name="key" size={17} />
            <div className="min-w-0 flex-1">
              <strong className="text-xs font-semibold text-amber-200">
                {revealedKey.kind === 'server' ? 'Copy this server key now' : 'Client key created'}
              </strong>
              <p className="mt-1 text-xs leading-5 text-amber-200/60">
                {revealedKey.kind === 'server'
                  ? 'The secret is stored only as a digest and will not be shown again.'
                  : 'Client keys are public identifiers and may be embedded in browser applications.'}
              </p>
              <code className="mt-3 block overflow-x-auto rounded-lg border border-amber-900/50 bg-slate-950 px-3 py-2 text-xs text-amber-100">
                {revealedKey.key}
              </code>
            </div>
            <button
              className="text-xs font-medium text-amber-300/70 hover:text-amber-200"
              onClick={() => setRevealedKey(undefined)}
              type="button"
            >
              Dismiss
            </button>
          </div>
          <SDKQuickstart kind={revealedKey.kind} sdkKey={revealedKey.key} />
        </div>
      ) : null}

      {creating ? (
        <form className="grid gap-4 border-b border-slate-800 bg-slate-950/30 p-5 md:grid-cols-3" onSubmit={createCredential}>
          <label className="block">
            <span className="mb-1.5 block text-xs font-medium text-slate-400">Name</span>
            <input
              className="w-full rounded-lg border border-slate-800 bg-slate-950 px-3 py-2 text-sm text-slate-200 outline-none transition focus:border-switchonyourcode-500"
              maxLength={160}
              onChange={(event) => setName(event.target.value)}
              placeholder="Production backend"
              required
              value={name}
            />
          </label>
          <label className="block">
            <span className="mb-1.5 block text-xs font-medium text-slate-400">Environment</span>
            <select
              className="w-full rounded-lg border border-slate-800 bg-slate-950 px-3 py-2 text-sm text-slate-200 outline-none transition focus:border-switchonyourcode-500"
              onChange={(event) => setEnvironmentID(event.target.value)}
              value={selectedEnvironmentID}
            >
              {environments.map((environment) => (
                <option key={environment.id} value={environment.id}>{environment.name}</option>
              ))}
            </select>
          </label>
          <label className="block">
            <span className="mb-1.5 block text-xs font-medium text-slate-400">Key type</span>
            <select
              className="w-full rounded-lg border border-slate-800 bg-slate-950 px-3 py-2 text-sm text-slate-200 outline-none transition focus:border-switchonyourcode-500"
              onChange={(event) => setKind(event.target.value as SDKCredentialKind)}
              value={kind}
            >
              <option value="server">Server — secret</option>
              <option value="client">Client — public</option>
            </select>
          </label>
          <div className="flex items-center justify-between gap-4 md:col-span-3">
            <p className="text-xs text-slate-600">
              {kind === 'server'
                ? 'Use on trusted servers only. The full secret is returned once.'
                : 'Safe to identify a client SDK, but only client-visible flags will be delivered.'}
            </p>
            <button
              className="rounded-lg bg-switchonyourcode-600 px-3 py-2 text-xs font-semibold text-white transition hover:bg-switchonyourcode-500 disabled:opacity-50"
              disabled={submitting}
              type="submit"
            >
              {submitting ? 'Creating…' : 'Create key'}
            </button>
          </div>
        </form>
      ) : null}

      <CardContent>
        {error ? (
          <div className="m-5 rounded-lg border border-red-900/60 bg-red-950/30 px-3 py-2.5 text-xs text-red-300">{error}</div>
        ) : null}
        {credentials.length === 0 ? (
          <div className="flex min-h-36 flex-col items-center justify-center px-5 py-8 text-center">
            <Icon className="text-slate-700" name="key" size={22} />
            <strong className="mt-3 text-sm font-medium text-slate-300">No SDK credentials yet</strong>
            <p className="mt-1 text-xs text-slate-600">Create a server or client key for one of this project&apos;s environments.</p>
          </div>
        ) : (
          <div className="divide-y divide-slate-800">
            {credentials.map((credential) => {
              const environment = environments.find((candidate) => candidate.id === credential.environment_id)
              const revoked = Boolean(credential.revoked_at)
              return (
                <div className="flex flex-col gap-3 px-5 py-4 lg:flex-row lg:items-center" key={credential.id}>
                  <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-slate-800/60 text-slate-400">
                    <Icon name="key" size={17} />
                  </span>
                  <div className="min-w-0 flex-1">
                    <div className="flex flex-wrap items-center gap-2">
                      <strong className="text-sm font-medium text-slate-200">{credential.name}</strong>
                      <span className="rounded border border-slate-800 bg-slate-950 px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-slate-500">
                        {credential.kind}
                      </span>
                      {revoked ? <span className="text-[10px] font-semibold uppercase tracking-wide text-red-400">Revoked</span> : null}
                    </div>
                    <p className="mt-1 text-xs text-slate-600">
                      {environment?.name ?? 'Unknown environment'} · {credential.kind === 'client' ? 'Public client identifier' : 'Secret stored as digest'}
                    </p>
                    {credential.kind === 'client' && credential.client_key ? (
                      <code className="mt-2 block max-w-full overflow-x-auto text-[11px] text-slate-500">{credential.client_key}</code>
                    ) : null}
                  </div>
                  {canManage && !revoked ? (
                    <button
                      className="self-start rounded-lg border border-red-900/50 px-2.5 py-1.5 text-xs font-medium text-red-400 transition hover:bg-red-950/30 disabled:opacity-50 lg:self-auto"
                      disabled={revokingID === credential.id}
                      onClick={() => void revokeCredential(credential)}
                      type="button"
                    >
                      {revokingID === credential.id ? 'Revoking…' : 'Revoke'}
                    </button>
                  ) : null}
                </div>
              )
            })}
          </div>
        )}
      </CardContent>
    </Card>
  )
}
