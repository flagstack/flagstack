import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router'
import type { OrganisationMembership } from '../auth/types'
import { Icon } from '../components/icons'
import { CreateProjectForm } from '../components/projects/CreateProjectForm'
import { Card, CardContent, CardHeader, CardTitle } from '../components/ui/Card'
import { PageHeader } from '../components/ui/PageHeader'
import { StatCard } from '../components/ui/StatCard'
import { APIError, apiRequest } from '../lib/api'
import type { Project, ProjectListResponse } from '../project/types'

const setupSteps = [
  ['Create your first project', 'Projects keep flags and environments isolated by application.'],
  ['Add environments', 'Start with development, staging and production when they make sense.'],
  ['Create a feature flag', 'Define the key, type and safe default before adding targeting rules.'],
  ['Connect an SDK', 'Use an environment-scoped key so evaluation stays local to your app.'],
] as const

interface DashboardPageProps {
  organisation: OrganisationMembership
}

export function DashboardPage({ organisation }: DashboardPageProps) {
  const [projects, setProjects] = useState<Project[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string>()
  const [creatingProject, setCreatingProject] = useState(false)

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError(undefined)

    void apiRequest<ProjectListResponse>(
      `/api/v1/organisations/${encodeURIComponent(organisation.slug)}/projects`,
    )
      .then((response) => {
        if (!cancelled) {
          setProjects(response.projects)
        }
      })
      .catch((requestError) => {
        if (!cancelled) {
          setError(requestError instanceof APIError ? requestError.message : 'Projects could not be loaded.')
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
  }, [organisation.slug])

  const environmentCount = useMemo(
    () => projects.reduce((total, project) => total + project.environment_count, 0),
    [projects],
  )
  const featureFlagCount = useMemo(
    () => projects.reduce((total, project) => total + project.feature_flag_count, 0),
    [projects],
  )
  const completedSteps = [projects.length > 0, environmentCount > 0, featureFlagCount > 0, false].filter(Boolean).length
  const canCreateProject = organisation.role === 'owner' || organisation.role === 'admin'

  return (
    <div className="mx-auto w-full max-w-[100rem] px-4 py-6 lg:px-6 lg:py-8" id="dashboard">
      <PageHeader
        actions={
          <span className="inline-flex h-8 items-center gap-2 rounded-lg border border-slate-800 bg-slate-900/60 px-3 text-xs font-medium text-slate-400">
            <span className="h-1.5 w-1.5 rounded-full bg-emerald-400 ring-4 ring-emerald-400/10" />
            {organisation.name}
          </span>
        }
        description="A quick view of projects, environments and feature-flag activity across your organisation."
        eyebrow="Workspace"
        title="Dashboard"
      />

      <div className="mt-6 grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <StatCard
          helper={projects.length === 0 ? 'Create a project to get started' : 'Active projects'}
          icon={<Icon name="project" />}
          label="Projects"
          value={loading ? '—' : String(projects.length)}
        />
        <StatCard
          accent="blue"
          helper="Scoped within projects"
          icon={<Icon name="environment" />}
          label="Environments"
          value={loading ? '—' : String(environmentCount)}
        />
        <StatCard
          accent="green"
          helper={featureFlagCount === 0 ? 'No flags configured yet' : 'Active flags'}
          icon={<Icon name="flag" />}
          label="Feature flags"
          value={loading ? '—' : String(featureFlagCount)}
        />
        <StatCard
          accent="orange"
          helper="Environment SDK access"
          icon={<Icon name="code" />}
          label="SDK keys"
          value="0"
        />
      </div>

      <div className="mt-6 grid gap-4 xl:grid-cols-12">
        <Card className="xl:col-span-7" id="projects">
          <CardHeader className="flex items-start justify-between gap-4 border-b border-slate-800">
            <div>
              <CardTitle>Projects</CardTitle>
              <p className="mt-1 text-xs text-slate-500">Applications and services using SwitchOnYourCode.</p>
            </div>
            <div className="flex items-center gap-2">
              <span className="rounded-md border border-slate-800 bg-slate-950/60 px-2 py-1 text-[10px] font-semibold text-slate-500">
                {loading ? '—' : projects.length} total
              </span>
              {canCreateProject ? (
                <button
                  className="rounded-lg bg-switchonyourcode-600 px-3 py-1.5 text-xs font-semibold text-white transition hover:bg-switchonyourcode-500"
                  onClick={() => setCreatingProject((value) => !value)}
                  type="button"
                >
                  {creatingProject ? 'Close' : 'Create project'}
                </button>
              ) : null}
            </div>
          </CardHeader>

          {creatingProject ? (
            <CreateProjectForm
              onCancel={() => setCreatingProject(false)}
              onCreated={(project) => {
                setProjects((current) => [project, ...current])
                setCreatingProject(false)
              }}
              organisation={organisation}
            />
          ) : null}

          <CardContent className="min-h-56">
            {error ? (
              <div className="m-5 rounded-lg border border-red-900/60 bg-red-950/30 px-3 py-2.5 text-xs leading-5 text-red-300">
                {error}
              </div>
            ) : loading ? (
              <div className="flex min-h-56 items-center justify-center text-xs text-slate-600">Loading projects…</div>
            ) : projects.length === 0 ? (
              <div className="flex min-h-56 flex-col items-center justify-center px-5 py-10 text-center">
                <div className="flex h-11 w-11 items-center justify-center rounded-xl border border-slate-700 bg-slate-800/60 text-slate-400">
                  <Icon name="project" size={22} />
                </div>
                <strong className="mt-3 text-sm font-medium text-slate-300">No projects yet</strong>
                <p className="mt-1 max-w-md text-xs leading-5 text-slate-600">
                  Your first project will contain its own environments, flags and SDK credentials.
                </p>
                {canCreateProject && !creatingProject ? (
                  <button
                    className="mt-4 rounded-lg bg-switchonyourcode-600 px-3 py-2 text-xs font-semibold text-white transition hover:bg-switchonyourcode-500"
                    onClick={() => setCreatingProject(true)}
                    type="button"
                  >
                    Create project
                  </button>
                ) : null}
              </div>
            ) : (
              <div className="divide-y divide-slate-800">
                {projects.map((project) => (
                  <Link
                    className="group flex items-center gap-4 px-5 py-4 transition hover:bg-slate-800/35"
                    key={project.id}
                    to={`/projects/${project.key}`}
                  >
                    <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-switchonyourcode-500/10 text-switchonyourcode-400">
                      <Icon name="project" size={17} />
                    </span>
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-2">
                        <strong className="truncate text-sm font-medium text-slate-200 group-hover:text-white">{project.name}</strong>
                        <code className="rounded bg-slate-950 px-1.5 py-0.5 text-[10px] text-slate-500">{project.key}</code>
                      </div>
                      <p className="mt-1 truncate text-xs text-slate-600">
                        {project.description || 'No description'}
                      </p>
                    </div>
                    <div className="hidden shrink-0 items-center gap-4 text-right sm:flex">
                      <div>
                        <strong className="block text-xs font-semibold text-slate-300">{project.environment_count}</strong>
                        <span className="text-[10px] text-slate-600">environments</span>
                      </div>
                      <div>
                        <strong className="block text-xs font-semibold text-slate-300">{project.feature_flag_count}</strong>
                        <span className="text-[10px] text-slate-600">flags</span>
                      </div>
                      <Icon className="text-slate-700 transition group-hover:translate-x-0.5 group-hover:text-slate-400" name="chevron-right" size={15} />
                    </div>
                  </Link>
                ))}
              </div>
            )}
          </CardContent>
        </Card>

        <Card className="xl:col-span-5">
          <CardHeader className="flex items-start justify-between gap-4 border-b border-slate-800">
            <div>
              <CardTitle>Recent activity</CardTitle>
              <p className="mt-1 text-xs text-slate-500">Changes to flags and configuration.</p>
            </div>
            <Icon className="text-slate-600" name="activity" />
          </CardHeader>
          <CardContent className="min-h-56">
            <div className="flex gap-3 px-5 py-4">
              <span className="mt-1.5 h-2 w-2 shrink-0 rounded-full border border-slate-600 bg-slate-800" />
              <div>
                <strong className="text-sm font-medium text-slate-300">No activity yet</strong>
                <p className="mt-1 text-xs leading-5 text-slate-600">Flag changes will appear here with actor and environment details.</p>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      <div className="mt-4 grid gap-4 xl:grid-cols-12">
        <Card className="xl:col-span-7" id="feature-flags">
          <CardHeader className="flex items-start justify-between gap-4 border-b border-slate-800">
            <div>
              <CardTitle>Feature flags</CardTitle>
              <p className="mt-1 text-xs text-slate-500">Configuration status across the current organisation.</p>
            </div>
            <span className="rounded-md border border-emerald-900/40 bg-emerald-950/20 px-2 py-1 text-[10px] font-semibold text-emerald-400">
              {featureFlagCount === 0 ? 'No flags' : `${featureFlagCount} active`}
            </span>
          </CardHeader>
          <CardContent>
            <div className="flex items-center justify-between gap-4 px-5 py-4">
              <div className="flex min-w-0 items-center gap-3">
                <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-switchonyourcode-500/10 text-switchonyourcode-400">
                  <Icon name="flag" size={16} />
                </span>
                <div className="min-w-0">
                  <strong className="text-sm font-medium text-slate-300">
                    {featureFlagCount === 0 ? 'No flags configured' : `${featureFlagCount} feature flags`}
                  </strong>
                  <p className="mt-0.5 truncate text-xs text-slate-600">Open a project to create flags and control them per environment.</p>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card className="xl:col-span-5" id="environments">
          <CardHeader className="border-b border-slate-800">
            <CardTitle>Environments</CardTitle>
            <p className="mt-1 text-xs text-slate-500">Delivery boundaries for SDK configuration.</p>
          </CardHeader>
          <CardContent>
            <div className="flex items-center gap-3 px-5 py-4">
              <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-blue-500/10 text-blue-400">
                <Icon name="environment" size={17} />
              </span>
              <div>
                <strong className="text-sm font-medium text-slate-300">
                  {environmentCount === 0 ? 'No environments yet' : `${environmentCount} environments`}
                </strong>
                <p className="mt-0.5 text-xs text-slate-600">Environments are created inside a project.</p>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      <Card className="mt-4">
        <CardHeader className="flex flex-col gap-3 border-b border-slate-800 sm:flex-row sm:items-start sm:justify-between">
          <div>
            <CardTitle>Getting started</CardTitle>
            <p className="mt-1 text-xs text-slate-500">The shortest path from an empty workspace to your first evaluated flag.</p>
          </div>
          <span className="w-fit rounded-md border border-slate-800 bg-slate-950/60 px-2 py-1 text-[10px] font-semibold text-slate-500">
            {completedSteps} of {setupSteps.length} complete
          </span>
        </CardHeader>
        <CardContent className="grid sm:grid-cols-2 xl:grid-cols-4">
          {setupSteps.map(([label, detail], index) => {
            const complete = index < 3 ? [projects.length > 0, environmentCount > 0, featureFlagCount > 0][index] : false
            return (
              <div className="flex gap-3 border-slate-800 p-5 sm:[&:nth-child(even)]:border-l xl:border-l xl:first:border-l-0" key={label}>
                <span
                  className={`flex h-6 w-6 shrink-0 items-center justify-center rounded-full border text-[10px] font-bold ${
                    complete
                      ? 'border-emerald-800 bg-emerald-950/40 text-emerald-400'
                      : 'border-violet-900/70 bg-violet-950/30 text-switchonyourcode-400'
                  }`}
                >
                  {complete ? '✓' : index + 1}
                </span>
                <div>
                  <strong className="text-sm font-medium text-slate-300">{label}</strong>
                  <p className="mt-1 text-xs leading-5 text-slate-600">{detail}</p>
                </div>
              </div>
            )
          })}
        </CardContent>
      </Card>
    </div>
  )
}
