export interface UserIdentity {
  id: string
  email: string
  display_name: string
}

export interface OrganisationMembership {
  id: string
  name: string
  slug: string
  role: 'owner' | 'admin' | 'developer' | 'viewer'
}

export interface Principal {
  user: UserIdentity
  organisations: OrganisationMembership[]
}

export interface BootstrapStatus {
  required: boolean
}

export interface BootstrapPayload {
  email: string
  display_name: string
  password: string
  organisation_name: string
  organisation_slug: string
}

export interface LoginPayload {
  email: string
  password: string
}
