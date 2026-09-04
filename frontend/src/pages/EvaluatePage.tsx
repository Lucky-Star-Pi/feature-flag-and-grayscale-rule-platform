import { useMutation } from '@tanstack/react-query'
import { Alert, Button, Form, Input, Select, Typography, Card, Descriptions, Tag } from 'antd'
import { useState } from 'react'
import { api, type EvaluateResult } from '../api'

type FormValues = {
  key: string
  environment: string
  attributesText: string
}

export default function EvaluatePage() {
  const [result, setResult] = useState<EvaluateResult | null>(null)
  const [clientError, setClientError] = useState<string | null>(null)

  const mut = useMutation({
    mutationFn: api.evaluate,
    onSuccess: (data) => {
      setResult(data)
      setClientError(null)
    },
    onError: (e: Error & { code?: string }) => {
      setResult(null)
      setClientError(`${e.code || 'ERROR'}: ${e.message}`)
    },
  })

  const onFinish = (values: FormValues) => {
    setClientError(null)
    let attributes: Record<string, unknown>
    try {
      const parsed = JSON.parse(values.attributesText)
      if (parsed === null || typeof parsed !== 'object' || Array.isArray(parsed)) {
        setClientError('INVALID_JSON: 用户属性必须是 JSON 对象')
        return
      }
      attributes = parsed
    } catch {
      setClientError('INVALID_JSON: JSON 格式错误')
      return
    }
    mut.mutate({
      key: values.key.trim(),
      environment: values.environment,
      attributes,
    })
  }

  const reasonText = (r: EvaluateResult) => {
    switch (r.reason) {
      case 'FLAG_DISABLED':
        return 'Flag 已停用，始终返回 false（未评估任何规则）'
      case 'RULE_MATCHED':
        return `命中规则 #${r.matchedRule?.id}（priority=${r.matchedRule?.priority}）`
      case 'DEFAULT_VALUE':
        return '未命中任何规则，使用 Flag 默认值'
      default:
        return r.reason
    }
  }

  return (
    <div style={{ maxWidth: 800 }}>
      <Typography.Title level={3}>在线评估控制台</Typography.Title>
      <Alert
        type="info"
        showIcon
        style={{ marginBottom: 16 }}
        message="匹配语义：priority 升序首条命中即停；属性缺失视为不命中；标量统一转 string 比较；enabled=false 短路返回 false。"
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

      {clientError ? (
        <Alert type="error" showIcon style={{ marginTop: 16 }} message={clientError} />
      ) : null}

      {result ? (
        <Card title="评估结果" style={{ marginTop: 24 }}>
          <Descriptions column={1} bordered size="small">
            <Descriptions.Item label="最终返回值">
              <Tag color={result.finalValue ? 'green' : 'red'}>{String(result.finalValue)}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="是否命中规则">
              {result.matched ? '是' : '否'}
            </Descriptions.Item>
            <Descriptions.Item label="原因">{reasonText(result)}</Descriptions.Item>
            <Descriptions.Item label="命中规则">
              {result.matchedRule ? (
                <code>
                  id={result.matchedRule.id}; {result.matchedRule.attribute} {result.matchedRule.operator}{' '}
                  {result.matchedRule.expectedValue} → {String(result.matchedRule.returnValue)}
                </code>
              ) : (
                '无（使用默认值或 Flag 停用）'
              )}
            </Descriptions.Item>
            <Descriptions.Item label="Flag">
              {result.flag.key}@{result.flag.environment} enabled={String(result.flag.enabled)} default=
              {String(result.flag.defaultValue)}
            </Descriptions.Item>
          </Descriptions>
        </Card>
      ) : null}
    </div>
  )
}
