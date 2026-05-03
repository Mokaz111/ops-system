import type { InstanceSpec } from '../types/api';

export function parseSpec(spec: string): InstanceSpec {
  try {
    return JSON.parse(spec);
  } catch {
    return { cpu: 0, memory: 0, storage: 0, retention: 0 };
  }
}
