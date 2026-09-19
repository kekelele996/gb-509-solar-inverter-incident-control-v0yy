
import { NavLink, Outlet } from 'react-router-dom';
import { useAuth } from './hooks/useAuth';
const navigation = [{ to: '/sites', label: '光伏场站' }, { to: '/inverters', label: '逆变器' }, { to: '/faults', label: '故障事件' }, { to: '/actions', label: '处置动作' }, { to: '/audit', label: '审计记录' }];
export default function App() {
  const { session, loading, logout } = useAuth();
  if (loading) return <div className="app-loading">正在建立安全会话…</div>;
  return <div className="app-shell"><aside><div className="brand"><span>CONTROL DESK</span><strong>光伏逆变器故障处置控制</strong></div><nav>{navigation.map((item) => <NavLink key={item.to} to={item.to}>{item.label}</NavLink>)}</nav><div className="user-panel"><span>{session?.displayName || '系统管理员'}</span><small>{session?.role || 'admin'}</small><button onClick={logout}>重新登录</button></div></aside><section className="content"><header className="topbar"><span>运行态势</span><span className="live-dot">服务已连接</span></header><Outlet /></section></div>;
}
