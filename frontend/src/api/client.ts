import type { AxiosRequestConfig } from 'axios';
import axios from 'axios';

// withSignal 给业务调用方提供一个最小的可取消请求工具。
//
// axios 接受 AxiosRequestConfig.signal（AbortSignal），任何 baseURL 走我们 api 单例的
// 请求都能复用同一个签名习惯。把这一点单独提出来主要是为了给 useEffect cleanup 收尾时
// 调用 controller.abort()，避免 setState-on-unmounted 与重复请求的竞态。
export function makeAbortController(): AbortController {
  return new AbortController();
}

// isAbortError 用来在 catch 里早退；axios 取消的请求会抛出 CanceledError，不应再 toast。
export function isAbortError(err: unknown): boolean {
  return axios.isCancel(err) || (err instanceof Error && err.name === 'AbortError');
}

// 兼容旧调用：把 signal 注入 axios config 的小语法糖。
export function withSignal<T extends AxiosRequestConfig>(cfg: T, signal: AbortSignal): T {
  return { ...cfg, signal } as T;
}
