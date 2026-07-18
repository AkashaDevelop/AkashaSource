import { useEffect, useRef } from 'react';
import { Activity, DollarSign, Download, RefreshCw, Copy, Trash2 } from 'lucide-react';

// ✨ 超萌的右键菜单组件～
// 支持渠道操作的快捷菜单，点哪用哪！

export interface ContextMenuItem {
  key: string;
  label: string;
  icon: React.ReactNode;
  color?: 'default' | 'primary' | 'secondary' | 'success' | 'warning' | 'danger';
  onClick: () => void;
}

interface ChannelContextMenuProps {
  items: ContextMenuItem[];
  position: { x: number; y: number } | null;
  onClose: () => void;
}

export default function ChannelContextMenu({
  items,
  position,
  onClose,
}: ChannelContextMenuProps) {
  const menuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!position) return;

    const handleClickOutside = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        onClose();
      }
    };

    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose();
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    document.addEventListener('keydown', handleEscape);

    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
      document.removeEventListener('keydown', handleEscape);
    };
  }, [position, onClose]);

  if (!position) return null;

  const getColorClass = (color?: string) => {
    switch (color) {
      case 'primary':
        return 'hover:bg-primary-50 hover:text-primary';
      case 'secondary':
        return 'hover:bg-secondary-50 hover:text-secondary';
      case 'success':
        return 'hover:bg-success-50 hover:text-success';
      case 'warning':
        return 'hover:bg-warning-50 hover:text-warning';
      case 'danger':
        return 'hover:bg-danger-50 hover:text-danger';
      default:
        return 'hover:bg-default-100';
    }
  };

  return (
    <div
      ref={menuRef}
      className="fixed z-50 min-w-[200px] bg-white dark:bg-gray-800 border border-default-200 rounded-lg shadow-lg overflow-hidden"
      style={{
        top: position.y,
        left: position.x,
      }}
    >
      {items.map((item) => (
        <button
          key={item.key}
          className={`w-full px-4 py-2.5 flex items-center gap-3 text-sm transition-colors ${getColorClass(
            item.color
          )}`}
          onClick={() => {
            item.onClick();
            onClose();
          }}
        >
          <span className="flex-shrink-0">{item.icon}</span>
          <span className="flex-1 text-left">{item.label}</span>
        </button>
      ))}
    </div>
  );
}

// 预设的渠道操作菜单项
export const createChannelContextMenuItems = (
  handlers: {
    onTest: () => void;
    onBalance: () => void;
    onFetchModels: () => void;
    onUpdate: () => void;
    onCopy: () => void;
    onDelete: () => void;
  }
): ContextMenuItem[] => [
  {
    key: 'test',
    label: '测试连接',
    icon: <Activity size={16} />,
    color: 'primary',
    onClick: handlers.onTest,
  },
  {
    key: 'balance',
    label: '查询余额',
    icon: <DollarSign size={16} />,
    color: 'success',
    onClick: handlers.onBalance,
  },
  {
    key: 'fetch-models',
    label: '获取模型',
    icon: <Download size={16} />,
    color: 'secondary',
    onClick: handlers.onFetchModels,
  },
  {
    key: 'update',
    label: '上游更新',
    icon: <RefreshCw size={16} />,
    color: 'warning',
    onClick: handlers.onUpdate,
  },
  {
    key: 'copy',
    label: '复制渠道',
    icon: <Copy size={16} />,
    onClick: handlers.onCopy,
  },
  {
    key: 'delete',
    label: '删除',
    icon: <Trash2 size={16} />,
    color: 'danger',
    onClick: handlers.onDelete,
  },
];
