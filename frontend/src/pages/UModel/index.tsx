import { useCallback, useEffect, useState } from 'react';
import {
  Box, Button, Card, CardContent, Chip, Grid, MenuItem, Select, FormControl, InputLabel,
  Tab, Tabs, TextField, Typography,
} from '@mui/material';
import { useSnackbar } from 'notistack';
import { umodelAPI, type Entity, type MetricSet } from '../../api/umodel';
import { extractApiError } from '../../api';

export default function UModelPage() {
  const { enqueueSnackbar } = useSnackbar();
  const [tab, setTab] = useState(0);
  const [entities, setEntities] = useState<Entity[]>([]);
  const [metricSets, setMetricSets] = useState<MetricSet[]>([]);
  const [entityForm, setEntityForm] = useState({ entity_type: 'service', name: '', display_name: '' });
  const [metricSetForm, setMetricSetForm] = useState({ name: '', component: '', display_name: '' });

  const load = useCallback(async () => {
    const [eRes, mRes] = await Promise.allSettled([
      umodelAPI.listEntities({ page: 1, page_size: 100 }),
      umodelAPI.listMetricSets({ page: 1, page_size: 100 }),
    ]);
    if (eRes.status === 'fulfilled') setEntities(eRes.value.data.data?.items || []);
    if (mRes.status === 'fulfilled') setMetricSets(mRes.value.data.data?.items || []);
  }, []);

  useEffect(() => { load(); }, [load]);

  const createEntity = async () => {
    try {
      await umodelAPI.createEntity(entityForm);
      enqueueSnackbar('Entity 已创建', { variant: 'success' });
      setEntityForm({ entity_type: 'service', name: '', display_name: '' });
      load();
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '创建失败'), { variant: 'error' });
    }
  };

  const createMetricSet = async () => {
    try {
      await umodelAPI.createMetricSet(metricSetForm);
      enqueueSnackbar('MetricSet 已创建', { variant: 'success' });
      setMetricSetForm({ name: '', component: '', display_name: '' });
      load();
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '创建失败'), { variant: 'error' });
    }
  };

  return (
    <Box>
      <Typography variant="h5" sx={{ fontWeight: 600, mb: 1 }}>UModel 元数据</Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
        Entity / MetricSet 骨架 — 为后续日志、链路关联打地基
      </Typography>

      <Tabs value={tab} onChange={(_, v) => setTab(v)} sx={{ mb: 2 }}>
        <Tab label="Entity" />
        <Tab label="MetricSet" />
      </Tabs>

      {tab === 0 && (
        <Grid container spacing={2}>
          <Grid size={{ xs: 12, md: 4 }}>
            <Card>
              <CardContent>
                <Typography variant="subtitle2" sx={{ mb: 2 }}>新建 Entity</Typography>
                <FormControl fullWidth sx={{ mb: 2 }}>
                  <InputLabel>类型</InputLabel>
                  <Select value={entityForm.entity_type} label="类型" onChange={(e) => setEntityForm({ ...entityForm, entity_type: e.target.value })}>
                    <MenuItem value="service">service</MenuItem>
                    <MenuItem value="k8s_cluster">k8s_cluster</MenuItem>
                    <MenuItem value="namespace">namespace</MenuItem>
                    <MenuItem value="workload">workload</MenuItem>
                  </Select>
                </FormControl>
                <TextField fullWidth label="名称" sx={{ mb: 2 }} value={entityForm.name} onChange={(e) => setEntityForm({ ...entityForm, name: e.target.value })} />
                <TextField fullWidth label="显示名" sx={{ mb: 2 }} value={entityForm.display_name} onChange={(e) => setEntityForm({ ...entityForm, display_name: e.target.value })} />
                <Button variant="contained" onClick={createEntity} disabled={!entityForm.name.trim()}>创建</Button>
              </CardContent>
            </Card>
          </Grid>
          <Grid size={{ xs: 12, md: 8 }}>
            <Card>
              <CardContent>
                {entities.map((e) => (
                  <Box key={e.id} sx={{ display: 'flex', gap: 1, alignItems: 'center', py: 1, borderBottom: '1px solid', borderColor: 'divider' }}>
                    <Chip size="small" label={e.entity_type} />
                    <Typography sx={{ fontWeight: 500 }}>{e.display_name || e.name}</Typography>
                    <Typography variant="caption" color="text.secondary">{e.name}</Typography>
                  </Box>
                ))}
              </CardContent>
            </Card>
          </Grid>
        </Grid>
      )}

      {tab === 1 && (
        <Grid container spacing={2}>
          <Grid size={{ xs: 12, md: 4 }}>
            <Card>
              <CardContent>
                <Typography variant="subtitle2" sx={{ mb: 2 }}>新建 MetricSet</Typography>
                <TextField fullWidth label="名称" sx={{ mb: 2 }} value={metricSetForm.name} onChange={(e) => setMetricSetForm({ ...metricSetForm, name: e.target.value })} />
                <TextField fullWidth label="组件" sx={{ mb: 2 }} value={metricSetForm.component} onChange={(e) => setMetricSetForm({ ...metricSetForm, component: e.target.value })} />
                <Button variant="contained" onClick={createMetricSet} disabled={!metricSetForm.name.trim()}>创建</Button>
              </CardContent>
            </Card>
          </Grid>
          <Grid size={{ xs: 12, md: 8 }}>
            <Card>
              <CardContent>
                {metricSets.map((m) => (
                  <Box key={m.id} sx={{ display: 'flex', gap: 1, alignItems: 'center', py: 1, borderBottom: '1px solid', borderColor: 'divider' }}>
                    <Typography sx={{ fontWeight: 500 }}>{m.display_name || m.name}</Typography>
                    {m.component && <Chip size="small" label={m.component} variant="outlined" />}
                  </Box>
                ))}
              </CardContent>
            </Card>
          </Grid>
        </Grid>
      )}
    </Box>
  );
}
