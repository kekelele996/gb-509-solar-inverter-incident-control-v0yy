
import { request } from './client';
import type { DomainRecord } from '../types/domain';

export async function listFaultEvent(page = 1, pageSize = 20, search = '') {
  return request<DomainRecord[]>(`/faults?page=${page}&pageSize=${pageSize}&search=${encodeURIComponent(search)}`);
}
export async function createFaultEvent(input: Partial<DomainRecord>) {
  return request<DomainRecord>('/faults', { method: 'POST', body: JSON.stringify(input) });
}
export async function transitionFaultEvent(id: number, status: string, expectedVersion: number, reason: string) {
  return request<DomainRecord>(`/faults/${id}/transition`, {
    method: 'POST', body: JSON.stringify({ status, expectedVersion, reason }),
  });
}
