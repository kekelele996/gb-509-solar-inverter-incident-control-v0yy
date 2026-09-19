
import { request } from './client';
import type { DomainRecord, InterlockConfirmResult } from '../types/domain';

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
export async function confirmMitigationAction(id: number, expectedVersion: number, reason: string) {
  return request<InterlockConfirmResult>(`/actions/${id}/confirm`, {
    method: 'POST', body: JSON.stringify({ expectedVersion, reason, confirmed: true }),
  });
}
