const API_BASE = "/api/v1"

interface ApiError {
  request_id: string
  error_code: string
  message: string
  diagnosis: string
  suggested_action: string
}

class ApiClientError extends Error {
  constructor(
    public status: number,
    public apiError: ApiError,
  ) {
    super(apiError.message)
  }
}

function getToken(): string {
  return sessionStorage.getItem("API_BEARER_TOKEN") || ""
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const token = getToken()
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...(options.headers as Record<string, string>),
  }
  if (token) {
    headers["Authorization"] = `Bearer ${token}`
  }

  const controller = new AbortController()
  const timeoutMs = path.startsWith("/feishu") ? 120_000 : 10_000
  const timeout = setTimeout(() => controller.abort(), timeoutMs)

  try {
    const resp = await fetch(`${API_BASE}${path}`, {
      ...options,
      headers,
      signal: controller.signal,
    })

    if (!resp.ok) {
      const body = await resp.json().catch(() => ({}))
      if (body.error_code) {
        throw new ApiClientError(resp.status, body as ApiError)
      }
      throw new Error(`HTTP ${resp.status}: ${resp.statusText}`)
    }

    return (await resp.json()) as T
  } finally {
    clearTimeout(timeout)
  }
}

export const apiClient = {
  get<T>(path: string, params?: Record<string, string>): Promise<T> {
    const qs = params ? "?" + new URLSearchParams(params).toString() : ""
    return request<T>(`${path}${qs}`)
  },
  post<T>(path: string, body?: unknown): Promise<T> {
    return request<T>(path, {
      method: "POST",
      body: body ? JSON.stringify(body) : undefined,
    })
  },
}

export { ApiClientError }
export type { ApiError }
