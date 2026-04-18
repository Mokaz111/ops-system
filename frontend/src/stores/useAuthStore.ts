import { create } from 'zustand';
import axios from 'axios';
import { authAPI } from '../api/auth';
import type { User } from '../types/api';

// extractLoginError 优先从后端 {message} 读错误，再退到 axios 状态码，最后给通用提示。
function extractLoginError(err: unknown): string {
  if (axios.isAxiosError(err)) {
    const data = err.response?.data as { message?: string; error?: string } | undefined;
    const backendMsg = data?.message || data?.error;
    if (backendMsg && typeof backendMsg === 'string') return backendMsg;
    const status = err.response?.status;
    if (status === 401 || status === 403) return '用户名或密码错误';
    if (status === 429) return '请求过于频繁，请稍后重试';
    if (!err.response) return '网络异常，无法连接后端';
  }
  if (err instanceof Error && err.message) return err.message;
  return '登录失败，请稍后重试';
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
