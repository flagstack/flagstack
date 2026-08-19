import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router'
import type { OrganisationMembership } from '../auth/types'
import { CreateEnvironmentForm } from '../components/environments/CreateEnvironmentForm'
import { Icon } from '../components/icons'
import { Card, CardContent, CardHeader, CardTitle } from '../components/ui/Card'
import { PageHeader } from '../components/ui/PageHeader'
import type { Environment, EnvironmentListResponse } from '../environment/types'
import { APIError, apiRequest } from '../lib/api'
import type { Project, ProjectListResponse } from '../project/types'

interface ProjectPageProps {
  organisation: OrganisationMembership
}

export function ProjectPage({ organisation }: ProjectPageProps) {
  const { projectKey } = useParams<{ projectKey: string }>()
  const [project, setProject] = useState<Project>()
  const [environments, setEnvironments] = useState<Environment[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string>()
  const [creatingEnvironment, setCreatingEnvironment] = useState(false)

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

        const environmentResponse = await apiRequest<EnvironmentListResponse>(
          `/api/v1/organisations/${encodeURIComponent(organisation.slug)}/projects/${encodeURIComponent(selected.id)}/environments`,
        )
        if (!cancelled) {
          setProject(selected)
          setEnvironments(environmentResponse.environments)
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
  }, [organisation.slug, projectKey])

  if (loading) {
    return <ProjectState message="Loading project…" />
  }

  if (error || !project) {
    return <ProjectState message={error ?? 'Project was not found.'} />
  }

  const canManageEnvironments = organisation.role !== 'viewer'

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
            {canManageEnvironments ? (
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
            <OverviewRow icon="flag" label="Feature flags" value={String(project.feature_flag_count)} />
            <OverviewRow icon="key" label="SDK keys" value="0" />
          </CardContent>
        </Card>
      </div>

      <Card className="mt-4">
        <CardHeader className="border-b border-slate-800">
          <CardTitle>Feature flags</CardTitle>
          <p className="mt-1 text-xs text-slate-500">Project-scoped definitions with per-environment configuration.</p>
        </CardHeader>
        <CardContent>
          <div className="flex items-center gap-3 px-5 py-5">
            <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-flagstack-500/10 text-flagstack-400">
              <Icon name="flag" size={17} />
            </span>
            <div>
              <strong className="text-sm font-medium text-slate-300">
                {project.feature_flag_count === 0 ? 'No feature flags yet' : `${project.feature_flag_count} active flags`}
              </strong>
              <p className="mt-1 text-xs text-slate-600">Feature-flag creation and environment configuration is the next workflow.</p>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  )
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
