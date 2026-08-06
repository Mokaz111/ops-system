import { create } from 'zustand';
import { workspaceAPI } from '../api/workspace';
import { isPlatformAdmin } from '../utils/membership';
import type { User } from '../types/api';

export interface WorkspaceOption {
  id: string;
  name: string;
}

const STORAGE_KEY = 'currentWorkspaceId';

interface WorkspaceState {
  options: WorkspaceOption[];
  /** 当前选中的工作空间；空字符串表示"全部"（仅平台管理员可用） */
  currentId: string;
  loaded: boolean;
  init: (user: User | null) => Promise<void>;
  setCurrent: (id: string) => void;
  reset: () => void;
}

/**
 * 全局工作空间上下文：
 * - 平台管理员：拉取全量工作空间列表，可选"全部"或具体空间；
 * - 普通用户：由 memberships 派生，默认选中第一个空间。
 * 各页面（概览 / 告警 / 日志等）消费 currentId 作为默认过滤。
 */
export const useWorkspaceStore = create<WorkspaceState>((set, get) => ({
  options: [],
  currentId: localStorage.getItem(STORAGE_KEY) || '',
  loaded: false,

  init: async (user) => {
    if (!user) return;
    let options: WorkspaceOption[] = [];
    if (isPlatformAdmin(user)) {
      try {
        const { data: res } = await workspaceAPI.list({ page: 1, page_size: 200 });
        options = (res.data?.items || []).map((w) => ({ id: w.id, name: w.workspace_name }));
      } catch {
        options = [];
      }
    } else {
      options = (user.memberships || []).map((m) => ({ id: m.workspace_id, name: m.workspace_name }));
    }
    let currentId = get().currentId;
    if (currentId && !options.some((o) => o.id === currentId)) currentId = '';
    // 普通用户没有"全部"视图，默认锁定第一个空间。
    if (!currentId && !isPlatformAdmin(user) && options.length > 0) currentId = options[0].id;
    set({ options, currentId, loaded: true });
    if (currentId) localStorage.setItem(STORAGE_KEY, currentId);
  },

  setCurrent: (id) => {
    set({ currentId: id });
    if (id) localStorage.setItem(STORAGE_KEY, id);
    else localStorage.removeItem(STORAGE_KEY);
  },

  reset: () => {
    localStorage.removeItem(STORAGE_KEY);
    set({ options: [], currentId: '', loaded: false });
  },
}));
