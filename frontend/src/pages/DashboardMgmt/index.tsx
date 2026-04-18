import { useEffect, useState } from 'react';
import {
  Alert,
  Box,
  Card,
  CardContent,
  Chip,
  Grid,
  Typography,
} from '@mui/material';
import PageHeader from '../../components/common/PageHeader';
import EmptyState from '../../components/common/EmptyState';
import LoadingScreen from '../../components/common/LoadingScreen';
import { integrationAPI, type IntegrationInstallation } from '../../api/integration';

export default function DashboardMgmtPage() {
  const [loading, setLoading] = useState(true);
  const [installations, setInstallations] = useState<IntegrationInstallation[]>([]);

  useEffect(() => {
    let alive = true;
    (async () => {
      try {
        const { data: res } = await integrationAPI.listInstallations({ page: 1, page_size: 50 });
        if (!alive) return;
        setInstallations(res.data?.items || []);
      } finally {
        if (alive) setLoading(false);
      }
    })();
    return () => {
      alive = false;
    };
  }, []);

  if (loading) return <LoadingScreen />;

  const withDashboard = installations.filter((i) => (i.installed_parts || '').includes('dashboard'));

  return (
    <Box>
      <PageHeader
        title="Dashboard 管理"
        subtitle="平台托管 Dashboard 的安装记录（按 Grafana 主机 / Org 维度）"
      />

      <Alert severity="info" sx={{ mb: 2 }}>
        M1 占位：展示已登记的、包含 dashboard 部件的安装记录。M5 会补齐模版市场与一键安装/卸载到指定 Grafana Org 的能力。
      </Alert>

      {withDashboard.length === 0 ? (
        <EmptyState title="暂无已安装 Dashboard" description="通过接入中心安装模版并勾选 Dashboard 部件后，会出现在此处。" />
      ) : (
        <Grid container spacing={2}>
          {withDashboard.map((i) => (
            <Grid key={i.id} size={{ xs: 12, sm: 6, md: 4 }}>
              <Card>
                <CardContent>
                  <Typography variant="subtitle1" sx={{ fontWeight: 600 }}>
                    模版 {i.template_id.slice(0, 8)} · {i.template_version}
                  </Typography>
                  <Typography variant="caption" color="text.secondary">
                    实例 {i.instance_id.slice(0, 8)} · Grafana Org {i.grafana_org_id || '-'}
                  </Typography>
                  <Box sx={{ mt: 1 }}>
                    <Chip size="small" label={i.status} color={i.status === 'success' ? 'success' : 'default'} />
                  </Box>
                </CardContent>
              </Card>
            </Grid>
          ))}
        </Grid>
      )}
    </Box>
  );
}
