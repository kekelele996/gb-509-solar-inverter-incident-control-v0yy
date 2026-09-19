
import { request } from './client';
import type { DomainRecord } from '../types/domain';

export async function listSolarSite(page = 1, pageSize = 20, search = '') {
  return request<DomainRecord[]>(`/sites?page=${page}&pageSize=${pageSize}&search=${encodeURIComponent(search)}`);
}
export async function createSolarSite(input: Partial<DomainRecord>) {
  return request<DomainRecord>('/sites', { method: 'POST', body: JSON.stringify(input) });
}
export async function transitionSolarSite(id: number, status: string, expectedVersion: number, reason: string) {
  return request<DomainRecord>(`/sites/${id}/transition`, {
    method: 'POST', body: JSON.stringify({ status, expectedVersion, reason }),
  });
}
