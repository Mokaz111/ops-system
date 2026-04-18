import axios, { AxiosError } from 'axios';

const api = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '/api/v1',
  timeout: 15000,
  headers: { 'Content-Type': 'application/json' },
});

// token 仍从 localStorage 读取（store 初始化时也是从这里 rehydrate），
// 这样保证拦截器对所有请求都拿到最新值，避免 hook 闭包过期。
// 写路径已收敛到 useAuthStore（login/logout/fetchMe），不再有"绕过 store 直接 setItem"的情况。
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// UNAUTHORIZED_EVENT 由 App 层订阅，统一做 logout + 路由跳转 + 记住来源路径。
export const UNAUTHORIZED_EVENT = 'ops:unauthorized';

// extractApiError 把任意错误（首选 AxiosError）映射成可直接给 toast 的中文文案。
//
// 优先级：
//   1. 后端返回体里的 message / error 字段（业务语义最准确）；
//   2. 按 HTTP 状态码兜底（认证、权限、冲突…）；
//   3. 网络异常；
//   4. 退化到 fallback。
//
// 使用方式：catch (err) { enqueueSnackbar(extractApiError(err, '伸缩失败'), ...) }
export function extractApiError(err: unknown, fallback = '请求失败'): string {
  if (axios.isAxiosError(err)) {
    const data = err.response?.data as { message?: string; error?: string } | undefined;
    const backendMsg = data?.message || data?.error;
    if (backendMsg && typeof backendMsg === 'string') return backendMsg;
    const status = err.response?.status;
    if (status === 401) return '未登录或会话已过期';
    if (status === 403) return '当前账号无权限执行该操作';
    if (status === 404) return '资源不存在或已被删除';
    if (status === 409) return '操作冲突，请刷新后重试';
    if (status === 422) return '参数校验失败';
    if (status === 429) return '请求过于频繁，请稍后重试';
    if (status === 500) return '服务端异常，请稍后重试';
    if (!err.response) return '网络异常，无法连接后端';
  }
  if (err instanceof Error && err.message) return err.message;
  return fallback;
}

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
