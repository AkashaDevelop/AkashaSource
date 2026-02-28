import { create } from 'zustand';

interface SystemState {
  systemName: string;
  logoUrl: string;
  notice: string;
  chatLink: string;
  chatLink2: string;
  footerHtml: string;
  loaded: boolean;
  fetch: () => Promise<void>;
}

export const useSystemStore = create<SystemState>((set, get) => ({
  systemName: 'Akasha',
  logoUrl: '',
  notice: '',
  chatLink: '',
  chatLink2: '',
  footerHtml: '',
  loaded: false,
  fetch: async () => {
    if (get().loaded) return;
    try {
      const res = await fetch('/api/system/status');
      const data = await res.json();
      const opts = (data.code === 0 ? data.data : data)?.options ?? {};
      set({
        systemName: opts.system_name || 'Akasha',
        logoUrl: opts.logo_url || '',
        notice: opts.notice || '',
        chatLink: opts.chat_link || '',
        chatLink2: opts.chat_link2 || '',
        footerHtml: opts.footer_html || '',
        loaded: true,
      });
    } catch { /* ignore */ }
  },
}));
