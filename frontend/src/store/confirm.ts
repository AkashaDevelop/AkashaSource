import { create } from 'zustand';

interface ConfirmOptions {
  title?: string;
  message: string;
  confirmText?: string;
  cancelText?: string;
  danger?: boolean;
}

interface ConfirmStore {
  open: boolean;
  options: ConfirmOptions | null;
  resolve: ((ok: boolean) => void) | null;
  show: (options: ConfirmOptions) => Promise<boolean>;
  close: (ok: boolean) => void;
}

export const useConfirmStore = create<ConfirmStore>((set, get) => ({
  open: false,
  options: null,
  resolve: null,
  show: (options) =>
    new Promise<boolean>((resolve) => {
      set({ open: true, options, resolve });
    }),
  close: (ok) => {
    get().resolve?.(ok);
    set({ open: false, options: null, resolve: null });
  },
}));

// Call anywhere without hooks
export const confirm = (options: ConfirmOptions | string) =>
  useConfirmStore.getState().show(
    typeof options === 'string' ? { message: options } : options
  );
