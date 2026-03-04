import { Routes, Route, Navigate } from 'react-router-dom';
import Login from './pages/Login';
import Register from './pages/Register';
import UserDashboard from './pages/Dashboard';
import AdminDashboard from './pages/admin/Dashboard';
import ChannelManagement from './pages/admin/Channel';
import UserManagement from './pages/admin/User';
import SystemSettings from './pages/admin/Setting';
import GroupManagement from './pages/admin/Group';
import ModelManagement from './pages/admin/Model';
import Pricing from './pages/Pricing';
import Playground from './pages/Playground';
import { useAuthStore } from './store/auth';
import UserLayout from './layouts/UserLayout';
import AdminLayout from './layouts/AdminLayout';
import LogPage from './pages/Log';
import TokenManagement from './pages/user/Token';
import RedemptionManagement from './pages/admin/Redemption';
import TopupPage from './pages/user/Topup';
import ProfilePage from './pages/user/Profile';
import UserInvitation from './pages/user/Invitation';
import SubscriptionManagement from './pages/admin/Subscription';
import SubscriptionPage from './pages/user/Subscription';
import AdminInvitation from './pages/admin/Invitation';
import MigrationPage from './pages/admin/Migration';
import FilesPage from './pages/user/Files';
import TasksPage from './pages/admin/Tasks';
import UserTasksPage from './pages/user/Tasks';
import ModelMetaManagement from './pages/admin/ModelMeta';

function PrivateRoute({ children, roleRequired }: { children: React.ReactElement, roleRequired?: number }) {
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
      <Route path="/pricing" element={<Pricing />} />

      {/* 用户路由 */}
      <Route path="/" element={<PrivateRoute><UserLayout /></PrivateRoute>}>
        <Route index element={<UserDashboard />} />
        <Route path="token" element={<TokenManagement />} />
        <Route path="log" element={<LogPage />} />
        <Route path="topup" element={<TopupPage />} />
        <Route path="invitation" element={<UserInvitation />} />
        <Route path="subscription" element={<SubscriptionPage />} />
        <Route path="profile" element={<ProfilePage />} />
        <Route path="playground" element={<Playground />} />
        <Route path="files" element={<FilesPage />} />
        <Route path="tasks" element={<UserTasksPage />} />
        <Route path="pricing" element={<Pricing />} />
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
        <Route path="group" element={<GroupManagement />} />
        <Route path="model" element={<ModelManagement />} />
        <Route
          path="setting"
          element={
            <PrivateRoute roleRequired={100}>
              <SystemSettings />
            </PrivateRoute>
          }
        />
        <Route path="log" element={<LogPage />} />
        <Route path="subscription" element={<SubscriptionManagement />} />
        <Route path="invitation" element={<AdminInvitation />} />
        <Route path="migration" element={<MigrationPage />} />
        <Route path="tasks" element={<TasksPage />} />
        <Route path="ops" element={<TasksPage />} />
        <Route path="model-meta" element={<ModelMetaManagement />} />
      </Route>
    </Routes>
  );
}

export default App;
