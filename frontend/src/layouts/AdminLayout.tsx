import { Outlet, useNavigate, useLocation } from 'react-router-dom';
import {
  Listbox,
  ListboxItem,
  User,
  Button,
} from '@heroui/react';
import { LayoutDashboard, Server, Settings, LogOut, Users, Gift } from 'lucide-react';
import { useAuthStore } from '../store/auth';

export default function AdminLayout() {
  const navigate = useNavigate();
  const location = useLocation();
  const { user, logout } = useAuthStore();

  const handleLogout = () => {
    logout();
    navigate('/login');
  };

  return (
    <div className="flex min-h-screen w-full bg-gray-50 dark:bg-gray-950">
      {/* 管理员侧边栏 */}
      <div className="w-64 bg-gray-900 text-white flex flex-col">
        <div className="p-6">
          <h1 className="text-xl font-bold text-white">
            Akasha
          </h1>
          <div className="flex items-center gap-2 mt-1">
            <span className="px-2 py-0.5 rounded bg-purple-500/20 text-purple-300 text-xs border border-purple-500/30">
              Admin
            </span>
            <p className="text-xs text-gray-400">管理控制台</p>
          </div>
        </div>

        <div className="flex-1 px-4">
          <Listbox 
            aria-label="Admin Menu"
            onAction={(key) => navigate(key as string)}
            variant="flat"
            className="p-0 gap-2"
            itemClasses={{
              base: "px-3 rounded-lg gap-3 h-10 text-gray-300 data-[hover=true]:bg-gray-800 data-[hover=true]:text-white",
              title: "text-sm",
            }}
          >
            <ListboxItem
              key="/admin"
              startContent={<LayoutDashboard size={18} />}
              className={location.pathname === '/admin' ? "bg-primary text-white" : ""}
            >
              系统概览
            </ListboxItem>
            <ListboxItem
              key="/admin/channel"
              startContent={<Server size={18} />}
              className={location.pathname === '/admin/channel' ? "bg-primary text-white" : ""}
            >
              渠道管理
            </ListboxItem>
            <ListboxItem
              key="/admin/user"
              startContent={<Users size={18} />}
              className={location.pathname === '/admin/user' ? "bg-primary text-white" : ""}
            >
              用户管理
            </ListboxItem>
            <ListboxItem
              key="/admin/redemption"
              startContent={<Gift size={18} />}
              className={location.pathname === '/admin/redemption' ? "bg-primary text-white" : ""}
            >
              兑换码
            </ListboxItem>
            <ListboxItem
              key="/admin/setting"
              startContent={<Settings size={18} />}
              className={location.pathname === '/admin/setting' ? "bg-primary text-white" : ""}
            >
              系统设置
            </ListboxItem>
          </Listbox>
        </div>

        <div className="p-4 border-t border-gray-800 bg-gray-900">
          <div 
            className="flex items-center gap-3 mb-4 px-2 cursor-pointer hover:bg-gray-800 rounded-lg p-2 transition-colors"
            onClick={() => navigate('/profile')}
          >
            <User   
              name={user?.username}
              description="超级管理员"
              classNames={{
                name: "text-white",
                description: "text-gray-400"
              }}
              avatarProps={{
                src: "https://i.pravatar.cc/150?u=a04258114e29026708c"
              }}
            />
          </div>
          <Button 
            color="danger" 
            variant="flat" 
            startContent={<LogOut size={18} />}
            className="w-full"
            onPress={handleLogout}
          >
            退出登录
          </Button>
        </div>
      </div>

      {/* 主内容区 */}
      <div className="flex-1 overflow-auto p-6">
        <Outlet />
      </div>
    </div>
  );
}
