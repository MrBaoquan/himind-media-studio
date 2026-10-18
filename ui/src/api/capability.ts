import {mockCapability} from './mock';

type Invoke = (cmd: string, args?: unknown) => Promise<unknown>;

// 宿主注入两种可能的桥（不同 Tauri 版本）。都没有时进入浏览器 mock 模式。
const host: Invoke | undefined =
  (window as any).__TAURI__?.core?.invoke ?? (window as any).__TAURI_INTERNALS__?.invoke;

export const hosted = Boolean(host);

/** 调用插件能力。宿主模式下走内核的审批通道。 */
export async function callCapability<T>(id: string, input: Record<string, unknown>): Promise<T> {
  if (host) {
    return (await host('invoke_plugin_view_capability', {capabilityId: id, input})) as T;
  }
  return mockCapability<T>(id, input);
}

/** 调用宿主内置命令（非能力通道，不经审批）。 */
export async function hostInvoke<T>(cmd: string, args?: unknown): Promise<T> {
  if (!host) throw new Error(`宿主不可用：${cmd}`);
  return (await host(cmd, args)) as T;
}
