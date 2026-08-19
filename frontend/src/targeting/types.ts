export type MatchMode = 'all' | 'any'

export type ConditionOperator =
  | 'equals'
  | 'not_equals'
  | 'in'
  | 'not_in'
  | 'contains'
  | 'not_contains'
  | 'starts_with'
  | 'ends_with'
  | 'greater_than'
  | 'greater_than_or_equal'
  | 'less_than'
  | 'less_than_or_equal'
  | 'exists'
  | 'not_exists'
  | 'matches_regex'
  | 'semver_greater_than'
  | 'semver_greater_than_or_equal'
  | 'semver_less_than'
  | 'semver_less_than_or_equal'
  | 'in_segment'
  | 'not_in_segment'

export interface Variant {
  key: string
  value: unknown
}

export interface Condition {
  attribute?: string
  operator: ConditionOperator
  value?: unknown
}

export interface Allocation {
  variant: string
  weight: number
}

export interface Outcome {
  variant?: string
  rollout?: Allocation[]
  bucket_by?: string
}

export interface Rule {
  id: string
  name?: string
  match: MatchMode
  conditions: Condition[]
  outcome: Outcome
}

export interface Policy {
  rules?: Rule[]
  fallthrough?: Outcome
}

export interface EnvironmentTargetingState {
  environment_id: string
  enabled: boolean
  policy: Policy
  revision: number
}

export interface FlagTargetingState {
  id: string
  key: string
  kind: 'boolean' | 'string' | 'number' | 'json'
  default_value: unknown
  variants: Variant[]
  environments: EnvironmentTargetingState[]
}

export interface Segment {
  id: string
  project_id: string
  name: string
  key: string
  description: string
  match: MatchMode
  conditions: Condition[]
  created_at: string
  updated_at: string
}

export interface SegmentListResponse {
  segments: Segment[]
}

export interface SchedulePatch {
  enabled?: boolean
  policy?: Policy
}

export interface ScheduledChange {
  id: string
  project_id: string
  environment_id: string
  feature_flag_id: string
  created_by_user_id?: string
  execute_at: string
  patch: SchedulePatch
  status: 'pending' | 'running' | 'executed' | 'cancelled' | 'failed'
  claimed_at?: string
  executed_at?: string
  last_error?: string
  created_at: string
  updated_at: string
}

export interface ScheduledChangeListResponse {
  scheduled_changes: ScheduledChange[]
}

export interface EvaluationResult {
  value: unknown
  variant?: string
  reason: 'STATIC' | 'DEFAULT' | 'TARGETING_MATCH' | 'SPLIT' | 'DISABLED' | 'ERROR'
  rule_id?: string
  error_code?: string
  error_message?: string
}
