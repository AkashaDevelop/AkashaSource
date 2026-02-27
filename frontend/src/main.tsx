import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import './index.css'
import App from './App.tsx'
import ToastProvider from './components/ToastProvider'
import ConfirmModal from './components/ConfirmModal'

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
