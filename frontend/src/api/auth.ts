import api from './index';
import type { ApiResponse, LoginRequest, LoginResponse, User } from '../types/api';

export const authAPI = {
  login: (data: LoginRequest) =>
    api.post<ApiResponse<LoginResponse>>('/auth/login', data),

  me: () =>
    api.get<ApiResponse<User>>('/auth/me'),

  bootstrap: (data: { username: string; password: string; display_name: string }) =>
    api.post<ApiResponse<User>>('/users/bootstrap', data),
};
