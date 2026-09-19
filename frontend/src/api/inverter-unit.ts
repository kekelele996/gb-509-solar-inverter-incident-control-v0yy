
import { request } from './client';
import type { DomainRecord } from '../types/domain';

export async function listInverterUnit(page = 1, pageSize = 20, search = '') {
  return request<DomainRecord[]>(`/inverters?page=${page}&pageSize=${pageSize}&search=${encodeURIComponent(search)}`);
}
export async function createInverterUnit(input: Partial<DomainRecord>) {
  return request<DomainRecord>('/inverters', { method: 'POST', body: JSON.stringify(input) });
}
export async function transitionInverterUnit(id: number, status: string, expectedVersion: number, reason: string) {
  return request<DomainRecord>(`/inverters/${id}/transition`, {
    method: 'POST', body: JSON.stringify({ status, expectedVersion, reason }),
  });
}
