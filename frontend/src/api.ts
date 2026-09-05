import { ApiError, type Environment, type EvalResult, type Flag, type History, type Rule } from './types'

export async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const headers = new Headers(init?.headers)
  if (init?.body && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json')
  }
  const res = await fetch(path, { ...init, headers })
  if (res.status === 204) {
    return undefined as T
  }
  const data: unknown = await res.json().catch(() => ({}))
  if (!res.ok) {
    const payload = data as { error?: { code?: string; message?: string } }
    throw new ApiError(
      payload.error?.code || 'INTERNAL_ERROR',
      payload.error?.message || res.statusText,
      res.status,
    )
  }
  return data as T
}

function qs(params: Record<string, string | undefined>): string {
  const sp = new URLSearchParams()
  Object.entries(params).forEach(([k, v]) => {
    if (v) sp.set(k, v)
  })
  const s = sp.toString()
  return s ? `?${s}` : ''
}

export const api = {
  listFlags: (params: { key?: string; environment?: string; enabled?: string }) =>
    apiFetch<{ items: Flag[] }>(`/api/v1/flags${qs(params)}`),

  getFlag: (id: number) => apiFetch<{ flag: Flag; rules: Rule[] }>(`/api/v1/flags/${id}`),

  createFlag: (body: {
    name: string
    key: string
    environment: Environment
    enabled: boolean
    defaultValue: boolean
  }) => apiFetch<{ flag: Flag }>('/api/v1/flags', { method: 'POST', body: JSON.stringify(body) }),

  updateFlag: (id: number, body: { name: string; defaultValue: boolean; version: number }) =>
    apiFetch<{ flag: Flag }>(`/api/v1/flags/${id}`, { method: 'PATCH', body: JSON.stringify(body) }),

  enableFlag: (id: number) =>
    apiFetch<{ flag: Flag }>(`/api/v1/flags/${id}/enable`, { method: 'POST' }),

  disableFlag: (id: number) =>
    apiFetch<{ flag: Flag }>(`/api/v1/flags/${id}/disable`, { method: 'POST' }),

  createRule: (
    flagId: number,
    body: {
      attribute: string
      operator: 'equals' | 'in'
      expectedValue: string
      returnValue: boolean
      priority: number
    },
  ) =>
    apiFetch<{ rule: Rule }>(`/api/v1/flags/${flagId}/rules`, {
      method: 'POST',
      body: JSON.stringify(body),
    }),

  updateRule: (
    flagId: number,
    ruleId: number,
    body: {
      attribute: string
      operator: 'equals' | 'in'
      expectedValue: string
      returnValue: boolean
      priority: number
      version: number
    },
  ) =>
    apiFetch<{ rule: Rule }>(`/api/v1/flags/${flagId}/rules/${ruleId}`, {
      method: 'PATCH',
      body: JSON.stringify(body),
    }),

  deleteRule: (flagId: number, ruleId: number) =>
    apiFetch<void>(`/api/v1/flags/${flagId}/rules/${ruleId}`, { method: 'DELETE' }),

  listHistory: (flagId: number) => apiFetch<{ items: History[] }>(`/api/v1/flags/${flagId}/history`),

  evaluate: (body: { key: string; environment: Environment; attributes: Record<string, unknown> }) =>
    apiFetch<EvalResult>('/api/v1/evaluate', { method: 'POST', body: JSON.stringify(body) }),
}

export function isJsonStringArray(text: string): boolean {
  try {
    const parsed: unknown = JSON.parse(text)
    return Array.isArray(parsed) && parsed.every((x) => typeof x === 'string')
  } catch {
    return false
  }
}
