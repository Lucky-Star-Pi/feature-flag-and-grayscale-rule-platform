import { useQuery } from '@tanstack/react-query'
import { Alert, Button, Empty, Input, Select, Space, Table, Tag, Typography } from 'antd'
import { Link, useNavigate } from 'react-router-dom'
import { useState } from 'react'
import { api, type Flag } from '../api'

export default function FlagListPage() {
  const [q, setQ] = useState('')
  const [environment, setEnvironment] = useState<string | undefined>()
  const [enabled, setEnabled] = useState<string | undefined>()
  const nav = useNavigate()

  const { data, isLoading, error, refetch, isFetching } = useQuery({
    queryKey: ['flags', q, environment, enabled],
    queryFn: () => api.listFlags({ q, environment, enabled }),
  })

  const columns = [
    { title: '名称', dataIndex: 'name', key: 'name' },
    { title: 'Key', dataIndex: 'key', key: 'key', render: (v: string) => <code>{v}</code> },
    {
      title: '环境',
      dataIndex: 'environment',
      key: 'environment',
      render: (v: string) => <Tag>{v}</Tag>,
    },
    {
      title: '启用',
      dataIndex: 'enabled',
      key: 'enabled',
      render: (v: boolean) => (v ? <Tag color="green">启用</Tag> : <Tag color="red">停用</Tag>),
    },
    {
      title: '默认值',
      dataIndex: 'defaultValue',
      key: 'defaultValue',
      render: (v: boolean) => String(v),
    },
    {
      title: '更新时间',
      dataIndex: 'updatedAt',
      key: 'updatedAt',
      render: (v: string) => new Date(v).toLocaleString(),
    },
    {
      title: '操作',
      key: 'actions',
      render: (_: unknown, row: Flag) => (
        <Space>
          <Button type="link" onClick={() => nav(`/flags/${row.id}`)}>
            详情/规则
          </Button>
          <Button type="link" onClick={() => nav(`/flags/${row.id}/edit`)}>
            编辑
          </Button>
        </Space>
      ),
    },
  ]

  return (
    <div>
      <Space style={{ marginBottom: 16, width: '100%', justifyContent: 'space-between' }}>
        <Typography.Title level={3} style={{ margin: 0 }}>
          Flag 列表
        </Typography.Title>
        <Button type="primary">
          <Link to="/flags/new" style={{ color: 'inherit' }}>
            新建 Flag
          </Link>
        </Button>
      </Space>

      <Space wrap style={{ marginBottom: 16 }}>
        <Input.Search
          placeholder="搜索名称或 Key"
          allowClear
          onSearch={setQ}
          style={{ width: 240 }}
          loading={isFetching}
        />
        <Select
          allowClear
          placeholder="环境"
          style={{ width: 160 }}
          value={environment}
          onChange={setEnvironment}
          options={[
            { value: 'development', label: 'development' },
            { value: 'staging', label: 'staging' },
            { value: 'production', label: 'production' },
          ]}
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

      {error ? (
        <Alert
          type="error"
          showIcon
          style={{ marginBottom: 16 }}
          message={(error as Error).message || '加载失败'}
        />
      ) : null}

      <Table
        rowKey="id"
        loading={isLoading}
        columns={columns}
        dataSource={data?.items || []}
        locale={{ emptyText: <Empty description="暂无 Flag" /> }}
      />
    </div>
  )
}
