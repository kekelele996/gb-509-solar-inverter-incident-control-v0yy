import type { EntityConfig } from './domain';

export type InverterState = 'online' | 'warning' | 'tripped' | 'isolated';
export const ALL_INVERTER_STATE: readonly InverterState[] = ['online', 'warning', 'tripped', 'isolated'];
export type FaultState = 'open' | 'acknowledged' | 'mitigated' | 'closed';
export const ALL_FAULT_STATE: readonly FaultState[] = ['open', 'acknowledged', 'mitigated', 'closed'];

export const ENTITY_CONFIGS: readonly EntityConfig[] = [
  { key: 'solarSite', path: 'sites', label: '光伏场站', statuses: ['online', 'limited', 'offline', 'maintenance'] as const },
  { key: 'inverterUnit', path: 'inverters', label: '逆变器', statuses: ['online', 'warning', 'tripped', 'isolated'] as const },
  { key: 'faultEvent', path: 'faults', label: '故障事件', statuses: ['open', 'acknowledged', 'mitigated', 'closed'] as const },
  { key: 'mitigationAction', path: 'actions', label: '处置动作', statuses: ['draft', 'confirmed', 'executing', 'completed', 'failed'] as const }
];
