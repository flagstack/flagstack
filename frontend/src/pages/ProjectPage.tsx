import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router'
import type { OrganisationMembership } from '../auth/types'
import { CreateEnvironmentForm } from '../components/environments/CreateEnvironmentForm'
import { CreateFeatureFlagForm } from '../components/feature-flags/CreateFeatureFlagForm'
import { FeatureTargetingPanel } from '../components/feature-flags/FeatureTargetingPanel'
import { Icon } from '../components/icons'
import { SDKCredentialsPanel } from '../components/sdk/SDKCredentialsPanel'
import { Card, CardContent, CardHeader, CardTitle } from '../components/ui/Card'
import { PageHeader } from '../components/ui/PageHeader'
import type { Environment, EnvironmentListResponse } from '../environment/types'
import type { FeatureFlag, FeatureFlagListResponse } from '../featureflag/types'
import type { FlagConfig, FlagConfigListResponse } from '../flagconfig/types'
import { APIError, apiRequest } from '../lib/api'
import type { Project, ProjectListResponse } from '../project/types'
import type { ClientVisibilityResponse, SDKCredential, SDKCredentialListResponse } from '../sdkconfig/types'

interface ProjectPageProps {
  organisation: OrganisationMembership
}

export function ProjectPage({ organisation }: ProjectPageProps) {
  const { projectKey } = useParams<{ projectKey: string }>()
  const [project, setProject] = useState<Project>()
  const [environments, setEnvironments] = useState<Environment[]>([])
  const [featureFlags, setFeatureFlags] = useState<FeatureFlag[]>([])
  const [flagConfigs, setFlagConfigs] = useState<FlagConfig[]>([])
  const [sdkCredentials, setSDKCredentials] = useState<SDKCredential[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string>()
  const [actionError, setActionError] = useState<string>()
  const [creatingEnvironment, setCreatingEnvironment] = useState(false)
  const [creatingFeatureFlag, setCreatingFeatureFlag] = useState(false)
  const [updatingConfig, setUpdatingConfig] = useState<string>()
  const [updatingClientVisibility, setUpdatingClientVisibility] = useState<string>()
  const [targetingFlagID, setTargetingFlagID] = useState<string>()
  const canManageSDKCredentials = organisation.role === 'owner' || organisation.role === 'admin'

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError(undefined)

    void apiRequest<ProjectListResponse>(
      `/api/v1/organisations/${encodeURIComponent(organisation.slug)}/projects`,
    )
      .then(async (response) => {
        const selected = response.projects.find((candidate) => candidate.key === projectKey)
        if (!selected) {
          throw new APIError(404, 'project_not_found', 'Project was not found.')
        }

        const projectPath = `/api/v1/organisations/${encodeURIComponent(organisation.slug)}/projects/${encodeURIComponent(selected.id)}`
        const credentialRequest = canManageSDKCredentials
          ? apiRequest<SDKCredentialListResponse>(`${projectPath}/sdk-credentials`)
          : Promise.resolve<SDKCredentialListResponse>({ credentials: [] })
        const [environmentResponse, featureFlagResponse, flagConfigResponse, credentialResponse] = await Promise.all([
          apiRequest<EnvironmentListResponse>(`${projectPath}/environments`),
          apiRequest<FeatureFlagListResponse>(`${projectPath}/feature-flags`),
          apiRequest<FlagConfigListResponse>(`${projectPath}/flag-configs`),
          credentialRequest,
        ])

        if (!cancelled) {
          setProject(selected)
          setEnvironments(environmentResponse.environments)
          setFeatureFlags(featureFlagResponse.feature_flags)
          setFlagConfigs(flagConfigResponse.configs)
          setSDKCredentials(credentialResponse.credentials)
        }
      })
      .catch((requestError) => {
        if (!cancelled) {
          setError(requestError instanceof APIError ? requestError.message : 'Project could not be loaded.')
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
  }, [canManageSDKCredentials, organisation.slug, projectKey])

  if (loading) {
    return <ProjectState message="Loading project…" />
  }

  if (error || !project) {
    return <ProjectState message={error ?? 'Project was not found.'} />
  }

  const canManage = organisation.role !== 'viewer'

  async function setFlagEnabled(environmentID: string, featureFlagID: string, enabled: boolean) {
    if (!project) {
      return
    }

    const configKey = `${environmentID}:${featureFlagID}`
    setUpdatingConfig(configKey)
    setActionError(undefined)

    try {
      const config = await apiRequest<FlagConfig>(
        `/api/v1/organisations/${encodeURIComponent(organisation.slug)}/projects/${encodeURIComponent(project.id)}/environments/${encodeURIComponent(environmentID)}/feature-flags/${encodeURIComponent(featureFlagID)}`,
        { method: 'PUT', body: JSON.stringify({ enabled }) },
      )

      setFlagConfigs((current) => {
        const existingIndex = current.findIndex(
          (candidate) => candidate.environment_id === environmentID && candidate.feature_flag_id === featureFlagID,
        )
        if (existingIndex === -1) {
          return [...current, config]
        }
        return current.map((candidate, index) => (index === existingIndex ? config : candidate))
      })
    } catch (requestError) {
      setActionError(requestError instanceof APIError ? requestError.message : 'Feature flag state could not be updated.')
    } finally {
      setUpdatingConfig(undefined)
    }
  }

  async function setFlagClientVisible(featureFlagID: string, clientVisible: boolean) {
    if (!project || !canManageSDKCredentials) {
      return
    }
    setUpdatingClientVisibility(featureFlagID)
    setActionError(undefined)
    try {
      const response = await apiRequest<ClientVisibilityResponse>(
        `/api/v1/organisations/${encodeURIComponent(organisation.slug)}/projects/${encodeURIComponent(project.id)}/feature-flags/${encodeURIComponent(featureFlagID)}/client-visibility`,
        { method: 'PUT', body: JSON.stringify({ client_visible: clientVisible }) },
      )
      setFeatureFlags((current) => current.map((flag) => (
        flag.id === featureFlagID ? { ...flag, client_visible: response.client_visible } : flag
      )))
    } catch (requestError) {
      setActionError(requestError instanceof APIError ? requestError.message : 'Client visibility could not be updated.')
    } finally {
      setUpdatingClientVisibility(undefined)
    }
  }

  const activeSDKCredentials = sdkCredentials.filter((credential) => !credential.revoked_at)

  return (
    <div className="mx-auto w-full max-w-[100rem] px-4 py-6 lg:px-6 lg:py-8">
      <div className="mb-4">
        <Link className="inline-flex items-center gap-1.5 text-xs font-medium text-slate-500 transition hover:text-slate-300" to="/">
          <Icon name="chevron-left" size={14} />
          Dashboard
        </Link>
      </div>

      <PageHeader
        actions={
          <code className="rounded-lg border border-slate-800 bg-slate-900/60 px-3 py-2 text-xs text-slate-400">
            {project.key}
          </code>
        }
        description={project.description || 'Manage environments and feature flags for this project.'}
        eyebrow={organisation.name}
        title={project.name}
      />

      <div className="mt-6 grid gap-4 xl:grid-cols-12">
        <Card className="xl:col-span-7">
          <CardHeader className="flex items-start justify-between gap-4 border-b border-slate-800">
            <div>
              <CardTitle>Environments</CardTitle>
              <p className="mt-1 text-xs text-slate-500">Independent delivery and configuration boundaries for this project.</p>
            </div>
            {canManage ? (
              <button
                className="rounded-lg bg-flagstack-600 px-3 py-1.5 text-xs font-semibold text-white transition hover:bg-flagstack-500"
                onClick={() => setCreatingEnvironment((value) => !value)}
                type="button"
              >
                {creatingEnvironment ? 'Close' : 'Create environment'}
              </button>
            ) : null}
          </CardHeader>

          {creatingEnvironment ? (
            <CreateEnvironmentForm
              onCancel={() => setCreatingEnvironment(false)}
              onCreated={(environment) => {
                setEnvironments((current) => [...current, environment])
                setProject((current) => current ? { ...current, environment_count: current.environment_count + 1 } : current)
                setCreatingEnvironment(false)
              }}
              organisation={organisation}
              projectID={project.id}
            />
          ) : null}

          <CardContent className="min-h-52">
            {environments.length === 0 ? (
              <div className="flex min-h-52 flex-col items-center justify-center px-5 py-8 text-center">
                <span className="flex h-10 w-10 items-center justify-center rounded-xl bg-blue-500/10 text-blue-400">
                  <Icon name="environment" size={20} />
                </span>
                <strong className="mt-3 text-sm font-medium text-slate-300">No environments yet</strong>
                <p className="mt-1 max-w-md text-xs leading-5 text-slate-600">
                  Add a delivery boundary such as development, staging or production.
                </p>
              </div>
            ) : (
              <div className="divide-y divide-slate-800">
                {environments.map((environment) => (
                  <div className="flex items-center gap-4 px-5 py-4" key={environment.id}>
                    <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-blue-500/10 text-blue-400">
                      <Icon name="environment" size={17} />
                    </span>
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-2">
                        <strong className="truncate text-sm font-medium text-slate-200">{environment.name}</strong>
                        <code className="rounded bg-slate-950 px-1.5 py-0.5 text-[10px] text-slate-500">{environment.key}</code>
                      </div>
                      <p className="mt-1 truncate text-xs text-slate-600">{environment.description || 'No description'}</p>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </CardContent>
        </Card>

        <Card className="xl:col-span-5">
          <CardHeader className="border-b border-slate-800">
            <CardTitle>Project overview</CardTitle>
            <p className="mt-1 text-xs text-slate-500">Current configuration footprint.</p>
          </CardHeader>
          <CardContent className="divide-y divide-slate-800">
            <OverviewRow icon="environment" label="Environments" value={String(environments.length)} />
            <OverviewRow icon="flag" label="Feature flags" value={String(featureFlags.length)} />
            <OverviewRow icon="key" label="SDK keys" value={canManageSDKCredentials ? String(activeSDKCredentials.length) : '—'} />
          </CardContent>
        </Card>
      </div>

      {canManageSDKCredentials ? (
        <SDKCredentialsPanel
          canManage
          credentials={sdkCredentials}
          environments={environments}
          onCreated={(credential) => setSDKCredentials((current) => [...current, credential])}
          onRevoked={(credential) => setSDKCredentials((current) => current.map((candidate) => (
            candidate.id === credential.id ? credential : candidate
          )))}
          organisation={organisation}
          projectID={project.id}
        />
      ) : null}

      <Card className="mt-4">
        <CardHeader className="flex items-start justify-between gap-4 border-b border-slate-800">
          <div>
            <CardTitle>Feature flags</CardTitle>
            <p className="mt-1 text-xs text-slate-500">Simple environment toggles plus variants, targeting rules, percentage rollouts and schedules.</p>
          </div>
          {canManage ? (
            <button
              className="rounded-lg bg-flagstack-600 px-3 py-1.5 text-xs font-semibold text-white transition hover:bg-flagstack-500"
              onClick={() => setCreatingFeatureFlag((value) => !value)}
              type="button"
            >
              {creatingFeatureFlag ? 'Close' : 'Create feature flag'}
            </button>
          ) : null}
        </CardHeader>

        {creatingFeatureFlag ? (
          <CreateFeatureFlagForm
            onCancel={() => setCreatingFeatureFlag(false)}
            onCreated={(featureFlag) => {
              setFeatureFlags((current) => [...current, featureFlag])
              setProject((current) => current ? { ...current, feature_flag_count: current.feature_flag_count + 1 } : current)
              setCreatingFeatureFlag(false)
            }}
            organisation={organisation}
            projectID={project.id}
          />
        ) : null}

        <CardContent className="min-h-52">
          {actionError ? (
            <div className="m-5 rounded-lg border border-red-900/60 bg-red-950/30 px-3 py-2.5 text-xs leading-5 text-red-300">
              {actionError}
            </div>
          ) : null}

          {featureFlags.length === 0 ? (
            <div className="flex min-h-52 flex-col items-center justify-center px-5 py-8 text-center">
              <span className="flex h-10 w-10 items-center justify-center rounded-xl bg-flagstack-500/10 text-flagstack-400">
                <Icon name="flag" size={20} />
              </span>
              <strong className="mt-3 text-sm font-medium text-slate-300">No feature flags yet</strong>
              <p className="mt-1 max-w-md text-xs leading-5 text-slate-600">
                Create a flag once, then control delivery independently in every environment.
              </p>
            </div>
          ) : (
            <div className="divide-y divide-slate-800">
              {featureFlags.map((featureFlag) => (
                <div key={featureFlag.id}>
                  <div className="px-5 py-4">
                    <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
                      <div className="flex min-w-0 gap-3">
                        <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-flagstack-500/10 text-flagstack-400">
                          <Icon name="flag" size={17} />
                        </span>
                        <div className="min-w-0">
                          <div className="flex flex-wrap items-center gap-2">
                            <strong className="text-sm font-medium text-slate-200">{featureFlag.name}</strong>
                            <code className="rounded bg-slate-950 px-1.5 py-0.5 text-[10px] text-slate-500">{featureFlag.key}</code>
                            <span className="rounded border border-slate-800 bg-slate-950/70 px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-slate-500">
                              {featureFlag.kind}
                            </span>
                            <button
                              className={`rounded border px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wide transition disabled:cursor-default ${
                                featureFlag.client_visible
                                  ? 'border-sky-800/70 bg-sky-950/30 text-sky-300'
                                  : 'border-slate-800 bg-slate-950/70 text-slate-600'
                              }`}
                              disabled={!canManageSDKCredentials || updatingClientVisibility === featureFlag.id}
                              onClick={() => void setFlagClientVisible(featureFlag.id, !featureFlag.client_visible)}
                              title={canManageSDKCredentials ? 'Control whether browser/client SDKs receive this flag.' : 'Only owners and admins can change client SDK exposure.'}
                              type="button"
                            >
                              {updatingClientVisibility === featureFlag.id ? 'Updating…' : featureFlag.client_visible ? 'Client visible' : 'Server only'}
                            </button>
                          </div>
                          <p className="mt-1 text-xs text-slate-600">{featureFlag.description || 'No description'}</p>
                          <p className="mt-2 text-[11px] text-slate-500">
                            Default: <code className="text-slate-400">{formatDefaultValue(featureFlag.default_value)}</code>
                          </p>
                        </div>
                      </div>

                      <div className="min-w-0 lg:max-w-[60%]">
                        <div className="flex flex-wrap items-center justify-end gap-2">
                          {environments.length === 0 ? (
                            <span className="text-xs text-slate-600">Create an environment to control delivery.</span>
                          ) : environments.map((environment) => {
                            const config = flagConfigs.find(
                              (candidate) => candidate.environment_id === environment.id && candidate.feature_flag_id === featureFlag.id,
                            )
                            const enabled = config?.enabled ?? false
                            const configKey = `${environment.id}:${featureFlag.id}`
                            const updating = updatingConfig === configKey

                            return (
                              <button
                                aria-pressed={enabled}
                                className={`inline-flex items-center gap-2 rounded-lg border px-2.5 py-1.5 text-xs font-medium transition disabled:cursor-not-allowed disabled:opacity-60 ${
                                  enabled
                                    ? 'border-emerald-800/80 bg-emerald-950/40 text-emerald-300 hover:bg-emerald-950/60'
                                    : 'border-slate-800 bg-slate-950/60 text-slate-500 hover:border-slate-700 hover:text-slate-300'
                                }`}
                                disabled={!canManage || updating}
                                key={environment.id}
                                onClick={() => void setFlagEnabled(environment.id, featureFlag.id, !enabled)}
                                title={canManage ? `Turn ${featureFlag.name} ${enabled ? 'off' : 'on'} in ${environment.name}` : 'Viewer access is read-only'}
                                type="button"
                              >
                                <span className={`h-1.5 w-1.5 rounded-full ${enabled ? 'bg-emerald-400' : 'bg-slate-600'}`} />
                                <span className="max-w-36 truncate">{environment.name}</span>
                                <span className="text-[10px] font-semibold uppercase tracking-wide opacity-70">
                                  {updating ? '…' : enabled ? 'On' : 'Off'}
                                </span>
                              </button>
                            )
                          })}
                          <button
                            className={`rounded-lg border px-2.5 py-1.5 text-xs font-semibold transition ${
                              targetingFlagID === featureFlag.id
                                ? 'border-flagstack-500/60 bg-flagstack-500/10 text-flagstack-300'
                                : 'border-slate-700 bg-slate-900 text-slate-400 hover:border-slate-600 hover:text-slate-200'
                            }`}
                            onClick={() => setTargetingFlagID((current) => current === featureFlag.id ? undefined : featureFlag.id)}
                            type="button"
                          >
                            {targetingFlagID === featureFlag.id ? 'Close targeting' : 'Targeting & rollout'}
                          </button>
                        </div>
                      </div>
                    </div>
                  </div>

                  {targetingFlagID === featureFlag.id ? (
                    <FeatureTargetingPanel
                      canManage={canManage}
                      environments={environments}
                      featureFlag={featureFlag}
                      onClose={() => setTargetingFlagID(undefined)}
                      organisation={organisation}
                      projectID={project.id}
                    />
                  ) : null}
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

function formatDefaultValue(value: unknown): string {
  if (typeof value === 'string') {
    return JSON.stringify(value)
  }
  const serialized = JSON.stringify(value)
  return serialized ?? String(value)
}

function OverviewRow({ icon, label, value }: { icon: 'environment' | 'flag' | 'key'; label: string; value: string }) {
  return (
    <div className="flex items-center gap-3 px-5 py-4">
      <Icon className="text-slate-500" name={icon} size={17} />
      <span className="flex-1 text-xs text-slate-500">{label}</span>
      <strong className="text-sm font-semibold text-slate-200">{value}</strong>
    </div>
  )
}

function ProjectState({ message }: { message: string }) {
  return (
    <div className="mx-auto flex min-h-[calc(100vh-4rem)] max-w-xl items-center justify-center px-6 text-center">
      <div>
        <Icon className="mx-auto text-slate-700" name="project" size={28} />
        <p className="mt-3 text-sm text-slate-500">{message}</p>
        <Link className="mt-4 inline-block text-xs font-semibold text-flagstack-400 hover:text-flagstack-300" to="/">
          Return to dashboard
        </Link>
      </div>
    </div>
  )
}
