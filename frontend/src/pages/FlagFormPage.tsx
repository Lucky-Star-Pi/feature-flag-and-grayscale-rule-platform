import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Alert, Button, Form, Input, Radio, Select, Space, Switch, Typography, message } from 'antd'
import { useNavigate, useParams } from 'react-router-dom'
import { api } from '../api'

type FormValues = {
  name: string
  key: string
  environment: string
  enabled: boolean
  defaultValue: boolean
}

export default function FlagFormPage() {
  const { id } = useParams()
  const isEdit = Boolean(id)
  const nav = useNavigate()
  const qc = useQueryClient()
  const [form] = Form.useForm<FormValues>()

  const detailQuery = useQuery({
    queryKey: ['flag', id],
    enabled: isEdit,
    queryFn: () => api.getFlag(Number(id)),
  })

  if (isEdit && detailQuery.data && !form.isFieldsTouched()) {
    const f = detailQuery.data.flag
    form.setFieldsValue({
      name: f.name,
      key: f.key,
      environment: f.environment,
      enabled: f.enabled,
      defaultValue: f.defaultValue,
    })
  }

  const createMut = useMutation({
    mutationFn: api.createFlag,
    onSuccess: (res) => {
      message.success('创建成功')
      qc.invalidateQueries({ queryKey: ['flags'] })
      nav(`/flags/${res.flag.id}`)
    },
    onError: (e: Error & { code?: string }) => {
      message.error(e.message || '创建失败')
    },
  })

  const updateMut = useMutation({
    mutationFn: (values: FormValues) =>
      api.updateFlag(Number(id), { name: values.name, defaultValue: values.defaultValue }),
    onSuccess: () => {
      message.success('更新成功')
      qc.invalidateQueries({ queryKey: ['flags'] })
      qc.invalidateQueries({ queryKey: ['flag', id] })
      nav(`/flags/${id}`)
    },
    onError: (e: Error) => message.error(e.message || '更新失败'),
  })

  const onFinish = (values: FormValues) => {
    if (isEdit) updateMut.mutate(values)
    else createMut.mutate(values)
  }

  return (
    <div style={{ maxWidth: 640 }}>
      <Typography.Title level={3}>{isEdit ? '编辑 Flag' : '新建 Flag'}</Typography.Title>
      <Alert
        type="info"
        showIcon
        style={{ marginBottom: 16 }}
        message="Key 与环境创建后不可修改；同一环境下 Key 唯一（数据库约束保证）。"
      />
      {detailQuery.error ? (
        <Alert type="error" message={(detailQuery.error as Error).message} style={{ marginBottom: 16 }} />
      ) : null}
      <Form
        form={form}
        layout="vertical"
        initialValues={{ enabled: true, defaultValue: false, environment: 'development' }}
        onFinish={onFinish}
        disabled={isEdit && detailQuery.isLoading}
      >
        <Form.Item name="name" label="名称" rules={[{ required: true, max: 100 }]}>
          <Input />
        </Form.Item>
        <Form.Item
          name="key"
          label="Key"
          rules={[
            { required: true },
            { pattern: /^[a-z][a-z0-9_.:-]{0,63}$/, message: '需匹配 ^[a-z][a-z0-9_.:-]{0,63}$' },
          ]}
        >
          <Input disabled={isEdit} />
        </Form.Item>
        <Form.Item name="environment" label="环境" rules={[{ required: true }]}>
          <Select
            disabled={isEdit}
            options={[
              { value: 'development', label: 'development' },
              { value: 'staging', label: 'staging' },
              { value: 'production', label: 'production' },
            ]}
          />
        </Form.Item>
        {!isEdit ? (
          <Form.Item name="enabled" label="启用" valuePropName="checked">
            <Switch />
          </Form.Item>
        ) : null}
        <Form.Item name="defaultValue" label="默认返回值" rules={[{ required: true }]}>
          <Radio.Group
            options={[
              { value: true, label: 'true' },
              { value: false, label: 'false' },
            ]}
          />
        </Form.Item>
        <Space>
          <Button type="primary" htmlType="submit" loading={createMut.isPending || updateMut.isPending}>
            保存
          </Button>
          <Button onClick={() => nav(-1)}>取消</Button>
        </Space>
      </Form>
    </div>
  )
}
