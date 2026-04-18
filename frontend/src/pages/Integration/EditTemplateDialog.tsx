import { useState, useEffect } from 'react';
import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControl,
  Grid,
  InputLabel,
  MenuItem,
  Select,
  TextField,
} from '@mui/material';
import { useSnackbar } from 'notistack';
import {
  integrationAPI,
  type IntegrationCategory,
  type IntegrationTemplate,
} from '../../api/integration';
import { extractApiError } from '../../api';

interface Props {
  open: boolean;
  template: IntegrationTemplate | null;
  categories: IntegrationCategory[];
  onClose: () => void;
  onSuccess: () => void;
}

function parseTags(raw: string): string[] {
  if (!raw) return [];
  try {
    const arr = JSON.parse(raw);
    if (Array.isArray(arr)) return arr.map(String);
  } catch {
    // JSON 以外直接按逗号分隔解析
  }
  return raw.split(',').map((s) => s.trim()).filter(Boolean);
}

export default function EditTemplateDialog({ open, template, categories, onClose, onSuccess }: Props) {
  const { enqueueSnackbar } = useSnackbar();
  const [displayName, setDisplayName] = useState('');
  const [category, setCategory] = useState('');
  const [component, setComponent] = useState('');
  const [description, setDescription] = useState('');
  const [icon, setIcon] = useState('');
  const [tags, setTags] = useState('');
  const [status, setStatus] = useState('active');
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    if (!open || !template) return;
    setDisplayName(template.display_name || '');
    setCategory(template.category || '');
    setComponent(template.component || '');
    setDescription(template.description || '');
    setIcon(template.icon || '');
    setTags(parseTags(template.tags || '').join(', '));
    setStatus(template.status || 'active');
  }, [open, template]);

  const submit = async () => {
    if (!template) return;
    setSubmitting(true);
    try {
      const tagsArr = tags.split(',').map((s) => s.trim()).filter(Boolean);
      await integrationAPI.updateTemplate(template.id, {
        display_name: displayName,
        category,
        component,
        description,
        icon,
        tags: tagsArr,
        status,
      });
      enqueueSnackbar('模板已更新', { variant: 'success' });
      onClose();
      onSuccess();
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '更新失败'), { variant: 'error' });
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Dialog open={open} onClose={() => !submitting && onClose()} maxWidth="sm" fullWidth>
      <DialogTitle>编辑模板 · {template?.name}</DialogTitle>
      <DialogContent sx={{ pt: '16px !important' }}>
        <Grid container spacing={2}>
          <Grid size={{ xs: 12 }}>
            <TextField
              fullWidth
              size="small"
              label="显示名"
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
            />
          </Grid>
          <Grid size={{ xs: 6 }}>
            <FormControl fullWidth size="small">
              <InputLabel>分类</InputLabel>
              <Select label="分类" value={category} onChange={(e) => setCategory(e.target.value)}>
                {categories.map((c) => (
                  <MenuItem key={c.key} value={c.key}>
                    {c.label}
                  </MenuItem>
                ))}
              </Select>
            </FormControl>
          </Grid>
          <Grid size={{ xs: 6 }}>
            <TextField
              fullWidth
              size="small"
              label="组件"
              value={component}
              onChange={(e) => setComponent(e.target.value)}
            />
          </Grid>
          <Grid size={{ xs: 12 }}>
            <TextField
              fullWidth
              size="small"
              multiline
              minRows={2}
              label="描述"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
            />
          </Grid>
          <Grid size={{ xs: 8 }}>
            <TextField
              fullWidth
              size="small"
              label="Icon URL"
              value={icon}
              onChange={(e) => setIcon(e.target.value)}
            />
          </Grid>
          <Grid size={{ xs: 4 }}>
            <FormControl fullWidth size="small">
              <InputLabel>状态</InputLabel>
              <Select label="状态" value={status} onChange={(e) => setStatus(e.target.value)}>
                <MenuItem value="active">active</MenuItem>
                <MenuItem value="inactive">inactive</MenuItem>
                <MenuItem value="deprecated">deprecated</MenuItem>
              </Select>
            </FormControl>
          </Grid>
          <Grid size={{ xs: 12 }}>
            <TextField
              fullWidth
              size="small"
              label="标签"
              value={tags}
              onChange={(e) => setTags(e.target.value)}
              helperText="以英文逗号分隔"
            />
          </Grid>
        </Grid>
      </DialogContent>
      <DialogActions sx={{ px: 3, pb: 2 }}>
        <Button onClick={onClose} disabled={submitting}>取消</Button>
        <Button variant="contained" onClick={submit} disabled={submitting}>
          {submitting ? '保存中...' : '保存'}
        </Button>
      </DialogActions>
    </Dialog>
  );
}
