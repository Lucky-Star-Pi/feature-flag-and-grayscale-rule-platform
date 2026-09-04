import { Alert, Typography } from 'antd'

export default function FlagListPage() {
  return (
    <div>
      <Typography.Title level={3}>Flag 列表（占位）</Typography.Title>
      <Alert
        type="info"
        showIcon
        message="M1 仅脚手架。列表、搜索、新建将在 M2 对接真实 Go API。"
      />
    </div>
  )
}
