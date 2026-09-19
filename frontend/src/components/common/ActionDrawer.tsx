import { useEffect, useState } from 'react';
import type { ActionInterlock, DomainRecord } from '../../types/domain';
import { StatusBadge } from './StatusBadge';
import { UiButton } from './UiButton';

type ActionDrawerProps = {
  open: boolean;
  item: DomainRecord | null;
  target: string;
  interlock?: ActionInterlock | null;
  interlockLoading?: boolean;
  onCancel: () => void;
  onConfirm: () => void;
};

export function ActionDrawer({ open, item, target, interlock, interlockLoading = false, onCancel, onConfirm }: ActionDrawerProps) {
  const [checked, setChecked] = useState(false);
  useEffect(() => { if (open) setChecked(false); }, [open, item?.id, target]);
  if (!open || !item) return null;
  const isInterlockAction = target === 'executing';
  const hasInterlock = Boolean(isInterlockAction && interlock && interlock.action.id === item.id);
  const blocked = isInterlockAction && (interlockLoading || Boolean(interlock?.failureReason) || !interlock?.canConfirm);
  return <div className="drawer-backdrop" role="presentation"><aside className="action-drawer" role="dialog" aria-modal="true" aria-label="远程动作二次确认"><header><span>REMOTE CONTROL</span><h2>远程动作二次确认</h2></header><dl><div><dt>对象</dt><dd>{item.code} · {item.name}</dd></div><div><dt>状态变化</dt><dd>{item.status} → {target}</dd></div><div><dt>影响区域</dt><dd>{item.facility}</dd></div></dl>
    {isInterlockAction && !hasInterlock && <section className="interlock-panel" aria-live="polite"><p className="muted">正在校验故障联锁状态…</p></section>}
    {hasInterlock && <section className="interlock-panel" aria-live="polite"><h3>故障联锁对象</h3><div><dt>关联编号</dt><dd>{item.relatedCode || '-'}</dd></div>{interlock?.fault ? <div><dt>故障事件</dt><dd>{interlock.fault.code} · {interlock.fault.name} <StatusBadge status={interlock.fault.status} /></dd></div> : <div><dt>故障事件</dt><dd>未找到关联故障</dd></div>}{interlock?.inverter ? <div><dt>逆变器</dt><dd>{interlock.inverter.code} · {interlock.inverter.name} <StatusBadge status={interlock.inverter.status} /></dd></div> : <div><dt>逆变器</dt><dd>未找到关联逆变器</dd></div>}{interlockLoading && <p className="muted">正在校验联锁状态…</p>}{interlock?.failureReason && <p className="interlock-error" role="alert">{interlock.failureReason}</p>}</section>}
    <label className="confirm-check"><input type="checkbox" checked={checked} disabled={isInterlockAction && blocked} onChange={(event) => setChecked(event.target.checked)} />我已核对设备、影响范围和回退方案</label><footer><button className="link-button" onClick={onCancel}>返回复核</button><UiButton danger disabled={!checked || (isInterlockAction && blocked)} onClick={onConfirm}>执行远程动作</UiButton></footer></aside></div>;
}
