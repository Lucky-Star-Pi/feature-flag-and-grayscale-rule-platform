import { Layout, Menu, Typography } from 'antd'
import { Link, Outlet, useLocation } from 'react-router-dom'

const { Header, Content } = Layout

const items = [
  { key: '/', label: <Link to="/">Flag 列表</Link> },
  { key: '/flags/1', label: <Link to="/flags/1">Flag 详情（占位）</Link> },
  { key: '/evaluate', label: <Link to="/evaluate">评估控制台</Link> },
]

export default function AppLayout() {
  const loc = useLocation()
  const selected = loc.pathname.startsWith('/evaluate')
    ? '/evaluate'
    : loc.pathname.startsWith('/flags/')
      ? '/flags/1'
      : '/'

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Header style={{ display: 'flex', alignItems: 'center', gap: 24 }}>
        <Typography.Title level={4} style={{ color: '#fff', margin: 0 }}>
          Feature Flag 平台（M1）
        </Typography.Title>
        <Menu theme="dark" mode="horizontal" selectedKeys={[selected]} items={items} style={{ flex: 1 }} />
        <Typography.Text style={{ color: '#ddd' }}>操作者：local-admin</Typography.Text>
      </Header>
      <Content style={{ padding: 24 }}>
        <div style={{ background: '#fff', padding: 24, borderRadius: 8 }}>
          <Outlet />
        </div>
      </Content>
    </Layout>
  )
}
