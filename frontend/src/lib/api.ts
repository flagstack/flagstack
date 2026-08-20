interface APIErrorBody {
  error?: {
    code?: string
    message?: string
  }
}

export class APIError extends Error {
  readonly status: number
  readonly code: string

  constructor(status: number, code: string, message: string) {
    super(message)
    this.name = 'APIError'
    this.status = status
    this.code = code
  }
}

export async function apiRequest<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers)
  const method = (init.method ?? 'GET').toUpperCase()

  if (init.body && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json')
  }

  if (!['GET', 'HEAD', 'OPTIONS'].includes(method)) {
    const csrfToken = getCookie('switchonyourcode_csrf')
    if (csrfToken) {
      headers.set('X-CSRF-Token', csrfToken)
    }
  }

  const response = await fetch(path, {
    ...init,
    headers,
    credentials: 'same-origin',
  })

  if (!response.ok) {
    let errorBody: APIErrorBody = {}
    try {
      errorBody = (await response.json()) as APIErrorBody
    } catch {
      // Fall back to the status text below when the response is not JSON.
    }

    throw new APIError(
      response.status,
      errorBody.error?.code ?? 'request_failed',
      errorBody.error?.message ?? response.statusText ?? 'The request failed.',
    )
  }

  if (response.status === 204) {
    return undefined as T
  }

  return (await response.json()) as T
}

function getCookie(name: string): string | undefined {
  const prefix = `${encodeURIComponent(name)}=`
  for (const cookie of document.cookie.split(';')) {
    const trimmed = cookie.trim()
    if (trimmed.startsWith(prefix)) {
      return decodeURIComponent(trimmed.slice(prefix.length))
    }
  }
  return undefined
}
