export interface Environment {
  id: string
  project_id: string
  name: string
  key: string
  description: string
  created_at: string
}

export interface EnvironmentListResponse {
  environments: Environment[]
}

export interface CreateEnvironmentPayload {
  name: string
  key: string
  description: string
}
