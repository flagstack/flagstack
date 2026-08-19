export type SDKCredentialKind = 'server' | 'client'

export interface SDKCredential {
  id: string
  organisation_id: string
  project_id: string
  environment_id: string
  name: string
  kind: SDKCredentialKind
  client_key?: string
  revoked_at?: string
  created_at: string
}

export interface SDKCredentialListResponse {
  credentials: SDKCredential[]
}

export interface CreatedSDKCredential {
  credential: SDKCredential
  key: string
}

export interface ClientVisibilityResponse {
  client_visible: boolean
}
