import type { ApiResponse } from '../types/api';

interface GrafanaSsoResponse {
  proxyUrl: string;
}

export async function ssoLoginToGrafana(
  loginPromise: Promise<{ data: ApiResponse<GrafanaSsoResponse> }>,
) {
  const { data: apiRes } = await loginPromise;
  const { proxyUrl } = apiRes.data;
  window.open(proxyUrl, '_blank');
}
