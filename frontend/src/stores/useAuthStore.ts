import { create } from 'zustand';
import axios from 'axios';
import { authAPI } from '../api/auth';
import { extractApiError, UNAUTHORIZED_EVENT } from '../api';
import type { User } from '../types/api';

// extractLoginError 在通用 extractApiError 之上，针对登录场景把 401/403 翻译为
// "用户名或密码错误"——后端只会返回模糊的 "invalid credentials"，对终端用户不直观。
function extractLoginError(err: unknown): string {
  if (axios.isAxiosError(err)) {
    const status = err.response?.status;
    if (status === 401 || status === 403) {
      const data = err.response?.data as { message?: string; error?: string } | undefined;
      const backendMsg = data?.message || data?.error;
      return backendMsg && typeof backendMsg === 'string' ? backendMsg : '用户名或密码错误';
    }
  }
  return extractApiError(err, '登录失败，请稍后重试');
}

interface AuthState {
  user: User | null;
  token: string | null;
  isAuthenticated: boolean;
  loading: boolean;
  login: (username: string, password: string) => Promise<void>;
  logout: () => void;
  fetchMe: () => Promise<void>;
  setUser: (user: User) => void;
}

export const useAuthStore = create<AuthState>((set) => ({
  user: JSON.parse(localStorage.getItem('user') || 'null'),
  token: localStorage.getItem('token'),
  isAuthenticated: !!localStorage.getItem('token'),
  loading: false,

  login: async (username, password) => {
    set({ loading: true });
    try {
      const { data: res } = await authAPI.login({ username, password });
      const { token, user } = res.data;
      localStorage.setItem('token', token);
      localStorage.setItem('user', JSON.stringify(user));
      set({ user, token, isAuthenticated: true, loading: false });
    } catch (err) {
      set({ loading: false });
      throw new Error(extractLoginError(err));
    }
  },

  logout: () => {
    localStorage.removeItem('token');
    localStorage.removeItem('user');
    set({ user: null, token: null, isAuthenticated: false });
  },

  fetchMe: async () => {
    try {
      const { data: res } = await authAPI.me();
      const user = res.data;
      localStorage.setItem('user', JSON.stringify(user));
      set({ user });
    } catch {
      localStorage.removeItem('token');
      localStorage.removeItem('user');
      set({ user: null, token: null, isAuthenticated: false });
    }
  },

  setUser: (user) => {
    localStorage.setItem('user', JSON.stringify(user));
    set({ user });
  },
}));

// 拦截器在 401 时只动了 localStorage 并广播 UNAUTHORIZED_EVENT；
// 这里订阅事件把 store 状态同步清掉，AuthGuard 才能感知 token 失效并跳登录。
if (typeof window !== 'undefined') {
  window.addEventListener(UNAUTHORIZED_EVENT, () => {
    useAuthStore.setState({ user: null, token: null, isAuthenticated: false });
  });
}
