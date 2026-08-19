export type FeatureFlagKind = 'boolean' | 'string' | 'number' | 'json'

export interface FeatureFlag {
  id: string
  project_id: string
  name: string
  key: string
  description: string
  kind: FeatureFlagKind
  default_value: unknown
  created_at: string
}

export interface FeatureFlagListResponse {
  feature_flags: FeatureFlag[]
}

export interface CreateFeatureFlagPayload {
  name: string
  key: string
  description: string
  kind: FeatureFlagKind
  default_value: unknown
}
