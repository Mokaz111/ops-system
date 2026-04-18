import axios, { AxiosError } from 'axios';

const api = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '/api/v1',
  timeout: 15000,
  headers: { 'Content-Type': 'application/json' },
});

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// UNAUTHORIZED_EVENT 由 App 层订阅，统一做 logout + 路由跳转 + 记住来源路径。
export const UNAUTHORIZED_EVENT = 'ops:unauthorized';

api.interceptors.response.use(
  (response) => response,
  (error: AxiosError) => {
    const status = error.response?.status;
    const url = error.config?.url ?? '';
    // /auth/login 自己会返回 401 以表明凭证错误，不走全局 logout；由登录页自己处理。
    if (status === 401 && !url.includes('/auth/login')) {
      localStorage.removeItem('token');
      localStorage.removeItem('user');
      if (typeof window !== 'undefined') {
        try {
          const { pathname, search } = window.location;
          if (pathname && !pathname.startsWith('/login')) {
            sessionStorage.setItem('ops:redirectAfterLogin', pathname + (search || ''));
          }
        } catch {
          // storage 不可用时静默降级
        }
        window.dispatchEvent(new CustomEvent(UNAUTHORIZED_EVENT));
      }
    }
    return Promise.reject(error);
  },
);

export default api;
