import { Alert, Typography } from 'antd'

export default function EvaluatePage() {
  return (
    <div>
      <Typography.Title level={3}>评估控制台（占位）</Typography.Title>
      <Alert
        type="info"
        showIcon
        message="M1 不实现评估。后续阶段将调用 POST /api/v1/evaluate。"
      />
    </div>
  )
}
