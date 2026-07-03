import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import './index.css'
import App from './App.tsx'
import ToastProvider from './components/ToastProvider'
import ConfirmModal from './components/ConfirmModal'
import { installCxSecInterceptor } from './lib/cxsec/interceptor'

// ～宸汐御安全：握手调通后取消注释启用，暂时关闭以保证 app 正常运行～
installCxSecInterceptor()

function Provider({ children }: { children: React.ReactNode }) {
  return (
    <>
      {children}
      <ToastProvider />
      <ConfirmModal />
    </>
  )
}

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <BrowserRouter>
      <Provider>
        <App />
      </Provider>
    </BrowserRouter>
  </StrictMode>,
)
