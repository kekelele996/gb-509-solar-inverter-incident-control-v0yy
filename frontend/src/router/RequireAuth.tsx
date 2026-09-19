import type { ReactNode } from 'react';
import { Navigate } from 'react-router-dom';
import { getToken } from '../api/client';

export function RequireAuth({ children }: { children: ReactNode }) {
  return getToken() ? children : <Navigate to="/sites" replace />;
}
