import { useEffect } from 'react';
import { FormControl, MenuItem, Select, Typography } from '@mui/material';
import GroupsOutlinedIcon from '@mui/icons-material/GroupsOutlined';
import { useAuthStore } from '../../stores/useAuthStore';
import { useWorkspaceStore } from '../../stores/useWorkspaceStore';
import { isPlatformAdmin } from '../../utils/membership';

/**
 * TopBar 全局工作空间切换器。
 * 平台管理员可选"全部工作空间"；普通用户在自己的 memberships 之间切换。
 */
export default function WorkspaceSwitcher() {
  const user = useAuthStore((s) => s.user);
  const { options, currentId, loaded, init, setCurrent } = useWorkspaceStore();

  useEffect(() => {
    if (user && !loaded) init(user);
  }, [user, loaded, init]);

  if (!user || options.length === 0) return null;

  const admin = isPlatformAdmin(user);

  return (
    <FormControl size="small" sx={{ minWidth: 200, mr: 1.5 }}>
      <Select
        value={currentId}
        displayEmpty
        onChange={(e) => setCurrent(e.target.value)}
        renderValue={(v) => {
          const label = v ? options.find((o) => o.id === v)?.name || v : '全部工作空间';
          return (
            <Typography variant="body2" sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
              <GroupsOutlinedIcon sx={{ fontSize: 18, color: 'text.secondary' }} />
              {label}
            </Typography>
          );
        }}
        sx={{ '& .MuiSelect-select': { py: 0.75 } }}
      >
        {admin && <MenuItem value="">全部工作空间</MenuItem>}
        {options.map((o) => (
          <MenuItem key={o.id} value={o.id}>
            {o.name}
          </MenuItem>
        ))}
      </Select>
    </FormControl>
  );
}
