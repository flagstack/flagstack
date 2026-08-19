import { Icon } from '../components/icons'
import { Card, CardContent, CardHeader, CardTitle } from '../components/ui/Card'
import { PageHeader } from '../components/ui/PageHeader'
import { StatCard } from '../components/ui/StatCard'

const setupSteps = [
  { label: 'Create your first project', detail: 'Projects keep flags and environments isolated by application.' },
  { label: 'Add environments', detail: 'Start with development, staging and production when they make sense.' },
  { label: 'Create a feature flag', detail: 'Define the key, type and safe default before adding targeting rules.' },
  { label: 'Connect an SDK', detail: 'Use an environment-scoped key so evaluation stays local to your app.' },
]

export function DashboardPage() {
  return (
    <div className="page-container" id="dashboard">
      <PageHeader
        actions={
          <span className="preview-pill">
            <span className="status-dot status-dot--preview" />
            Development preview
          </span>
        }
        description="A quick view of projects, environments and feature-flag activity across your organisation."
        eyebrow="Workspace"
        title="Dashboard"
      />

      <div className="stat-grid">
        <StatCard
          helper="Create a project to get started"
          icon={<Icon name="project" />}
          label="Projects"
          value="0"
        />
        <StatCard
          accent="blue"
          helper="Scoped within projects"
          icon={<Icon name="environment" />}
          label="Environments"
          value="0"
        />
        <StatCard
          accent="green"
          helper="No flags configured yet"
          icon={<Icon name="flag" />}
          label="Feature flags"
          value="0"
        />
        <StatCard
          accent="orange"
          helper="Environment SDK access"
          icon={<Icon name="code" />}
          label="SDK keys"
          value="0"
        />
      </div>

      <div className="dashboard-grid dashboard-grid--primary">
        <Card className="dashboard-span-7" id="projects">
          <CardHeader className="card-header--split">
            <div>
              <CardTitle>Projects</CardTitle>
              <p>Applications and services using FlagStack.</p>
            </div>
            <span className="card-count">0 total</span>
          </CardHeader>
          <CardContent className="empty-state">
            <div className="empty-state-icon">
              <Icon name="project" size={22} />
            </div>
            <strong>No projects yet</strong>
            <p>Your first project will contain its own environments, flags and SDK credentials.</p>
            <button disabled type="button">Create project</button>
          </CardContent>
        </Card>

        <Card className="dashboard-span-5">
          <CardHeader className="card-header--split">
            <div>
              <CardTitle>Recent activity</CardTitle>
              <p>Changes to flags and configuration.</p>
            </div>
            <Icon className="card-header-icon" name="activity" />
          </CardHeader>
          <CardContent className="empty-list">
            <div className="empty-list-row">
              <span className="activity-marker" />
              <div>
                <strong>No activity yet</strong>
                <p>Flag changes will appear here with actor and environment details.</p>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      <div className="dashboard-grid">
        <Card className="dashboard-span-7" id="feature-flags">
          <CardHeader className="card-header--split">
            <div>
              <CardTitle>Feature flags</CardTitle>
              <p>Configuration status across the current organisation.</p>
            </div>
            <span className="status-badge">All clear</span>
          </CardHeader>
          <CardContent className="table-like-list">
            <div className="table-like-row table-like-row--muted">
              <div className="table-like-main">
                <span className="flag-glyph"><Icon name="flag" size={16} /></span>
                <div>
                  <strong>No flags configured</strong>
                  <p>Flags will be grouped by project with their type and rollout state.</p>
                </div>
              </div>
              <span>—</span>
            </div>
          </CardContent>
        </Card>

        <Card className="dashboard-span-5" id="environments">
          <CardHeader>
            <CardTitle>Environments</CardTitle>
            <p>Delivery boundaries for SDK configuration.</p>
          </CardHeader>
          <CardContent className="environment-list">
            <div className="environment-row environment-row--empty">
              <div className="environment-icon"><Icon name="environment" size={17} /></div>
              <div>
                <strong>No environments yet</strong>
                <p>Environments are created inside a project.</p>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      <Card className="getting-started">
        <CardHeader className="card-header--split">
          <div>
            <CardTitle>Getting started</CardTitle>
            <p>The shortest path from an empty workspace to your first evaluated flag.</p>
          </div>
          <span className="setup-progress">0 of {setupSteps.length} complete</span>
        </CardHeader>
        <CardContent className="setup-grid">
          {setupSteps.map((step, index) => (
            <div className="setup-step" key={step.label}>
              <span className="setup-step-number">{index + 1}</span>
              <div>
                <strong>{step.label}</strong>
                <p>{step.detail}</p>
              </div>
            </div>
          ))}
        </CardContent>
      </Card>
    </div>
  )
}
