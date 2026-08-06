import { useEffect, useState } from 'react';
import { FormControl, InputLabel, MenuItem, Select } from '@mui/material';
import { workspaceAPI } from '../../api/workspace';
import type { Workspace } from '../../types/api';

export const levelMeta: Record<string, { label: string; color: 'error' | 'warning' | 'info' | 'default' }> = {
  critical: { label: '严重', color: 'error' },
  warning: { label: '警告', color: 'warning' },
  info: { label: '信息', color: 'info' },
};

export const eventStatusMeta: Record<string, { label: string; color: 'error' | 'warning' | 'success' | 'default' }> = {
  firing: { label: '告警中', color: 'error' },
  acknowledged: { label: '已确认', color: 'warning' },
  resolved: { label: '已恢复', color: 'success' },
};

export const channelTypeLabels: Record<string, string> = {
  dingtalk: '钉钉',
  email: '邮件',
  slack: 'Slack',
  sms: '短信',
  webhook: 'Webhook',
};

export function parseChannelIds(raw: string): string[] {
  if (!raw) return [];
  try {
    const parsed = JSON.parse(raw);
    return Array.isArray(parsed) ? parsed.map(String) : [];
  } catch {
    return raw.split(',').map((s) => s.trim()).filter(Boolean);
  }
}

/** 拉取工作空间列表，供筛选与 tenant_id → 名称映射。 */
export function useWorkspaceOptions() {
  const [workspaces, setWorkspaces] = useState<Workspace[]>([]);
  useEffect(() => {
    let alive = true;
    (async () => {
      try {
        const { data: res } = await workspaceAPI.list({ page: 1, page_size: 200 });
        if (alive) setWorkspaces(res.data?.items || []);
      } catch {
        /* optional */
      }
    })();
    return () => {
      alive = false;
    };
  }, []);
  const tenantName = (id: string) => workspaces.find((t) => t.id === id)?.workspace_name || id.slice(0, 8);
  return { workspaces, tenantName };
}

interface WorkspaceFilterSelectProps {
  value: string;
  onChange: (value: string) => void;
  workspaces: Workspace[];
}

export function WorkspaceFilterSelect({ value, onChange, workspaces }: WorkspaceFilterSelectProps) {
  return (
    <FormControl size="small" sx={{ minWidth: 220 }}>
      <InputLabel>工作空间</InputLabel>
      <Select value={value} label="工作空间" onChange={(e) => onChange(e.target.value)}>
        <MenuItem value="">全部工作空间</MenuItem>
        {workspaces.map((tenant) => (
          <MenuItem key={tenant.id} value={tenant.id}>{tenant.workspace_name}</MenuItem>
        ))}
      </Select>
    </FormControl>
  );
}
