import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { ConfigProvider } from 'antd'
import zhCN from 'antd/locale/zh_CN'
import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'
import AppLayout from './AppLayout'
import EvaluatePage from './pages/EvaluatePage'
import FlagDetailPage from './pages/FlagDetailPage'
import FlagListPage from './pages/FlagListPage'
import './index.css'

const qc = new QueryClient({
  defaultOptions: {
    queries: { retry: false, refetchOnWindowFocus: false },
  },
})

export default function App() {
  return (
    <ConfigProvider locale={zhCN}>
      <QueryClientProvider client={qc}>
        <BrowserRouter>
          <Routes>
            <Route element={<AppLayout />}>
              <Route path="/" element={<FlagListPage />} />
              <Route path="/flags/:id" element={<FlagDetailPage />} />
              <Route path="/evaluate" element={<EvaluatePage />} />
              <Route path="*" element={<Navigate to="/" replace />} />
            </Route>
          </Routes>
        </BrowserRouter>
      </QueryClientProvider>
    </ConfigProvider>
  )
}
