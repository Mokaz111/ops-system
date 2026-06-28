import type { ApiResponse } from '../types/api';

interface GrafanaSsoResponse {
  proxyUrl: string;
}

// ssoLoginToGrafana 先同步打开空白窗口再 await 登录接口，使 window.open 处于
// 用户手势调用栈内，避免被浏览器以"非用户手势"拦截。
// 若弹窗仍被拦截，则抛出含代理 URL 的错误，供调用方提示用户手动跳转。
export async function ssoLoginToGrafana(
  loginPromise: Promise<{ data: ApiResponse<GrafanaSsoResponse> }>,
) {
  const popup = window.open('about:blank', '_blank');
  if (!popup) {
    const { data: apiRes } = await loginPromise;
    const url = apiRes.data.proxyUrl;
    const err = new Error(`Grafana 登录弹窗被浏览器拦截，请允许弹窗或手动访问：${url}`);
    (err as Error & { proxyUrl?: string }).proxyUrl = url;
    throw err;
  }
  const { data: apiRes } = await loginPromise;
  popup.location.href = apiRes.data.proxyUrl;
}

// ssoLoginAndRedirect 登录并跳转到指定的 Grafana 子路径（如 /d/uid/title）。
export async function ssoLoginAndRedirect(
  loginFn: (redirect: string) => Promise<{ data: ApiResponse<GrafanaSsoResponse> }>,
  redirect: string,
) {
  const popup = window.open('about:blank', '_blank');
  if (!popup) {
    const { data: apiRes } = await loginFn(redirect);
    const url = apiRes.data.proxyUrl;
    const err = new Error(`Grafana 登录弹窗被浏览器拦截，请允许弹窗或手动访问：${url}`);
    (err as Error & { proxyUrl?: string }).proxyUrl = url;
    throw err;
  }
  const { data: apiRes } = await loginFn(redirect);
  popup.location.href = apiRes.data.proxyUrl;
}
