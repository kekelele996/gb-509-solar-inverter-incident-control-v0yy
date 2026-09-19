
import type { DomainRecord } from '../../types/domain';
import { StatusBadge } from './StatusBadge';
export function SeverityTag({ records }: { records: DomainRecord[] }) {
  if (!records.length) return <div className="empty">暂无可展示的业务证据</div>;
  return <section className="severity-panel" aria-label="设备故障等级"><header><strong>设备故障等级</strong><span>{records.filter((item) => ['high', 'critical'].includes(item.riskLevel)).length} 项高风险</span></header><div className="evidence-strip">{records.slice(0, 4).map((item) => <article key={item.id}><strong>{item.code}</strong><span>{item.name}</span><em className={`severity severity--${item.riskLevel}`}>{item.riskLevel}</em><StatusBadge status={item.status} /></article>)}</div></section>;
}
