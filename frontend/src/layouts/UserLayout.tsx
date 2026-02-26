import { Outlet, useNavigate, useLocation } from 'react-router-dom';
import {
  Listbox,
  ListboxItem,
  User,
  Button,
} from '@heroui/react';
import { LayoutDashboard, Key, LogOut, History, Wallet, User as UserIcon, Share2, FlaskConical } from 'lucide-react';
import { useAuthStore } from '../store/auth';

export default function UserLayout() {
  const navigate = useNavigate();
  const location = useLocation();
  const { user, logout } = useAuthStore();

  const handleLogout = () => {
    logout();
    navigate('/login');
  };

  return (
    <div className="flex min-h-screen w-full bg-gray-50 dark:bg-gray-950">
      {/* 侧边栏 */}
      <div className="w-64 bg-white dark:bg-gray-900 border-r border-gray-200 dark:border-gray-800 flex flex-col">
        <div className="p-6">
          <h1 className="text-xl font-bold bg-gradient-to-r from-blue-500 to-purple-600 bg-clip-text text-transparent">
            Akasha
          </h1>
          <p className="text-xs text-default-400">用户控制台</p>
        </div>

        <div className="flex-1 px-4">
          <Listbox 
            aria-label="用户菜单"
            onAction={(key) => navigate(key as string)}
            className="p-0 gap-0 divide-y divide-default-300/50 dark:divide-default-100/80 bg-content1 max-w-[300px] overflow-visible shadow-small rounded-medium"
            itemClasses={{
              base: "px-3 first:rounded-t-medium last:rounded-b-medium rounded-none gap-3 h-12 data-[hover=true]:bg-default-100/80",
            }}
          >
            <ListboxItem
              key="/"
              startContent={<LayoutDashboard size={20} />}
              className={location.pathname === '/' ? "bg-primary/10 text-primary" : ""}
            >
              仪表盘
            </ListboxItem>
            <ListboxItem
              key="/token"
              startContent={<Key size={20} />}
              className={location.pathname === '/token' ? "bg-primary/10 text-primary" : ""}
            >
              令牌管理
            </ListboxItem>
            <ListboxItem
              key="/log"
              startContent={<History size={20} />}
              className={location.pathname === '/log' ? "bg-primary/10 text-primary" : ""}
            >
              调用日志
            </ListboxItem>
            <ListboxItem
              key="/topup"
              startContent={<Wallet size={20} />}
              className={location.pathname === '/topup' ? "bg-primary/10 text-primary" : ""}
            >
              充值
            </ListboxItem>
            <ListboxItem
              key="/invitation"
              startContent={<Share2 size={20} />}
              className={location.pathname === '/invitation' ? "bg-primary/10 text-primary" : ""}
            >
              邀请管理
            </ListboxItem>
            <ListboxItem
              key="/profile"
              startContent={<UserIcon size={20} />}
              className={location.pathname === '/profile' ? "bg-primary/10 text-primary" : ""}
            >
              个人设置
            </ListboxItem>
            <ListboxItem
              key="/playground"
              startContent={<FlaskConical size={20} />}
              className={location.pathname === '/playground' ? "bg-primary/10 text-primary" : ""}
            >
              Playground
            </ListboxItem>
          </Listbox>
        </div>

        <div className="p-4 border-t border-gray-200 dark:border-gray-800">
          <div className="flex items-center gap-4 mb-4">
            <User   
              name={user?.username}
              description="普通用户"
              avatarProps={{
                src: "https://i.pravatar.cc/150?u=a042581f4e29026024d"
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
      <div className="flex-1 overflow-auto">
        <Outlet />
      </div>
    </div>
  );
}
