import { useEffect, useMemo, useState } from 'react'
import type { OrganisationMembership } from '../../auth/types'
import type { Environment } from '../../environment/types'
import type { FeatureFlag } from '../../featureflag/types'
import { APIError, apiRequest } from '../../lib/api'
import type {
  ConditionOperator,
  EnvironmentTargetingState,
  EvaluationResult,
  FlagTargetingState,
  Policy,
  ScheduledChange,
  ScheduledChangeListResponse,
  Segment,
  SegmentListResponse,
  Variant,
} from '../../targeting/types'

interface FeatureTargetingPanelProps {
  organisation: OrganisationMembership
  projectID: string
  featureFlag: FeatureFlag
  environments: Environment[]
  canManage: boolean
  onClose: () => void
}

const operators: Array<{ value: ConditionOperator; label: string }> = [
  { value: 'equals', label: 'equals' },
  { value: 'not_equals', label: 'does not equal' },
  { value: 'in', label: 'is in list' },
  { value: 'not_in', label: 'is not in list' },
  { value: 'contains', label: 'contains' },
  { value: 'not_contains', label: 'does not contain' },
  { value: 'starts_with', label: 'starts with' },
  { value: 'ends_with', label: 'ends with' },
  { value: 'greater_than', label: '>' },
  { value: 'greater_than_or_equal', label: '>=' },
  { value: 'less_than', label: '<' },
  { value: 'less_than_or_equal', label: '<=' },
  { value: 'exists', label: 'exists' },
  { value: 'not_exists', label: 'does not exist' },
  { value: 'matches_regex', label: 'matches regex' },
  { value: 'semver_greater_than', label: 'semantic version >' },
  { value: 'semver_greater_than_or_equal', label: 'semantic version >=' },
  { value: 'semver_less_than', label: 'semantic version <' },
  { value: 'semver_less_than_or_equal', label: 'semantic version <=' },
  { value: 'in_segment', label: 'is in segment' },
  { value: 'not_in_segment', label: 'is not in segment' },
]

const inputClassName =
  'h-9 w-full rounded-lg border border-slate-700 bg-slate-950 px-3 text-xs text-slate-200 outline-none transition placeholder:text-slate-700 focus:border-flagstack-500 focus:ring-2 focus:ring-flagstack-500/15'
const textareaClassName =
  'min-h-28 w-full resize-y rounded-lg border border-slate-700 bg-slate-950 px-3 py-2 font-mono text-xs leading-5 text-slate-300 outline-none transition placeholder:text-slate-700 focus:border-flagstack-500 focus:ring-2 focus:ring-flagstack-500/15'

