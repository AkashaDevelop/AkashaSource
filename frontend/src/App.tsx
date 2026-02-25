import { Routes, Route, Navigate } from 'react-router-dom';
import Login from './pages/Login';
import Register from './pages/Register';
import UserDashboard from './pages/Dashboard'; // Reuse existing dashboard for user
import AdminDashboard from './pages/admin/Dashboard';
import ChannelManagement from './pages/admin/Channel';
import UserManagement from './pages/admin/User';
import SystemSettings from './pages/admin/Setting';
import { useAuthStore } from './store/auth';
import UserLayout from './layouts/UserLayout';
import AdminLayout from './layouts/AdminLayout';
import LogPage from './pages/Log';
import TokenManagement from './pages/user/Token';
import RedemptionManagement from './pages/admin/Redemption';
import TopupPage from './pages/user/Topup';
import ProfilePage from './pages/user/Profile';
import UserInvitation from './pages/user/Invitation';

function PrivateRoute({ children, roleRequired }: { children: JSX.Element, roleRequired?: number }) {
  const { token, user } = useAuthStore();
  
  if (!token || !user) {
    return <Navigate to="/login" />;
  }

  if (roleRequired && user.role < roleRequired) {
    return <Navigate to="/" />; // Redirect to user dashboard if not authorized
  }

  return children;
}

function App() {
  return (
    <Routes>
      <Route path="/login" element={<Login />} />
      <Route path="/register" element={<Register />} />
      
      {/* 用户路由 */}
      <Route
        path="/"
        element={
          <PrivateRoute>
            <UserLayout />
          </PrivateRoute>
        }
      >
        <Route index element={<UserDashboard />} />
        <Route path="token" element={<TokenManagement />} />
        <Route path="log" element={<LogPage />} />
        <Route path="topup" element={<TopupPage />} />
        <Route path="invitation" element={<UserInvitation />} />
        <Route path="profile" element={<ProfilePage />} />
      </Route>

      {/* 管理员路由 */}
      <Route
        path="/admin"
        element={
          <PrivateRoute roleRequired={10}>
            <AdminLayout />
          </PrivateRoute>
        }
      >
        <Route index element={<AdminDashboard />} />
        <Route path="channel" element={<ChannelManagement />} />
        <Route path="user" element={<UserManagement />} />
        <Route path="redemption" element={<RedemptionManagement />} />
        <Route path="setting" element={<SystemSettings />} />
        <Route path="log" element={<LogPage />} />
      </Route>
    </Routes>
  );
}

export default App;
