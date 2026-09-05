import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  Alert,
  Button,
  Empty,
  Form,
  Input,
  Modal,
  Select,
  Space,
  Switch,
  Table,
  Tag,
  Typography,
  message,
} from 'antd'
import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { api } from '../api'
import { getErrorMessage, type Environment, type Flag } from '../types'

const envOptions = [
  { value: 'development', label: 'development' },
  { value: 'staging', label: 'staging' },
  { value: 'production', label: 'production' },
]

type CreateValues = {
  name: string
  key: string
  environment: Environment
  enabled: boolean
  defaultValue: boolean
}

type EditValues = {
  name: string
  defaultValue: boolean
}

export default function FlagListPage() {
  const nav = useNavigate()
  const qc = useQueryClient()
  const [key, setKey] = useState('')
  const [environment, setEnvironment] = useState<string | undefined>()
  const [enabled, setEnabled] = useState<string | undefined>()
  const [createOpen, setCreateOpen] = useState(false)
  const [editing, setEditing] = useState<Flag | null>(null)
  const [createForm] = Form.useForm<CreateValues>()
  const [editForm] = Form.useForm<EditValues>()

  const { data, isLoading, error, refetch, isFetching } = useQuery({
    queryKey: ['flags', key, environment, enabled],
    queryFn: () => api.listFlags({ key, environment, enabled }),
  })

  const invalidate = () => {
    qc.invalidateQueries({ queryKey: ['flags'] })
  }

  const createMut = useMutation({
    mutationFn: api.createFlag,
    onSuccess: () => {
      message.success('创建成功')
      setCreateOpen(false)
      createForm.resetFields()
      invalidate()
    },
    onError: (e: unknown) => message.error(getErrorMessage(e)),
  })

  const updateMut = useMutation({
    mutationFn: ({ id, values }: { id: number; values: EditValues }) => api.updateFlag(id, values),
    onSuccess: () => {
      message.success('已更新')
      setEditing(null)
      invalidate()
    },
    onError: (e: unknown) => message.error(getErrorMessage(e)),
  })

  const toggleMut = useMutation({
    mutationFn: ({ id, next }: { id: number; next: boolean }) =>
      next ? api.enableFlag(id) : api.disableFlag(id),
    onSuccess: () => {
      message.success('状态已更新')
      invalidate()
    },
    onError: (e: unknown) => message.error(getErrorMessage(e)),
  })

  const openEdit = (row: Flag) => {
    setEditing(row)
    editForm.setFieldsValue({ name: row.name, defaultValue: row.defaultValue })
  }

  return (
    <div>
      <Space style={{ marginBottom: 16, width: '100%', justifyContent: 'space-between' }}>
        <Typography.Title level={3} style={{ margin: 0 }}>
          Flag 列表
        </Typography.Title>
        <Button type="primary" onClick={() => setCreateOpen(true)}>
          新建 Flag
        </Button>
      </Space>

      <Alert
        type="info"
        showIcon
        style={{ marginBottom: 16 }}
        message="优先级：数字越小越先匹配。停用后评估恒返回 false。同一环境下 Key 唯一。"
      />

      <Space wrap style={{ marginBottom: 16 }}>
        <Input.Search
          placeholder="按 Key 模糊搜索"
          allowClear
          onSearch={setKey}
          style={{ width: 240 }}
          loading={isFetching}
        />
        <Select
          allowClear
          placeholder="环境"
          style={{ width: 160 }}
          value={environment}
          onChange={setEnvironment}
          options={envOptions}
        />
        <Select
          allowClear
          placeholder="启用状态"
          style={{ width: 140 }}
          value={enabled}
          onChange={setEnabled}
          options={[
            { value: 'true', label: '启用' },
            { value: 'false', label: '停用' },
          ]}
        />
        <Button onClick={() => refetch()}>刷新</Button>
      </Space>

      {error ? <Alert type="error" showIcon style={{ marginBottom: 16 }} message={getErrorMessage(error)} /> : null}

      <Table
        rowKey="id"
        loading={isLoading}
        dataSource={data?.items || []}
        locale={{
          emptyText: (
            <Empty description="暂无 Flag">
              <Button type="primary" onClick={() => setCreateOpen(true)}>
                新建 Flag
              </Button>
            </Empty>
          ),
        }}
        columns={[
          { title: '名称', dataIndex: 'name' },
          { title: 'Key', dataIndex: 'key', render: (v: string) => <code>{v}</code> },
          { title: '环境', dataIndex: 'environment', render: (v: string) => <Tag>{v}</Tag> },
          {
            title: '启用',
            dataIndex: 'enabled',
            render: (v: boolean, row: Flag) => (
              <Switch
                checked={v}
                loading={toggleMut.isPending && toggleMut.variables?.id === row.id}
                onChange={(next) => toggleMut.mutate({ id: row.id, next })}
              />
            ),
          },
          {
            title: '默认值',
            dataIndex: 'defaultValue',
            render: (v: boolean) => String(v),
          },
          {
            title: '更新时间',
            dataIndex: 'updatedAt',
            render: (v: string) => new Date(v).toLocaleString(),
          },
          {
            title: '操作',
            render: (_: unknown, row: Flag) => (
              <Space>
                <Button type="link" onClick={() => nav(`/flags/${row.id}`)}>
                  详情
                </Button>
                <Button type="link" onClick={() => openEdit(row)}>
                  编辑
                </Button>
              </Space>
            ),
          },
        ]}
      />

      <Modal
        title="新建 Flag"
        open={createOpen}
        onCancel={() => setCreateOpen(false)}
        onOk={() => createForm.submit()}
        confirmLoading={createMut.isPending}
        destroyOnHidden
      >
        <Form
          form={createForm}
          layout="vertical"
          initialValues={{ enabled: true, defaultValue: false, environment: 'development' }}
          onFinish={(v) => createMut.mutate(v)}
        >
          <Form.Item name="name" label="名称" rules={[{ required: true, max: 100 }]}>
            <Input />
          </Form.Item>
          <Form.Item name="key" label="Key" rules={[{ required: true }]}>
            <Input placeholder="如 checkout_v2" />
          </Form.Item>
          <Form.Item name="environment" label="环境" rules={[{ required: true }]}>
            <Select options={envOptions} />
          </Form.Item>
          <Form.Item name="enabled" label="启用" valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item name="defaultValue" label="默认返回值" valuePropName="checked">
            <Switch checkedChildren="true" unCheckedChildren="false" />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title="编辑 Flag"
        open={Boolean(editing)}
        onCancel={() => setEditing(null)}
        onOk={() => editForm.submit()}
        confirmLoading={updateMut.isPending}
        destroyOnHidden
      >
        <Alert type="info" showIcon style={{ marginBottom: 12 }} message="Key 与环境创建后不可修改。" />
        <Form
          form={editForm}
          layout="vertical"
          onFinish={(values) => editing && updateMut.mutate({ id: editing.id, values })}
        >
          <Form.Item name="name" label="名称" rules={[{ required: true, max: 100 }]}>
            <Input />
          </Form.Item>
          <Form.Item name="defaultValue" label="默认返回值" valuePropName="checked">
            <Switch checkedChildren="true" unCheckedChildren="false" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}
