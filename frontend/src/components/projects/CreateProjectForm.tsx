import { useState, type FormEvent } from 'react'
import type { OrganisationMembership } from '../../auth/types'
import type { CreateProjectPayload, Project } from '../../project/types'
import { APIError, apiRequest } from '../../lib/api'

interface CreateProjectFormProps {
  organisation: OrganisationMembership
  onCancel: () => void
  onCreated: (project: Project) => void
}

function projectKey(value: string): string {
  return value
    .toLowerCase()
    .trim()
    .replace(/[^a-z0-9_-]+/g, '-')
    .replace(/^[-_]+|[-_]+$/g, '')
    .slice(0, 64)
}

export function CreateProjectForm({ organisation, onCancel, onCreated }: CreateProjectFormProps) {
  const [form, setForm] = useState<CreateProjectPayload>({ name: '', key: '', description: '' })
  const [keyEdited, setKeyEdited] = useState(false)
  const [error, setError] = useState<string>()
  const [submitting, setSubmitting] = useState(false)

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setSubmitting(true)
    setError(undefined)

    try {
      const project = await apiRequest<Project>(
        `/api/v1/organisations/${encodeURIComponent(organisation.slug)}/projects`,
        { method: 'POST', body: JSON.stringify(form) },
      )
      onCreated(project)
    } catch (requestError) {
      setError(requestError instanceof APIError ? requestError.message : 'Project could not be created.')
    } finally {
      setSubmitting(false)
    }
  }

  const inputClassName =
    'mt-1.5 h-9 w-full rounded-lg border border-slate-700 bg-slate-950 px-3 text-sm text-slate-200 outline-none transition placeholder:text-slate-700 focus:border-switchonyourcode-500 focus:ring-2 focus:ring-switchonyourcode-500/15'

  return (
    <form className="border-b border-slate-800 bg-slate-950/35 px-5 py-4" onSubmit={handleSubmit}>
      <div className="grid gap-4 md:grid-cols-2">
        <label className="text-xs font-medium text-slate-300">
          Project name
          <input
            autoFocus
            className={inputClassName}
            maxLength={160}
            onChange={(event) => {
              const name = event.target.value
              setForm((current) => ({ ...current, name, key: keyEdited ? current.key : projectKey(name) }))
            }}
            placeholder="Web application"
            required
            value={form.name}
          />
        </label>
        <label className="text-xs font-medium text-slate-300">
          Project key
          <input
            className={inputClassName}
            maxLength={64}
            onChange={(event) => {
              setKeyEdited(true)
              setForm((current) => ({ ...current, key: event.target.value.toLowerCase() }))
            }}
            pattern="[a-z0-9](?:[a-z0-9_-]{0,62}[a-z0-9])?"
            placeholder="web-application"
            required
            value={form.key}
          />
        </label>
      </div>

      <label className="mt-4 block text-xs font-medium text-slate-300">
        Description <span className="font-normal text-slate-600">(optional)</span>
        <textarea
          className="mt-1.5 min-h-20 w-full resize-y rounded-lg border border-slate-700 bg-slate-950 px-3 py-2 text-sm leading-5 text-slate-200 outline-none transition placeholder:text-slate-700 focus:border-switchonyourcode-500 focus:ring-2 focus:ring-switchonyourcode-500/15"
          maxLength={2000}
          onChange={(event) => setForm((current) => ({ ...current, description: event.target.value }))}
          placeholder="Where this project is used."
          value={form.description}
        />
      </label>

      {error ? (
        <div className="mt-3 rounded-lg border border-red-900/60 bg-red-950/30 px-3 py-2.5 text-xs leading-5 text-red-300">
          {error}
        </div>
      ) : null}

      <div className="mt-4 flex items-center justify-end gap-2">
        <button
          className="rounded-lg px-3 py-2 text-xs font-semibold text-slate-400 transition hover:bg-slate-800 hover:text-white"
          disabled={submitting}
          onClick={onCancel}
          type="button"
        >
          Cancel
        </button>
        <button
          className="rounded-lg bg-switchonyourcode-600 px-3 py-2 text-xs font-semibold text-white transition hover:bg-switchonyourcode-500 disabled:cursor-not-allowed disabled:opacity-60"
          disabled={submitting}
          type="submit"
        >
          {submitting ? 'Creating…' : 'Create project'}
        </button>
      </div>
    </form>
  )
}
