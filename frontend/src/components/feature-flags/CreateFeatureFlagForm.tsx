import { useState, type FormEvent } from 'react'
import type { OrganisationMembership } from '../../auth/types'
import type { CreateFeatureFlagPayload, FeatureFlag, FeatureFlagKind } from '../../featureflag/types'
import { APIError, apiRequest } from '../../lib/api'

interface CreateFeatureFlagFormProps {
  organisation: OrganisationMembership
  projectID: string
  onCancel: () => void
  onCreated: (featureFlag: FeatureFlag) => void
}

function featureFlagKey(value: string): string {
  return value
    .toLowerCase()
    .trim()
    .replace(/[^a-z0-9._-]+/g, '-')
    .replace(/^[._-]+|[._-]+$/g, '')
    .slice(0, 128)
}

function initialDefaultValue(kind: FeatureFlagKind): string {
  switch (kind) {
    case 'boolean':
      return 'false'
    case 'string':
      return ''
    case 'number':
      return '0'
    case 'json':
      return '{}'
  }
}

export function CreateFeatureFlagForm({ organisation, projectID, onCancel, onCreated }: CreateFeatureFlagFormProps) {
  const [name, setName] = useState('')
  const [key, setKey] = useState('')
  const [description, setDescription] = useState('')
  const [kind, setKind] = useState<FeatureFlagKind>('boolean')
  const [defaultValue, setDefaultValue] = useState(initialDefaultValue('boolean'))
  const [keyEdited, setKeyEdited] = useState(false)
  const [error, setError] = useState<string>()
  const [submitting, setSubmitting] = useState(false)

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setSubmitting(true)
    setError(undefined)

    let parsedDefaultValue: unknown
    try {
      parsedDefaultValue = parseDefaultValue(kind, defaultValue)
    } catch (parseError) {
      setError(parseError instanceof Error ? parseError.message : 'Default value is invalid.')
      setSubmitting(false)
      return
    }

    const payload: CreateFeatureFlagPayload = {
      name,
      key,
      description,
      kind,
      default_value: parsedDefaultValue,
    }

    try {
      const featureFlag = await apiRequest<FeatureFlag>(
        `/api/v1/organisations/${encodeURIComponent(organisation.slug)}/projects/${encodeURIComponent(projectID)}/feature-flags`,
        { method: 'POST', body: JSON.stringify(payload) },
      )
      onCreated(featureFlag)
    } catch (requestError) {
      setError(requestError instanceof APIError ? requestError.message : 'Feature flag could not be created.')
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
          Flag name
          <input
            autoFocus
            className={inputClassName}
            maxLength={160}
            onChange={(event) => {
              const nextName = event.target.value
              setName(nextName)
              if (!keyEdited) {
                setKey(featureFlagKey(nextName))
              }
            }}
            placeholder="New checkout flow"
            required
            value={name}
          />
        </label>

        <label className="text-xs font-medium text-slate-300">
          Flag key
          <input
            className={inputClassName}
            maxLength={128}
            onChange={(event) => {
              setKeyEdited(true)
              setKey(event.target.value.toLowerCase())
            }}
            pattern="[a-z0-9](?:[a-z0-9._-]{0,126}[a-z0-9])?"
            placeholder="checkout.new-flow"
            required
            value={key}
          />
        </label>
      </div>

      <div className="mt-4 grid gap-4 md:grid-cols-2">
        <label className="text-xs font-medium text-slate-300">
          Value type
          <select
            className={inputClassName}
            onChange={(event) => {
              const nextKind = event.target.value as FeatureFlagKind
              setKind(nextKind)
              setDefaultValue(initialDefaultValue(nextKind))
            }}
            value={kind}
          >
            <option value="boolean">Boolean</option>
            <option value="string">String</option>
            <option value="number">Number</option>
            <option value="json">JSON</option>
          </select>
        </label>

        <DefaultValueField
          className={inputClassName}
          kind={kind}
          onChange={setDefaultValue}
          value={defaultValue}
        />
      </div>

      <label className="mt-4 block text-xs font-medium text-slate-300">
        Description <span className="font-normal text-slate-600">(optional)</span>
        <textarea
          className="mt-1.5 min-h-20 w-full resize-y rounded-lg border border-slate-700 bg-slate-950 px-3 py-2 text-sm leading-5 text-slate-200 outline-none transition placeholder:text-slate-700 focus:border-switchonyourcode-500 focus:ring-2 focus:ring-switchonyourcode-500/15"
          maxLength={2000}
          onChange={(event) => setDescription(event.target.value)}
          placeholder="Controls the new checkout experience."
          value={description}
        />
      </label>

      {error ? (
        <div className="mt-3 rounded-lg border border-red-900/60 bg-red-950/30 px-3 py-2.5 text-xs leading-5 text-red-300">
          {error}
        </div>
      ) : null}

      <div className="mt-4 flex justify-end gap-2">
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
          {submitting ? 'Creating…' : 'Create feature flag'}
        </button>
      </div>
    </form>
  )
}

function DefaultValueField({
  kind,
  value,
  onChange,
  className,
}: {
  kind: FeatureFlagKind
  value: string
  onChange: (value: string) => void
  className: string
}) {
  if (kind === 'boolean') {
    return (
      <label className="text-xs font-medium text-slate-300">
        Default value
        <select className={className} onChange={(event) => onChange(event.target.value)} value={value}>
          <option value="false">False</option>
          <option value="true">True</option>
        </select>
      </label>
    )
  }

  if (kind === 'json') {
    return (
      <label className="text-xs font-medium text-slate-300">
        Default JSON
        <textarea
          className="mt-1.5 min-h-9 w-full resize-y rounded-lg border border-slate-700 bg-slate-950 px-3 py-2 font-mono text-xs leading-5 text-slate-200 outline-none transition placeholder:text-slate-700 focus:border-switchonyourcode-500 focus:ring-2 focus:ring-switchonyourcode-500/15"
          onChange={(event) => onChange(event.target.value)}
          required
          value={value}
        />
      </label>
    )
  }

  return (
    <label className="text-xs font-medium text-slate-300">
      Default value
      <input
        className={className}
        onChange={(event) => onChange(event.target.value)}
        required={kind === 'number'}
        type={kind === 'number' ? 'number' : 'text'}
        value={value}
      />
    </label>
  )
}

function parseDefaultValue(kind: FeatureFlagKind, value: string): unknown {
  switch (kind) {
    case 'boolean':
      return value === 'true'
    case 'string':
      return value
    case 'number': {
      const parsed = Number(value)
      if (!Number.isFinite(parsed)) {
        throw new Error('Default value must be a valid number.')
      }
      return parsed
    }
    case 'json':
      try {
        return JSON.parse(value) as unknown
      } catch {
        throw new Error('Default value must be valid JSON.')
      }
  }
}
