
import { request } from './client';
import type { DomainRecord } from '../types/domain';

export async function listMitigationAction(page = 1, pageSize = 20, search = '') {
  return request<DomainRecord[]>(`/actions?page=${page}&pageSize=${pageSize}&search=${encodeURIComponent(search)}`);
}
export async function createMitigationAction(input: Partial<DomainRecord>) {
  return request<DomainRecord>('/actions', { method: 'POST', body: JSON.stringify(input) });
}
export async function transitionMitigationAction(id: number, status: string, expectedVersion: number, reason: string) {
  return request<DomainRecord>(`/actions/${id}/transition`, {
    method: 'POST', body: JSON.stringify({ status, expectedVersion, reason }),
  });
}
