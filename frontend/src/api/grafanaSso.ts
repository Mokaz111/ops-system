import type { ApiResponse } from '../types/api';

interface GrafanaLoginInfo {
  url: string;
  user: string;
  password: string;
}

export async function ssoLoginToGrafana(
  loginPromise: Promise<{ data: ApiResponse<GrafanaLoginInfo> }>,
) {
  const { data: apiRes } = await loginPromise;
  const { url, user, password } = apiRes.data;
  const form = document.createElement('form');
  form.method = 'POST';
  form.action = `${url.replace(/\/$/, '')}/login`;
  form.target = '_blank';
  form.style.display = 'none';
  const u = document.createElement('input'); u.name = 'user'; u.value = user;
  const p = document.createElement('input'); p.name = 'password'; p.value = password;
  form.append(u, p);
  document.body.append(form);
  form.submit();
  form.remove();
}
