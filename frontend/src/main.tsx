import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import './index.css'
import App from './App.tsx'
import ToastProvider from './components/ToastProvider'
import ConfirmModal from './components/ConfirmModal'
import { installCxSecInterceptor } from './lib/cxsec/interceptor'
import { fetchCxSecConfig } from './lib/cxsec/cxsec'

function Provider({ children }: { children: React.ReactNode }) {
  return (
    <>
      {children}
      <ToastProvider />
      <ConfirmModal />
    </>
  )
}

// ～先拉服务端配置，再决定守护哪些接口，默认守护注册和签到 API～
fetchCxSecConfig().then(config => {
  if (config.enabled) {
    installCxSecInterceptor(config.paths)
  }
})

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <BrowserRouter>
      <Provider>
        <App />
      </Provider>
    </BrowserRouter>
  </StrictMode>,
)
