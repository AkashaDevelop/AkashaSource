import { create } from 'zustand';

export type ToastType = 'success' | 'error' | 'warning' | 'info';

export interface ToastItem {
  id: string;
  type: ToastType;
  title?: string;
  message: string;
  duration?: number;
}

interface ToastStore {
  toasts: ToastItem[];
  add: (toast: Omit<ToastItem, 'id'>) => void;
  remove: (id: string) => void;
}

export const useToastStore = create<ToastStore>((set) => ({
  toasts: [],
  add: (toast) => {
    const id = Math.random().toString(36).slice(2);
    set((s) => ({ toasts: [...s.toasts, { ...toast, id }] }));
    const duration = toast.duration ?? 3500;
    if (duration > 0) setTimeout(() => set((s) => ({ toasts: s.toasts.filter(t => t.id !== id) })), duration);
  },
  remove: (id) => set((s) => ({ toasts: s.toasts.filter(t => t.id !== id) })),
}));

// Convenience helpers — call anywhere without hooks
export const toast = {
  success: (message: string, title?: string) =>
    useToastStore.getState().add({ type: 'success', message, title }),
  error: (message: string, title?: string) =>
    useToastStore.getState().add({ type: 'error', message, title, duration: 8000 }),
  warning: (message: string, title?: string) =>
    useToastStore.getState().add({ type: 'warning', message, title }),
  info: (message: string, title?: string) =>
    useToastStore.getState().add({ type: 'info', message, title }),
};
