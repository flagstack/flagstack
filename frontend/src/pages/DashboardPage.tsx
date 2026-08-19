import { Icon } from '../components/icons'
import { Card, CardContent, CardHeader, CardTitle } from '../components/ui/Card'
import { PageHeader } from '../components/ui/PageHeader'
import { StatCard } from '../components/ui/StatCard'

const setupSteps = [
  ['Create your first project', 'Projects keep flags and environments isolated by application.'],
  ['Add environments', 'Start with development, staging and production when they make sense.'],
  ['Create a feature flag', 'Define the key, type and safe default before adding targeting rules.'],
  ['Connect an SDK', 'Use an environment-scoped key so evaluation stays local to your app.'],
] as const

export function DashboardPage() {
  return (
    <div className="mx-auto w-full max-w-[100rem] px-4 py-6 lg:px-6 lg:py-8" id="dashboard">
      <PageHeader
        actions={
          <span className="inline-flex h-8 items-center gap-2 rounded-lg border border-slate-800 bg-slate-900/60 px-3 text-xs font-medium text-slate-400">
            <span className="h-1.5 w-1.5 rounded-full bg-flagstack-400 ring-4 ring-flagstack-400/10" />
            Development preview
          </span>
        }
        description="A quick view of projects, environments and feature-flag activity across your organisation."
        eyebrow="Workspace"
        title="Dashboard"
      />

      <div className="mt-6 grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <StatCard helper="Create a project to get started" icon={<Icon name="project" />} label="Projects" value="0" />
        <StatCard accent="blue" helper="Scoped within projects" icon={<Icon name="environment" />} label="Environments" value="0" />
        <StatCard accent="green" helper="No flags configured yet" icon={<Icon name="flag" />} label="Feature flags" value="0" />
        <StatCard accent="orange" helper="Environment SDK access" icon={<Icon name="code" />} label="SDK keys" value="0" />
      </div>

      <div className="mt-6 grid gap-4 xl:grid-cols-12">
        <Card className="xl:col-span-7" id="projects">
          <CardHeader className="flex items-start justify-between gap-4 border-b border-slate-800">
            <div>
              <CardTitle>Projects</CardTitle>
              <p className="mt-1 text-xs text-slate-500">Applications and services using FlagStack.</p>
            </div>
            <span className="rounded-md border border-slate-800 bg-slate-950/60 px-2 py-1 text-[10px] font-semibold text-slate-500">0 total</span>
          </CardHeader>
          <CardContent className="flex min-h-56 flex-col items-center justify-center px-5 py-10 text-center">
            <div className="flex h-11 w-11 items-center justify-center rounded-xl border border-slate-700 bg-slate-800/60 text-slate-400">
              <Icon name="project" size={22} />
            </div>
            <strong className="mt-3 text-sm font-medium text-slate-300">No projects yet</strong>
            <p className="mt-1 max-w-md text-xs leading-5 text-slate-600">Your first project will contain its own environments, flags and SDK credentials.</p>
            <button className="mt-4 cursor-not-allowed rounded-lg border border-violet-900/70 bg-violet-950/40 px-3 py-2 text-xs font-semibold text-violet-300 opacity-60" disabled type="button">Create project</button>
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
            <span className="rounded-md border border-emerald-900/40 bg-emerald-950/20 px-2 py-1 text-[10px] font-semibold text-emerald-400">All clear</span>
          </CardHeader>
          <CardContent>
            <div className="flex items-center justify-between gap-4 px-5 py-4">
              <div className="flex min-w-0 items-center gap-3">
                <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-flagstack-500/10 text-flagstack-400"><Icon name="flag" size={16} /></span>
                <div className="min-w-0">
                  <strong className="text-sm font-medium text-slate-300">No flags configured</strong>
                  <p className="mt-0.5 truncate text-xs text-slate-600">Flags will be grouped by project with their type and rollout state.</p>
                </div>
              </div>
              <span className="text-xs text-slate-700">—</span>
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
              <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-blue-500/10 text-blue-400"><Icon name="environment" size={17} /></span>
              <div>
                <strong className="text-sm font-medium text-slate-300">No environments yet</strong>
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
          <span className="w-fit rounded-md border border-slate-800 bg-slate-950/60 px-2 py-1 text-[10px] font-semibold text-slate-500">0 of {setupSteps.length} complete</span>
        </CardHeader>
        <CardContent className="grid sm:grid-cols-2 xl:grid-cols-4">
          {setupSteps.map(([label, detail], index) => (
            <div className="flex gap-3 border-slate-800 p-5 sm:[&:nth-child(even)]:border-l xl:border-l xl:first:border-l-0" key={label}>
              <span className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full border border-violet-900/70 bg-violet-950/30 text-[10px] font-bold text-flagstack-400">{index + 1}</span>
              <div>
                <strong className="text-sm font-medium text-slate-300">{label}</strong>
                <p className="mt-1 text-xs leading-5 text-slate-600">{detail}</p>
              </div>
            </div>
          ))}
        </CardContent>
      </Card>
    </div>
  )
}
