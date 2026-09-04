export type Flag = {
  id: number
  name: string
  key: string
  environment: 'development' | 'staging' | 'production'
  enabled: boolean
  defaultValue: boolean
  createdAt: string
  updatedAt: string
}

export type Rule = {
  id: number
  flagId: number
  attribute: string
  operator: 'equals' | 'in'
  expectedValue: string
  returnValue: boolean
  priority: number
  createdAt: string
  updatedAt: string
}

export type History = {
  id: number
  flagId: number
  actor: string
  action: string
  summary: string
  createdAt: string
}

export type FlagDetail = {
  flag: Flag
  rules: Rule[]
  histories: History[]
}

export type EvaluateResult = {
  finalValue: boolean
  matched: boolean
  matchedRule: Rule | null
  reason: 'FLAG_DISABLED' | 'RULE_MATCHED' | 'DEFAULT_VALUE'
  flag: Flag
}

export type ApiError = {
  code: string
  message: string
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    headers: { 'Content-Type': 'application/json', ...(init?.headers || {}) },
    ...init,
  })
  if (res.status === 204) {
    return undefined as T
  }
  const data = await res.json().catch(() => ({}))
  if (!res.ok) {
    const err = data as ApiError
    throw Object.assign(new Error(err.message || res.statusText), {
      code: err.code || 'INTERNAL_ERROR',
      status: res.status,
    })
  }
  return data as T
}

export const api = {
  listFlags: (params: { q?: string; environment?: string; enabled?: string }) => {
    const qs = new URLSearchParams()
    if (params.q) qs.set('q', params.q)
    if (params.environment) qs.set('environment', params.environment)
    if (params.enabled) qs.set('enabled', params.enabled)
    const q = qs.toString()
    return request<{ items: Flag[] }>(`/api/v1/flags${q ? `?${q}` : ''}`)
  },
  getFlag: (id: number) => request<FlagDetail>(`/api/v1/flags/${id}`),
  createFlag: (body: {
    name: string
    key: string
    environment: string
    enabled: boolean
    defaultValue: boolean
  }) => request<{ flag: Flag }>('/api/v1/flags', { method: 'POST', body: JSON.stringify(body) }),
  updateFlag: (id: number, body: { name: string; defaultValue: boolean }) =>
    request<{ flag: Flag }>(`/api/v1/flags/${id}`, { method: 'PUT', body: JSON.stringify(body) }),
  setEnabled: (id: number, enabled: boolean) =>
    request<{ flag: Flag }>(`/api/v1/flags/${id}/enabled`, {
      method: 'PATCH',
      body: JSON.stringify({ enabled }),
    }),
  createRule: (
    flagId: number,
    body: {
      attribute: string
      operator: string
      expectedValue: string | string[]
      returnValue: boolean
      priority: number
    },
  ) =>
    request<{ rule: Rule }>(`/api/v1/flags/${flagId}/rules`, {
      method: 'POST',
      body: JSON.stringify(body),
    }),
  updateRule: (
    flagId: number,
    ruleId: number,
    body: {
      attribute: string
      operator: string
      expectedValue: string | string[]
      returnValue: boolean
      priority: number
    },
  ) =>
    request<{ rule: Rule }>(`/api/v1/flags/${flagId}/rules/${ruleId}`, {
      method: 'PUT',
      body: JSON.stringify(body),
    }),
  deleteRule: (flagId: number, ruleId: number) =>
    request<void>(`/api/v1/flags/${flagId}/rules/${ruleId}`, { method: 'DELETE' }),
  evaluate: (body: { key: string; environment: string; attributes: Record<string, unknown> }) =>
    request<EvaluateResult>('/api/v1/evaluate', { method: 'POST', body: JSON.stringify(body) }),
}
