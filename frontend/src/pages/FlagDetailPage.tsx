import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  Alert,
  Button,
  Descriptions,
  Empty,
  Form,
  Input,
  InputNumber,
  Modal,
  Popconfirm,
  Select,
  Space,
  Switch,
  Table,
  Tag,
  Typography,
  message,
} from 'antd'
import { useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { api, isJsonStringArray } from '../api'
import { ApiError, getErrorMessage, type Operator, type Rule } from '../types'

type RuleForm = {
  attribute: string
  operator: Operator
  expectedValue: string
  returnValue: boolean
  priority: number
}

export default function FlagDetailPage() {
  const { id } = useParams()
  const flagId = Number(id)
  const qc = useQueryClient()
  const [open, setOpen] = useState(false)
  const [editing, setEditing] = useState<Rule | null>(null)
  const [form] = Form.useForm<RuleForm>()
  const operator = Form.useWatch('operator', form)

  const detailQuery = useQuery({
    queryKey: ['flag', flagId],
    enabled: Number.isFinite(flagId) && flagId > 0,
    queryFn: () => api.getFlag(flagId),
  })
  const histQuery = useQuery({
    queryKey: ['history', flagId],
    enabled: Number.isFinite(flagId) && flagId > 0,
    queryFn: () => api.listHistory(flagId),
  })

  const invalidate = () => {
    qc.invalidateQueries({ queryKey: ['flag', flagId] })
    qc.invalidateQueries({ queryKey: ['history', flagId] })
    qc.invalidateQueries({ queryKey: ['flags'] })
  }

  const toggleMut = useMutation({
    mutationFn: (next: boolean) => (next ? api.enableFlag(flagId) : api.disableFlag(flagId)),
    onSuccess: () => {
      message.success('状态已更新')
      invalidate()
    },
    onError: (e: unknown) => message.error(getErrorMessage(e)),
  })

  const saveMut = useMutation({
    mutationFn: (values: RuleForm) => {
      const body = {
        attribute: values.attribute.trim(),
        operator: values.operator,
        expectedValue: values.expectedValue.trim(),
        returnValue: values.returnValue,
        priority: values.priority,
      }
      if (editing) return api.updateRule(flagId, editing.id, { ...body, version: editing.version })
      return api.createRule(flagId, body)
    },
    onSuccess: () => {
      message.success(editing ? '规则已更新' : '规则已创建')
      setOpen(false)
      setEditing(null)
      form.resetFields()
      invalidate()
    },
    onError: (e: unknown) => {
      message.error(getErrorMessage(e))
      if (e instanceof ApiError && e.code === 'VERSION_CONFLICT') {
        setOpen(false)
        setEditing(null)
        qc.invalidateQueries({ queryKey: ['flag', flagId] })
        qc.invalidateQueries({ queryKey: ['history', flagId] })
      }
    },
  })

  const deleteMut = useMutation({
    mutationFn: (ruleId: number) => api.deleteRule(flagId, ruleId),
    onSuccess: () => {
      message.success('规则已删除')
      invalidate()
    },
    onError: (e: unknown) => message.error(getErrorMessage(e)),
  })

  const openCreate = () => {
    setEditing(null)
    form.resetFields()
    form.setFieldsValue({ operator: 'equals', returnValue: true, priority: 0 })
    setOpen(true)
  }

  const openEdit = (r: Rule) => {
    // 记录打开瞬间的 version 快照，PATCH body 带该值。
    setEditing(r)
    form.setFieldsValue({
      attribute: r.attribute,
      operator: r.operator,
      expectedValue: r.expectedValue,
      returnValue: r.returnValue,
      priority: r.priority,
    })
    setOpen(true)
  }

  if (detailQuery.error) {
    return <Alert type="error" showIcon message={getErrorMessage(detailQuery.error)} />
  }

  const flag = detailQuery.data?.flag

  return (
    <div>
      <Space style={{ marginBottom: 16, width: '100%', justifyContent: 'space-between' }}>
        <Typography.Title level={3} style={{ margin: 0 }}>
          Flag 详情与规则
        </Typography.Title>
        <Button>
          <Link to="/evaluate">去评估控制台</Link>
        </Button>
      </Space>

      <Alert
        type="info"
        showIcon
        style={{ marginBottom: 16 }}
        message="规则优先级：数字越小越先匹配（ASC）。同一 Flag 内 priority 不可重复。in 的期望值须为 JSON 字符串数组，如 [&quot;pro&quot;,&quot;enterprise&quot;]。停用后评估恒返回 false。"
      />

      {flag ? (
        <Descriptions bordered size="small" column={2} style={{ marginBottom: 24 }}>
          <Descriptions.Item label="名称">{flag.name}</Descriptions.Item>
          <Descriptions.Item label="Key">
            <code>{flag.key}</code>
          </Descriptions.Item>
          <Descriptions.Item label="环境">{flag.environment}</Descriptions.Item>
          <Descriptions.Item label="默认值">{String(flag.defaultValue)}</Descriptions.Item>
          <Descriptions.Item label="启用">
            <Switch checked={flag.enabled} loading={toggleMut.isPending} onChange={(v) => toggleMut.mutate(v)} />
          </Descriptions.Item>
          <Descriptions.Item label="更新时间">{new Date(flag.updatedAt).toLocaleString()}</Descriptions.Item>
        </Descriptions>
      ) : null}

      <Space style={{ marginBottom: 12, width: '100%', justifyContent: 'space-between' }}>
        <Typography.Title level={4} style={{ margin: 0 }}>
          匹配规则（按 priority 升序）
        </Typography.Title>
        <Button type="primary" onClick={openCreate}>
          新增规则
        </Button>
      </Space>

      <Table
        rowKey="id"
        loading={detailQuery.isLoading}
        dataSource={detailQuery.data?.rules || []}
        locale={{ emptyText: <Empty description="暂无规则，评估将使用默认值" /> }}
        pagination={false}
        style={{ marginBottom: 32 }}
        columns={[
          { title: '优先级', dataIndex: 'priority', width: 90 },
          { title: '属性', dataIndex: 'attribute' },
          { title: '操作符', dataIndex: 'operator', render: (v: string) => <Tag>{v}</Tag> },
          { title: '期望值', dataIndex: 'expectedValue', render: (v: string) => <code>{v}</code> },
          { title: '返回值', dataIndex: 'returnValue', render: (v: boolean) => String(v) },
          {
            title: '操作',
            render: (_: unknown, r: Rule) => (
              <Space>
                <Button type="link" onClick={() => openEdit(r)}>
                  编辑
                </Button>
                <Popconfirm title="确认删除该规则？" onConfirm={() => deleteMut.mutate(r.id)}>
                  <Button type="link" danger>
                    删除
                  </Button>
                </Popconfirm>
              </Space>
            ),
          },
        ]}
      />

      <Typography.Title level={4}>操作历史</Typography.Title>
      {histQuery.error ? (
        <Alert type="error" showIcon message={getErrorMessage(histQuery.error)} style={{ marginBottom: 12 }} />
      ) : null}
      <Table
        rowKey="id"
        loading={histQuery.isLoading}
        dataSource={histQuery.data?.items || []}
        locale={{ emptyText: <Empty description="暂无历史" /> }}
        columns={[
          { title: '时间', dataIndex: 'createdAt', render: (v: string) => new Date(v).toLocaleString() },
          { title: '类型', dataIndex: 'operationType', render: (v: string) => <Tag>{v}</Tag> },
          { title: '操作者', dataIndex: 'operator' },
          { title: '摘要', dataIndex: 'summary' },
        ]}
      />

      <Modal
        title={editing ? '编辑规则' : '新增规则'}
        open={open}
        onCancel={() => setOpen(false)}
        onOk={() => form.submit()}
        confirmLoading={saveMut.isPending}
        destroyOnHidden
      >
        <Form form={form} layout="vertical" onFinish={(v) => saveMut.mutate(v)}>
          <Form.Item name="attribute" label="用户属性名" rules={[{ required: true, message: 'attribute 必填' }]}>
            <Input placeholder="如 country / plan / user_id" />
          </Form.Item>
          <Form.Item name="operator" label="操作符" rules={[{ required: true }]}>
            <Select
              options={[
                { value: 'equals', label: 'equals' },
                { value: 'in', label: 'in' },
              ]}
            />
          </Form.Item>
          <Form.Item
            name="expectedValue"
            label="期望值"
            extra={
              operator === 'in'
                ? 'in：必须是 JSON 字符串数组，如 ["pro","enterprise"]'
                : 'equals：与用户属性字符串化后精确相等'
            }
            rules={[
              { required: true, message: 'expectedValue 必填' },
              {
                validator: async (_, value: string) => {
                  if (form.getFieldValue('operator') !== 'in') return
                  if (!isJsonStringArray(value || '')) {
                    throw new Error('in 的 expectedValue 必须是 JSON 字符串数组，如 ["pro"]')
                  }
                },
              },
            ]}
          >
            <Input placeholder='equals: CN；in: ["pro","enterprise"]' />
          </Form.Item>
          <Form.Item name="returnValue" label="命中返回值" valuePropName="checked">
            <Switch checkedChildren="true" unCheckedChildren="false" />
          </Form.Item>
          <Form.Item
            name="priority"
            label="优先级（数字越小越先匹配）"
            rules={[{ required: true, type: 'number', min: 0, message: 'priority 必须是 ≥0 的整数' }]}
          >
            <InputNumber style={{ width: '100%' }} min={0} precision={0} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}
