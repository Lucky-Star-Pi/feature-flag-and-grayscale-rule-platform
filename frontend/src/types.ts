export type Environment = 'development' | 'staging' | 'production'
export type Operator = 'equals' | 'in'
export type EvalReason = 'disabled' | 'matched' | 'default'

export type Flag = {
  id: number
  name: string
  key: string
  environment: Environment
  enabled: boolean
  defaultValue: boolean
  createdAt: string
  updatedAt: string
}

export type Rule = {
  id: number
  flagId: number
  attribute: string
  operator: Operator
  expectedValue: string
  returnValue: boolean
  priority: number
  createdAt: string
  updatedAt: string
}

export type History = {
  id: number
  flagId: number | null
  operationType: string
  operator: string
  summary: string
  createdAt: string
}

export type EvalResult = {
  value: boolean
  matched: boolean
  matchedRule: Rule | null
  reason: EvalReason
}

export class ApiError extends Error {
  code: string
  status: number

  constructor(code: string, message: string, status: number) {
    super(message)
    this.name = 'ApiError'
    this.code = code
    this.status = status
  }
}

export function getErrorMessage(e: unknown): string {
  if (e instanceof ApiError) {
    if (e.code === 'KEY_CONFLICT') return '该环境下 Key 已存在'
    if (e.code === 'PRIORITY_CONFLICT') return '同一 Flag 内优先级不可重复'
    return e.message || e.code
  }
  if (e instanceof Error) return e.message
  return '请求失败'
}
