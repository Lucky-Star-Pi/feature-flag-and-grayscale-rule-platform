import { useMutation } from '@tanstack/react-query'
import { Alert, Button, Card, Descriptions, Form, Input, Select, Tag, Typography } from 'antd'
import { useState } from 'react'
import { api } from '../api'
import { getErrorMessage, type Environment, type EvalResult } from '../types'

type FormValues = {
  key: string
  environment: Environment
  attributesText: string
}

function reasonText(r: EvalResult): string {
  switch (r.reason) {
    case 'disabled':
      return 'Flag 已停用，恒返回 false（未评估任何规则）'
    case 'matched':
      return `命中规则 #${r.matchedRule?.id}（priority=${r.matchedRule?.priority}）`
    case 'default':
      return `未命中，使用默认值（defaultValue 即本次 value=${String(r.value)}）`
    default:
      return r.reason
  }
}

export default function EvaluatePage() {
  const [result, setResult] = useState<EvalResult | null>(null)
  const [clientError, setClientError] = useState<string | null>(null)

  const mut = useMutation({
    mutationFn: api.evaluate,
    onSuccess: (data) => {
      setResult(data)
      setClientError(null)
    },
    onError: (e: unknown) => {
      setResult(null)
      setClientError(getErrorMessage(e))
    },
  })

  const onFinish = (values: FormValues) => {
    setClientError(null)
    setResult(null)
    let parsed: unknown
    try {
      parsed = JSON.parse(values.attributesText)
    } catch {
      setClientError('JSON 格式错误：attributes 不是合法 JSON')
      return
    }
    if (parsed === null || typeof parsed !== 'object' || Array.isArray(parsed)) {
      setClientError('attributes 必须是 JSON 对象')
      return
    }
    mut.mutate({
      key: values.key.trim(),
      environment: values.environment,
      attributes: parsed as Record<string, unknown>,
    })
  }

  return (
    <div style={{ maxWidth: 800 }}>
      <Typography.Title level={3}>在线评估控制台</Typography.Title>
      <Alert
        type="info"
        showIcon
        style={{ marginBottom: 16 }}
        message="匹配语义：priority 数字越小越先匹配；属性缺失视为不命中；标量转 string 精确比较；enabled=false 短路返回 false。"
      />

      <Form
        layout="vertical"
        initialValues={{
          key: 'checkout_v2',
          environment: 'development',
          attributesText: '{\n  "country": "CN",\n  "plan": "pro"\n}',
        }}
        onFinish={onFinish}
      >
        <Form.Item name="key" label="Flag Key" rules={[{ required: true }]}>
          <Input />
        </Form.Item>
        <Form.Item name="environment" label="环境" rules={[{ required: true }]}>
          <Select
            options={[
              { value: 'development', label: 'development' },
              { value: 'staging', label: 'staging' },
              { value: 'production', label: 'production' },
            ]}
          />
        </Form.Item>
        <Form.Item name="attributesText" label="用户属性 JSON" rules={[{ required: true }]}>
          <Input.TextArea rows={8} style={{ fontFamily: 'monospace' }} />
        </Form.Item>
        <Button type="primary" htmlType="submit" loading={mut.isPending}>
          评估
        </Button>
      </Form>

      {clientError ? <Alert type="error" showIcon style={{ marginTop: 16 }} message={clientError} /> : null}

      {result ? (
        <Card title="评估结果" style={{ marginTop: 24 }}>
          <Descriptions column={1} bordered size="small">
            <Descriptions.Item label="最终返回值">
              <Tag color={result.value ? 'green' : 'red'}>{String(result.value)}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="是否命中规则">{result.matched ? '是' : '否'}</Descriptions.Item>
            <Descriptions.Item label="原因">{reasonText(result)}</Descriptions.Item>
            <Descriptions.Item label="命中规则">
              {result.matchedRule ? (
                <code>
                  id={result.matchedRule.id}; {result.matchedRule.attribute} {result.matchedRule.operator}{' '}
                  {result.matchedRule.expectedValue} → {String(result.matchedRule.returnValue)}
                  （priority={result.matchedRule.priority}）
                </code>
              ) : result.reason === 'disabled' ? (
                '无（Flag 已停用，恒返回 false）'
              ) : (
                `无（未命中，使用默认值 defaultValue=${String(result.value)}）`
              )}
            </Descriptions.Item>
          </Descriptions>
        </Card>
      ) : null}
    </div>
  )
}
