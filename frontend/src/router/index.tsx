import { Navigate, createBrowserRouter } from 'react-router-dom';
import App from '../App';
import SolarSitePage from '../pages/SolarSitePage';
import InverterUnitPage from '../pages/InverterUnitPage';
import FaultEventPage from '../pages/FaultEventPage';
import MitigationActionPage from '../pages/MitigationActionPage';
import AuditPage from '../pages/AuditPage';
import { RequireAuth } from './RequireAuth';
export const router = createBrowserRouter([{ path: '/', element: <App />, children: [
  { index: true, element: <Navigate to="/sites" replace /> },
  { path: 'sites', element: <RequireAuth><SolarSitePage /></RequireAuth> }, { path: 'inverters', element: <RequireAuth><InverterUnitPage /></RequireAuth> }, { path: 'faults', element: <RequireAuth><FaultEventPage /></RequireAuth> }, { path: 'actions', element: <RequireAuth><MitigationActionPage /></RequireAuth> },
  { path: 'audit', element: <RequireAuth><AuditPage /></RequireAuth> },
] }], { future: { v7_fetcherPersist: true, v7_normalizeFormMethod: true, v7_partialHydration: true, v7_relativeSplatPath: true, v7_skipActionErrorRevalidation: true } });