export function FeatureTargetingPanel({
  organisation,
  projectID,
  featureFlag,
  environments,
  canManage,
  onClose,
}: FeatureTargetingPanelProps) {
  const projectPath = `/api/v1/organisations/${encodeURIComponent(organisation.slug)}/projects/${encodeURIComponent(projectID)}`
  const flagPath = `${projectPath}/feature-flags/${encodeURIComponent(featureFlag.id)}`

  const [targeting, setTargeting] = useState<FlagTargetingState>()
  const [segments, setSegments] = useState<Segment[]>([])
  const [schedules, setSchedules] = useState<ScheduledChange[]>([])
  const [selectedEnvironmentID, setSelectedEnvironmentID] = useState(environments[0]?.id ?? '')
  const [variantsText, setVariantsText] = useState('[]')
  const [policyText, setPolicyText] = useState('{}')
  const [previewText, setPreviewText] = useState(
    JSON.stringify({ targetingKey: 'user-123', country: 'GB', groups: ['beta-testers'], plan: 'pro' }, null, 2),
  )
  const [previewResult, setPreviewResult] = useState<EvaluationResult>()
  const [rolloutPercent, setRolloutPercent] = useState(10)
  const [ruleName, setRuleName] = useState('')
  const [ruleAttribute, setRuleAttribute] = useState('')
  const [ruleOperator, setRuleOperator] = useState<ConditionOperator>('equals')
  const [ruleValue, setRuleValue] = useState('"value"')
  const [ruleVariant, setRuleVariant] = useState('on')
  const [segmentName, setSegmentName] = useState('')
  const [segmentKey, setSegmentKey] = useState('')
  const [segmentConditionsText, setSegmentConditionsText] = useState(
    JSON.stringify([{ attribute: 'groups', operator: 'contains', value: 'beta-testers' }], null, 2),
  )
  const [segmentMatch, setSegmentMatch] = useState<'all' | 'any'>('all')
  const [scheduleAt, setScheduleAt] = useState('')
  const [scheduleAction, setScheduleAction] = useState<'enable' | 'disable' | 'policy'>('enable')
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState<string>()
  const [error, setError] = useState<string>()

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError(undefined)

    void Promise.all([
      apiRequest<FlagTargetingState>(`${flagPath}/targeting`),
      apiRequest<SegmentListResponse>(`${projectPath}/segments`),
      apiRequest<ScheduledChangeListResponse>(`${projectPath}/scheduled-flag-changes`),
    ])
      .then(([targetingResponse, segmentResponse, scheduleResponse]) => {
        if (cancelled) {
          return
        }
        setTargeting(targetingResponse)
        setVariantsText(JSON.stringify(targetingResponse.variants ?? [], null, 2))
        setSegments(segmentResponse.segments)
        setSchedules(scheduleResponse.scheduled_changes)
      })
      .catch((requestError) => {
        if (!cancelled) {
          setError(requestError instanceof APIError ? requestError.message : 'Targeting configuration could not be loaded.')
        }
      })
      .finally(() => {
        if (!cancelled) {
          setLoading(false)
        }
      })

    return () => {
      cancelled = true
    }
  }, [flagPath, projectPath])

  const selectedEnvironment = environments.find((environment) => environment.id === selectedEnvironmentID)
  const selectedState = targeting?.environments.find((state) => state.environment_id === selectedEnvironmentID)
  const availableVariants = useMemo(() => variantKeys(featureFlag.kind, targeting?.variants ?? []), [featureFlag.kind, targeting?.variants])

  useEffect(() => {
    const policy = selectedState?.policy ?? {}
    setPolicyText(JSON.stringify(policy, null, 2))
    const onAllocation = policy.fallthrough?.rollout?.find((allocation) => allocation.variant === 'on')
    if (onAllocation) {
      setRolloutPercent(onAllocation.weight / 1000)
    }
  }, [selectedEnvironmentID, selectedState?.revision])

  useEffect(() => {
    if (!availableVariants.includes(ruleVariant)) {
      setRuleVariant(availableVariants[0] ?? 'default')
    }
  }, [availableVariants, ruleVariant])

  async function saveVariants() {
    setSaving('variants')
    setError(undefined)
    try {
      const variants = parseJSON<Variant[]>(variantsText, 'Variants must be valid JSON.')
      if (!Array.isArray(variants)) {
        throw new Error('Variants must be a JSON array.')
      }
      const updated = await apiRequest<FlagTargetingState>(`${flagPath}/variants`, {
        method: 'PUT',
        body: JSON.stringify({ variants }),
      })
      setTargeting(updated)
      setVariantsText(JSON.stringify(updated.variants ?? [], null, 2))
    } catch (requestError) {
      setError(errorMessage(requestError, 'Variants could not be saved.'))
    } finally {
      setSaving(undefined)
    }
  }

  async function savePolicy(policyOverride?: Policy) {
    if (!selectedEnvironmentID) {
      setError('Create an environment before configuring targeting.')
      return
    }
    setSaving('policy')
    setError(undefined)
    try {
      const policy = policyOverride ?? parseJSON<Policy>(policyText, 'Policy must be valid JSON.')
      const updated = await apiRequest<EnvironmentTargetingState>(
        `${projectPath}/environments/${encodeURIComponent(selectedEnvironmentID)}/feature-flags/${encodeURIComponent(featureFlag.id)}/policy`,
        { method: 'PUT', body: JSON.stringify({ policy }) },
      )
      setTargeting((current) => current ? {
        ...current,
        environments: replaceEnvironmentState(current.environments, updated),
      } : current)
      setPolicyText(JSON.stringify(updated.policy ?? {}, null, 2))
    } catch (requestError) {
      setError(errorMessage(requestError, 'Targeting policy could not be saved.'))
    } finally {
      setSaving(undefined)
    }
  }

  function applyBooleanRollout() {
    const percent = Math.min(100, Math.max(0, rolloutPercent))
    const onWeight = Math.round(percent * 1000)
    const policy = parsePolicyDraft(policyText)
    const next: Policy = {
      ...policy,
      fallthrough:
        onWeight === 100000
          ? { variant: 'on' }
          : onWeight === 0
            ? { variant: 'off' }
            : {
                rollout: [
                  { variant: 'on', weight: onWeight },
                  { variant: 'off', weight: 100000 - onWeight },
                ],
              },
    }
    setPolicyText(JSON.stringify(next, null, 2))
  }

  function addRule() {
    setError(undefined)
    try {
      const segmentOperator = ruleOperator === 'in_segment' || ruleOperator === 'not_in_segment'
      const noValueOperator = ruleOperator === 'exists' || ruleOperator === 'not_exists'
      if (!segmentOperator && !ruleAttribute.trim()) {
        throw new Error('Rule attribute is required.')
      }
      const condition = {
        ...(segmentOperator ? {} : { attribute: ruleAttribute.trim() }),
        operator: ruleOperator,
        ...(noValueOperator ? {} : { value: parseJSON<unknown>(ruleValue, 'Rule value must be valid JSON.') }),
      }
      const policy = parsePolicyDraft(policyText)
      const next: Policy = {
        ...policy,
        rules: [
          ...(policy.rules ?? []),
          {
            id: `rule-${Date.now()}`,
            name: ruleName.trim() || undefined,
            match: 'all',
            conditions: [condition],
            outcome: { variant: ruleVariant },
          },
        ],
      }
      setPolicyText(JSON.stringify(next, null, 2))
      setRuleName('')
    } catch (ruleError) {
      setError(errorMessage(ruleError, 'Rule could not be added.'))
    }
  }

  async function runPreview() {
    if (!selectedEnvironmentID) {
      setError('Select an environment to preview evaluation.')
      return
    }
    setSaving('preview')
    setError(undefined)
    try {
      const context = parseJSON<Record<string, unknown>>(previewText, 'Preview context must be valid JSON.')
      const result = await apiRequest<EvaluationResult>(
        `${projectPath}/environments/${encodeURIComponent(selectedEnvironmentID)}/feature-flags/${encodeURIComponent(featureFlag.id)}/preview`,
        { method: 'POST', body: JSON.stringify({ context }) },
      )
      setPreviewResult(result)
    } catch (requestError) {
      setError(errorMessage(requestError, 'Evaluation preview failed.'))
    } finally {
      setSaving(undefined)
    }
  }

  async function createSegment() {
    setSaving('segment')
    setError(undefined)
    try {
      const conditions = parseJSON<Segment['conditions']>(segmentConditionsText, 'Segment conditions must be valid JSON.')
      if (!Array.isArray(conditions)) {
        throw new Error('Segment conditions must be a JSON array.')
      }
      const segment = await apiRequest<Segment>(`${projectPath}/segments`, {
        method: 'POST',
        body: JSON.stringify({
          name: segmentName,
          key: segmentKey,
          description: '',
          match: segmentMatch,
          conditions,
        }),
      })
      setSegments((current) => [...current, segment])
      setSegmentName('')
      setSegmentKey('')
    } catch (requestError) {
      setError(errorMessage(requestError, 'Segment could not be created.'))
    } finally {
      setSaving(undefined)
    }
  }

  async function createSchedule() {
    if (!selectedEnvironmentID || !scheduleAt) {
      setError('Choose an environment and execution time for the schedule.')
      return
    }
    setSaving('schedule')
    setError(undefined)
    try {
      const patch =
        scheduleAction === 'enable'
          ? { enabled: true }
          : scheduleAction === 'disable'
            ? { enabled: false }
            : { policy: parseJSON<Policy>(policyText, 'Policy must be valid JSON.') }
      const change = await apiRequest<ScheduledChange>(`${projectPath}/scheduled-flag-changes`, {
        method: 'POST',
        body: JSON.stringify({
          environment_id: selectedEnvironmentID,
          feature_flag_id: featureFlag.id,
          execute_at: new Date(scheduleAt).toISOString(),
          patch,
        }),
      })
      setSchedules((current) => [...current, change].sort((left, right) => left.execute_at.localeCompare(right.execute_at)))
      setScheduleAt('')
    } catch (requestError) {
      setError(errorMessage(requestError, 'Scheduled change could not be created.'))
    } finally {
      setSaving(undefined)
    }
  }

  async function cancelSchedule(scheduleID: string) {
    setSaving(`schedule:${scheduleID}`)
    setError(undefined)
    try {
      const updated = await apiRequest<ScheduledChange>(`${projectPath}/scheduled-flag-changes/${encodeURIComponent(scheduleID)}/cancel`, {
        method: 'POST',
      })
      setSchedules((current) => current.map((change) => (change.id === updated.id ? updated : change)))
    } catch (requestError) {
      setError(errorMessage(requestError, 'Scheduled change could not be cancelled.'))
    } finally {
      setSaving(undefined)
    }
  }

  if (loading) {
    return <div className="border-t border-slate-800 bg-slate-950/30 px-5 py-6 text-xs text-slate-500">Loading targeting configuration…</div>
  }

  return (
    <div className="border-t border-slate-800 bg-slate-950/30 px-5 py-5">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <p className="text-[10px] font-bold uppercase tracking-[0.18em] text-flagstack-400">Targeting</p>
          <h3 className="mt-1 text-sm font-semibold text-slate-200">{featureFlag.name}</h3>
          <p className="mt-1 max-w-2xl text-xs leading-5 text-slate-500">
            Configure variants, ordered rules, deterministic rollouts, reusable segments, previews and scheduled changes before connecting an SDK.
          </p>
        </div>
        <button className="rounded-lg px-3 py-2 text-xs font-semibold text-slate-400 transition hover:bg-slate-800 hover:text-white" onClick={onClose} type="button">
          Close targeting
        </button>
      </div>

      {error ? <div className="mt-4 rounded-lg border border-red-900/60 bg-red-950/30 px-3 py-2.5 text-xs leading-5 text-red-300">{error}</div> : null}

      <div className="mt-5 grid gap-4 xl:grid-cols-2">
        <Section title="Variants" description="Named return values used for multivariate delivery. Boolean flags already include on/off/default.">
          <textarea className={textareaClassName} disabled={!canManage} onChange={(event) => setVariantsText(event.target.value)} spellCheck={false} value={variantsText} />
          <div className="mt-3 flex justify-end">
            <ActionButton disabled={!canManage || saving === 'variants'} onClick={() => void saveVariants()}>
              {saving === 'variants' ? 'Saving…' : 'Save variants'}
            </ActionButton>
          </div>
        </Section>

        <Section title="Environment" description="Rules and rollout are evaluated independently for each environment.">
          <select className={inputClassName} onChange={(event) => setSelectedEnvironmentID(event.target.value)} value={selectedEnvironmentID}>
            {environments.length === 0 ? <option value="">No environments</option> : null}
            {environments.map((environment) => <option key={environment.id} value={environment.id}>{environment.name}</option>)}
          </select>
          <div className="mt-3 flex items-center justify-between rounded-lg border border-slate-800 bg-slate-950/60 px-3 py-2.5">
            <span className="text-xs text-slate-500">Current revision</span>
            <strong className="text-xs text-slate-300">{selectedState?.revision ?? 'sparse / none'}</strong>
          </div>
        </Section>

        {featureFlag.kind === 'boolean' ? (
          <Section title="Quick percentage rollout" description="Stable subjects stay in the same cohort when you increase the percentage.">
            <div className="flex items-center gap-3">
              <input className="w-full accent-violet-500" max="100" min="0" onChange={(event) => setRolloutPercent(Number(event.target.value))} step="0.1" type="range" value={rolloutPercent} />
              <input className={`${inputClassName} w-24 text-right`} max="100" min="0" onChange={(event) => setRolloutPercent(Number(event.target.value))} step="0.1" type="number" value={rolloutPercent} />
              <span className="text-xs text-slate-500">%</span>
            </div>
            <div className="mt-3 flex justify-end gap-2">
              <button className="rounded-lg px-3 py-2 text-xs font-semibold text-slate-400 hover:bg-slate-800" onClick={applyBooleanRollout} type="button">Apply to draft</button>
              <ActionButton disabled={!canManage || saving === 'policy'} onClick={() => { applyBooleanRollout(); const percent = Math.min(100, Math.max(0, rolloutPercent)); const onWeight = Math.round(percent * 1000); const current = parsePolicyDraft(policyText); void savePolicy({ ...current, fallthrough: onWeight === 100000 ? { variant: 'on' } : onWeight === 0 ? { variant: 'off' } : { rollout: [{ variant: 'on', weight: onWeight }, { variant: 'off', weight: 100000 - onWeight }] } }) }}>
                Save rollout
              </ActionButton>
            </div>
          </Section>
        ) : null}

        <Section title="Add targeting rule" description="Target application-supplied user, group, tenant, device or custom context attributes.">
          <div className="grid gap-3 sm:grid-cols-2">
            <input className={inputClassName} onChange={(event) => setRuleName(event.target.value)} placeholder="Rule name (optional)" value={ruleName} />
            <input className={inputClassName} disabled={ruleOperator === 'in_segment' || ruleOperator === 'not_in_segment'} onChange={(event) => setRuleAttribute(event.target.value)} placeholder="Attribute, e.g. plan or groups" value={ruleAttribute} />
            <select className={inputClassName} onChange={(event) => setRuleOperator(event.target.value as ConditionOperator)} value={ruleOperator}>
              {operators.map((operator) => <option key={operator.value} value={operator.value}>{operator.label}</option>)}
            </select>
            <input className={inputClassName} disabled={ruleOperator === 'exists' || ruleOperator === 'not_exists'} onChange={(event) => setRuleValue(event.target.value)} placeholder='JSON value, e.g. "enterprise"' value={ruleValue} />
            <select className={inputClassName} onChange={(event) => setRuleVariant(event.target.value)} value={ruleVariant}>
              {availableVariants.map((variant) => <option key={variant} value={variant}>{variant}</option>)}
            </select>
          </div>
          <p className="mt-2 text-[11px] leading-4 text-slate-600">
            For segment operators, enter the segment key as a JSON string in the value field. Add multiple conditions or reorder rules in the advanced JSON editor below.
          </p>
          <div className="mt-3 flex justify-end"><button className="rounded-lg border border-slate-700 px-3 py-2 text-xs font-semibold text-slate-300 hover:bg-slate-800" onClick={addRule} type="button">Add rule to draft</button></div>
        </Section>

        <Section title="Advanced policy" description="Ordered rules use first-match-wins semantics; fallthrough can return a variant or deterministic allocation.">
          <textarea className={textareaClassName} disabled={!canManage} onChange={(event) => setPolicyText(event.target.value)} spellCheck={false} value={policyText} />
          <div className="mt-3 flex justify-end"><ActionButton disabled={!canManage || saving === 'policy'} onClick={() => void savePolicy()}>{saving === 'policy' ? 'Saving…' : 'Save policy'}</ActionButton></div>
        </Section>

        <Section title="Evaluation preview" description="Run the exact reference evaluator against an SDK-style context before deployment.">
          <textarea className={textareaClassName} onChange={(event) => setPreviewText(event.target.value)} spellCheck={false} value={previewText} />
          <div className="mt-3 flex justify-end"><ActionButton disabled={!selectedEnvironmentID || saving === 'preview'} onClick={() => void runPreview()}>{saving === 'preview' ? 'Evaluating…' : 'Preview evaluation'}</ActionButton></div>
          {previewResult ? (
            <pre className="mt-3 overflow-x-auto rounded-lg border border-slate-800 bg-black/30 p-3 text-[11px] leading-5 text-slate-400">{JSON.stringify(previewResult, null, 2)}</pre>
          ) : null}
        </Section>

        <Section title="Reusable segments" description="Create reusable groups from application-owned context attributes; no user database sync is required.">
          {segments.length > 0 ? (
            <div className="mb-3 flex flex-wrap gap-2">
              {segments.map((segment) => <span className="rounded-md border border-slate-800 bg-slate-950 px-2 py-1 text-[10px] text-slate-400" key={segment.id}>{segment.name} · {segment.key}</span>)}
            </div>
          ) : <p className="mb-3 text-xs text-slate-600">No segments yet.</p>}
          <div className="grid gap-3 sm:grid-cols-2">
            <input className={inputClassName} disabled={!canManage} onChange={(event) => { setSegmentName(event.target.value); if (!segmentKey) setSegmentKey(toKey(event.target.value)) }} placeholder="Segment name" value={segmentName} />
            <input className={inputClassName} disabled={!canManage} onChange={(event) => setSegmentKey(event.target.value.toLowerCase())} placeholder="segment-key" value={segmentKey} />
            <select className={inputClassName} disabled={!canManage} onChange={(event) => setSegmentMatch(event.target.value as 'all' | 'any')} value={segmentMatch}><option value="all">Match all conditions</option><option value="any">Match any condition</option></select>
          </div>
          <textarea className={`${textareaClassName} mt-3`} disabled={!canManage} onChange={(event) => setSegmentConditionsText(event.target.value)} spellCheck={false} value={segmentConditionsText} />
          <div className="mt-3 flex justify-end"><ActionButton disabled={!canManage || !segmentName || !segmentKey || saving === 'segment'} onClick={() => void createSegment()}>{saving === 'segment' ? 'Creating…' : 'Create segment'}</ActionButton></div>
        </Section>

        <Section title="Schedule change" description="Schedule enable/disable changes or the current policy draft. Multiple policy schedules can form a progressive rollout.">
          <div className="grid gap-3 sm:grid-cols-2">
            <select className={inputClassName} onChange={(event) => setScheduleAction(event.target.value as 'enable' | 'disable' | 'policy')} value={scheduleAction}>
              <option value="enable">Turn flag on</option>
              <option value="disable">Turn flag off</option>
              <option value="policy">Apply current policy draft</option>
            </select>
            <input className={inputClassName} min={localDateTimeMinimum()} onChange={(event) => setScheduleAt(event.target.value)} type="datetime-local" value={scheduleAt} />
          </div>
          <div className="mt-3 flex justify-end"><ActionButton disabled={!canManage || !selectedEnvironmentID || !scheduleAt || saving === 'schedule'} onClick={() => void createSchedule()}>{saving === 'schedule' ? 'Scheduling…' : 'Schedule change'}</ActionButton></div>

          <div className="mt-4 divide-y divide-slate-800 border-t border-slate-800">
            {schedules.filter((schedule) => schedule.feature_flag_id === featureFlag.id).length === 0 ? <p className="py-3 text-xs text-slate-600">No scheduled changes for this flag.</p> : null}
            {schedules.filter((schedule) => schedule.feature_flag_id === featureFlag.id).map((schedule) => {
              const environment = environments.find((candidate) => candidate.id === schedule.environment_id)
              return (
                <div className="flex items-center gap-3 py-3" key={schedule.id}>
                  <div className="min-w-0 flex-1">
                    <p className="text-xs font-medium text-slate-300">{environment?.name ?? 'Unknown environment'} · {scheduleSummary(schedule)}</p>
                    <p className="mt-1 text-[10px] text-slate-600">{new Date(schedule.execute_at).toLocaleString()} · {schedule.status}</p>
                  </div>
                  {schedule.status === 'pending' && canManage ? <button className="text-[11px] font-semibold text-red-400 hover:text-red-300" disabled={saving === `schedule:${schedule.id}`} onClick={() => void cancelSchedule(schedule.id)} type="button">Cancel</button> : null}
                </div>
              )
            })}
          </div>
        </Section>
      </div>
    </div>
  )
}

