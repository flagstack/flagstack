export interface FlagConfig {
  environment_id: string
  feature_flag_id: string
  enabled: boolean
  revision: number
}

export interface FlagConfigListResponse {
  configs: FlagConfig[]
}

export interface SetFlagEnabledPayload {
  enabled: boolean
}
