export interface Project {
  id: string
  name: string
  key: string
  description: string
  environment_count: number
  feature_flag_count: number
  created_at: string
}

export interface ProjectListResponse {
  projects: Project[]
}

export interface CreateProjectPayload {
  name: string
  key: string
  description: string
}
