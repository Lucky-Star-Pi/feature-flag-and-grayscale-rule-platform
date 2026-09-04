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
  Radio,
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
import { api, type Rule } from '../api'

type RuleForm = {
  attribute: string
  operator: 'equals' | 'in'
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

  const { data, isLoading, error } = useQuery({
    queryKey: ['flag', id],
    queryFn: () => api.getFlag(flagId),
  })

  const invalidate = () => qc.invalidateQueries({ queryKey: ['flag', id] })

  const enableMut = useMutation({
    mutationFn: (enabled: boolean) => api.setEnabled(flagId, enabled),
    onSuccess: () => {
      message.success('状态已更新')
      invalidate()
      qc.invalidateQueries({ queryKey: ['flags'] })
    },
    onError: (e: Error) => message.error(e.message),
  })

  const saveRuleMut = useMutation({
    mutationFn: async (values: RuleForm) => {
      const expectedValue =
        values.operator === 'in'
          ? values.expectedValue
              .split(',')
              .map((s) => s.trim())
              .filter(Boolean)
          : values.expectedValue
      const body = {
        attribute: values.attribute,
        operator: values.operator,
        expectedValue,
        returnValue: values.returnValue,
        priority: values.priority,
      }
      if (editing) return api.updateRule(flagId, editing.id, body)
      return api.createRule(flagId, body)
    },
    onSuccess: () => {
      message.success(editing ? '规则已更新' : '规则已创建')
      setOpen(false)
      setEditing(null)
      form.resetFields()
      invalidate()
    },
    onError: (e: Error) => message.error(e.message),
  })

  const deleteMut = useMutation({
    mutationFn: (ruleId: number) => api.deleteRule(flagId, ruleId),
    onSuccess: () => {
      message.success('规则已删除')
      invalidate()
    },
    onError: (e: Error) => message.error(e.message),
  })

  const openCreate = () => {
    setEditing(null)
    form.resetFields()
    form.setFieldsValue({ operator: 'equals', returnValue: true, priority: 0 })
    setOpen(true)
  }

  const openEdit = (r: Rule) => {
    setEditing(r)
    let expected = r.expectedValue
    if (r.operator === 'in') {
      try {
        const arr = JSON.parse(r.expectedValue) as string[]
        expected = arr.join(',')
      } catch {
        /* keep raw */
      }
    }
    form.setFieldsValue({
      attribute: r.attribute,
      operator: r.operator,
      expectedValue: expected,
      returnValue: r.returnValue,
      priority: r.priority,
    })
    setOpen(true)
  }

  if (error) {
    return <Alert type="error" message={(error as Error).message} />
  }

  const flag = data?.flag

  return (
    <div>
      <Space style={{ marginBottom: 16, width: '100%', justifyContent: 'space-between' }}>
        <Typography.Title level={3} style={{ margin: 0 }}>
          Flag 详情与规则
        </Typography.Title>
        <Space>
          <Button>
            <Link to={`/flags/${id}/edit`}>编辑基本信息</Link>
          </Button>
          <Button>
            <Link to="/evaluate">去评估控制台</Link>
          </Button>
        </Space>
      </Space>

      <Alert
        type="info"
        showIcon
        style={{ marginBottom: 16 }}
        message="规则优先级：数字越小越先匹配（ASC）。同一 Flag 内 priority 不可重复。停用时评估恒返回 false。"
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
            <Switch
              checked={flag.enabled}
              loading={enableMut.isPending}
              onChange={(v) => enableMut.mutate(v)}
            />
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
        loading={isLoading}
        dataSource={data?.rules || []}
        locale={{ emptyText: <Empty description="暂无规则，将使用默认值" /> }}
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
        pagination={false}
        style={{ marginBottom: 32 }}
      />

      <Typography.Title level={4}>操作历史</Typography.Title>
      <Table
        rowKey="id"
        loading={isLoading}
        dataSource={data?.histories || []}
        locale={{ emptyText: <Empty description="暂无历史" /> }}
        columns={[
          { title: '时间', dataIndex: 'createdAt', render: (v: string) => new Date(v).toLocaleString() },
          { title: '类型', dataIndex: 'action', render: (v: string) => <Tag>{v}</Tag> },
          { title: '操作者', dataIndex: 'actor' },
          { title: '摘要', dataIndex: 'summary' },
        ]}
      />

      <Modal
        title={editing ? '编辑规则' : '新增规则'}
        open={open}
        onCancel={() => setOpen(false)}
        onOk={() => form.submit()}
        confirmLoading={saveRuleMut.isPending}
        destroyOnClose
      >
        <Form form={form} layout="vertical" onFinish={(v) => saveRuleMut.mutate(v)}>
          <Form.Item name="attribute" label="用户属性名" rules={[{ required: true }]}>
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
            extra="equals 填单个字符串；in 用英文逗号分隔多个值（提交时转为 JSON 数组）"
            rules={[{ required: true }]}
          >
            <Input placeholder='equals: CN；in: pro,enterprise' />
          </Form.Item>
          <Form.Item name="returnValue" label="命中返回值" rules={[{ required: true }]}>
            <Radio.Group
              options={[
                { value: true, label: 'true' },
                { value: false, label: 'false' },
              ]}
            />
          </Form.Item>
          <Form.Item
            name="priority"
            label="优先级（数字越小越先匹配）"
            rules={[{ required: true, type: 'number', min: 0 }]}
          >
            <InputNumber style={{ width: '100%' }} min={0} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}
