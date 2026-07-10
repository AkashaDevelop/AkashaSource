import { create } from 'zustand';
import { initQuotaConfig } from '../lib/quota';

interface UpdateInfo {
  has_update: boolean;
  latest_version: string;
  current_version: string;
  release_url: string;
  changelog_summary: string;
  force_update: boolean;
  last_checked: string;
}

interface SystemState {
  systemName: string;
  logoUrl: string;
  notice: string;
  chatLink: string;
  chatLink2: string;
  footerHtml: string;
  version: string;
  updateInfo: UpdateInfo | null;
  loaded: boolean;
  fetch: () => Promise<void>;
  checkUpdate: () => Promise<UpdateInfo | null>;
}

export const useSystemStore = create<SystemState>((set, get) => ({
  systemName: 'Akasha',
  logoUrl: '',
  notice: '',
  chatLink: '',
  chatLink2: '',
  footerHtml: '',
  version: '',
  updateInfo: null,
  loaded: false,
  fetch: async () => {
    if (get().loaded) return;
    try {
      const res = await fetch('/api/system/status');
      const data = await res.json();
      const payload = data.code === 0 ? data.data : data;
      const opts = payload?.options ?? {};
      initQuotaConfig(opts);
      set({
        systemName: opts.system_name || 'Akasha',
        logoUrl: opts.logo_url || '',
        notice: opts.notice || '',
        chatLink: opts.chat_link || '',
        chatLink2: opts.chat_link2 || '',
        footerHtml: opts.footer_html || '',
        version: payload?.version || '',
        updateInfo: payload?.update_info ?? null,
        loaded: true,
      });
    } catch { /* ignore */ }
  },
  checkUpdate: async () => {
    try {
      const token = localStorage.getItem('auth-storage');
      const authToken = token ? JSON.parse(token)?.state?.token : '';
      const res = await fetch('/api/system/update-check', {
        method: 'POST',
        headers: { Authorization: `Bearer ${authToken}` },
      });
      const data = await res.json();
      if (data.code === 0) {
        const info: UpdateInfo = data.data;
        set({ updateInfo: info });
        return info;
      }
      return null;
    } catch { return null; }
  },
}));