function Section({ title, description, children }: { title: string; description: string; children: React.ReactNode }) {
  return (
    <section className="rounded-xl border border-slate-800 bg-slate-900/45 p-4">
      <h4 className="text-xs font-semibold text-slate-200">{title}</h4>
      <p className="mt-1 mb-3 text-[11px] leading-4 text-slate-600">{description}</p>
      {children}
    </section>
  )
}

function ActionButton({ disabled, onClick, children }: { disabled?: boolean; onClick: () => void; children: React.ReactNode }) {
  return <button className="rounded-lg bg-flagstack-600 px-3 py-2 text-xs font-semibold text-white transition hover:bg-flagstack-500 disabled:cursor-not-allowed disabled:opacity-50" disabled={disabled} onClick={onClick} type="button">{children}</button>
}

function parseJSON<T>(value: string, message: string): T {
  try {
    return JSON.parse(value) as T
  } catch {
    throw new Error(message)
  }
}

function parsePolicyDraft(value: string): Policy {
  return parseJSON<Policy>(value, 'Policy must be valid JSON.')
}

function errorMessage(error: unknown, fallback: string): string {
  return error instanceof APIError || error instanceof Error ? error.message : fallback
}

function replaceEnvironmentState(states: EnvironmentTargetingState[], updated: EnvironmentTargetingState): EnvironmentTargetingState[] {
  const found = states.some((state) => state.environment_id === updated.environment_id)
  return found ? states.map((state) => state.environment_id === updated.environment_id ? updated : state) : [...states, updated]
}

function variantKeys(kind: FeatureFlag['kind'], variants: Variant[]): string[] {
  const reserved = kind === 'boolean' ? ['on', 'off', 'default'] : ['default']
  return [...reserved, ...variants.map((variant) => variant.key)]
}

function toKey(value: string): string {
  return value.toLowerCase().trim().replace(/[^a-z0-9._-]+/g, '-').replace(/^[._-]+|[._-]+$/g, '').slice(0, 128)
}

function localDateTimeMinimum(): string {
  const date = new Date(Date.now() + 60_000)
  const offset = date.getTimezoneOffset()
  return new Date(date.getTime() - offset * 60_000).toISOString().slice(0, 16)
}

function scheduleSummary(schedule: ScheduledChange): string {
  if (schedule.patch.enabled === true) return 'Turn on'
  if (schedule.patch.enabled === false) return 'Turn off'
  if (schedule.patch.policy) return 'Apply targeting policy'
  return 'Configuration change'
}
