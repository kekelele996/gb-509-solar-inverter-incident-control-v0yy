import { useEffect, useState } from 'react';
import type { DomainRecord } from '../../types/domain';
import { UiButton } from './UiButton';

export function ActionDrawer({ open, item, target, onCancel, onConfirm }: { open: boolean; item: DomainRecord | null; target: string; onCancel: () => void; onConfirm: () => void }) {
  const [checked, setChecked] = useState(false);
  useEffect(() => { if (open) setChecked(false); }, [open, item?.id, target]);
  if (!open || !item) return null;
  return <div className="drawer-backdrop" role="presentation"><aside className="action-drawer" role="dialog" aria-modal="true" aria-label="远程动作二次确认"><header><span>REMOTE CONTROL</span><h2>远程动作二次确认</h2></header><dl><div><dt>对象</dt><dd>{item.code} · {item.name}</dd></div><div><dt>状态变化</dt><dd>{item.status} → {target}</dd></div><div><dt>影响区域</dt><dd>{item.facility}</dd></div></dl><label className="confirm-check"><input type="checkbox" checked={checked} onChange={(event) => setChecked(event.target.checked)} />我已核对设备、影响范围和回退方案</label><footer><button className="link-button" onClick={onCancel}>返回复核</button><UiButton danger disabled={!checked} onClick={onConfirm}>执行远程动作</UiButton></footer></aside></div>;
}
