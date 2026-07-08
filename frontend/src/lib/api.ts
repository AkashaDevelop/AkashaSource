// 统一 API 客户端：自动注入鉴权头、401 自动登出跳转
import { useAuthStore } from '../../store/auth';

const TOKEN_EXPIRED_CODES = [401];

interface ApiOptions extends RequestInit {
  skipAuth?: boolean; // 公开接口跳过鉴权头
}

// apiRequest 统一请求封装
export async function apiRequest(url: string, options: ApiOptions = {}): Promise<Response> {
  const { token, logout } = useAuthStore.getState();
  const headers: Record<string, string> = {
    ...(options.headers as Record<string, string>),
  };

  if (!options.skipAuth && token) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  // 有 body 且非 FormData 时自动设置 Content-Type
  if (options.body && !(options.body instanceof FormData) && !headers['Content-Type']) {
    headers['Content-Type'] = 'application/json';
  }

  const res = await fetch(url, { ...options, headers });

  // 401 → 自动登出跳转
  if (TOKEN_EXPIRED_CODES.includes(res.status)) {
    logout();
    // 避免在登录页重复跳转
    if (!window.location.pathname.startsWith('/login')) {
      window.location.href = '/login';
    }
  }

  return res;
}

// apiGet 便捷方法
export async function apiGet(url: string, options?: ApiOptions): Promise<any> {
  const res = await apiRequest(url, { ...options, method: 'GET' });
  return res.json();
}

// apiPost 便捷方法
export async function apiPost(url: string, body?: any, options?: ApiOptions): Promise<any> {
  const res = await apiRequest(url, {
    ...options,
    method: 'POST',
    body: body ? JSON.stringify(body) : undefined,
  });
  return res.json();
}

// apiPut 便捷方法
export async function apiPut(url: string, body?: any, options?: ApiOptions): Promise<any> {
  const res = await apiRequest(url, {
    ...options,
    method: 'PUT',
    body: body ? JSON.stringify(body) : undefined,
  });
  return res.json();
}

// apiDelete 便捷方法
export async function apiDelete(url: string, options?: ApiOptions): Promise<any> {
  const res = await apiRequest(url, { ...options, method: 'DELETE' });
  return res.json();
}

// apiFormData 用于文件上传
export async function apiFormData(url: string, formData: FormData, options?: ApiOptions): Promise<any> {
  const res = await apiRequest(url, {
    ...options,
    method: 'POST',
    body: formData,
  });
  return res.json();
}
