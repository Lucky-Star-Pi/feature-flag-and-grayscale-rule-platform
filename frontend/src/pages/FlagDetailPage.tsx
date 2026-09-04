import { Alert, Typography } from 'antd'
import { useParams } from 'react-router-dom'

export default function FlagDetailPage() {
  const { id } = useParams()
  return (
    <div>
      <Typography.Title level={3}>Flag 详情 / 规则（占位）</Typography.Title>
      <Alert
        type="info"
        showIcon
        message={`路由参数 id=${id ?? '-'}。规则管理与历史展示将在后续里程碑实现。`}
      />
    </div>
  )
}
