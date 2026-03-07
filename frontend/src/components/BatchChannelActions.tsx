import { useState } from 'react';
import { Button, Modal, Select } from '../../components/ui';
import { CheckSquare, Trash2, Power, ArrowUp } from 'lucide-react';

/**
 * 批量渠道操作组件
 *
 * 功能：批量启用/禁用/删除渠道，批量修改优先级
 */
interface BatchChannelActionsProps {
  selectedIds: number[];           // 选中的渠道 ID 列表
  onSuccess: () => void;            // 操作成功回调
  onClearSelection: () => void;     // 清空选择回调
  token: string;                    // 认证 Token
}

export default function BatchChannelActions({
  selectedIds,
  onSuccess,
  onClearSelection,
  token,
}: BatchChannelActionsProps) {
  const [showModal, setShowModal] = useState(false);
  const [action, setAction] = useState<'status' | 'priority' | 'delete'>('status');
  const [statusValue, setStatusValue] = useState(1);
  const [priorityValue, setPriorityValue] = useState(10);
  const [loading, setLoading] = useState(false);

  if (selectedIds.length === 0) return null;

  /**
   * 执行批量操作
   */
  const executeBatchAction = async () => {
    setLoading(true);
    try {
      let endpoint = '';
      let body: any = { channel_ids: selectedIds };

      // 根据操作类型构建请求
      if (action === 'status') {
        endpoint = '/api/admin/channels/batch-status';
        body.status = statusValue;
      } else if (action === 'priority') {
        endpoint = '/api/admin/channels/batch-priority';
        body.priority = priorityValue;
      } else if (action === 'delete') {
        endpoint = '/api/admin/channels/batch-delete';
      }

      const res = await fetch(endpoint, {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(body),
      });

      const data = await res.json();
      if (data.code === 0) {
        alert(`操作成功，已处理 ${selectedIds.length} 个渠道`);
        setShowModal(false);
        onClearSelection();
        onSuccess();
      } else {
        alert(`操作失败: ${data.message}`);
      }
    } catch (error) {
      alert('操作失败');
      console.error(error);
    } finally {
      setLoading(false);
    }
  };

  return (
    <>
      <div className="flex items-center gap-2 p-4 bg-blue-50 border border-blue-200 rounded-lg">
        <CheckSquare className="w-5 h-5 text-blue-600" />
        <span className="font-medium">已选择 {selectedIds.length} 个渠道</span>
        <div className="flex-1" />
        <Button
          size="sm"
          variant="outline"
          onClick={() => {
            setAction('status');
            setShowModal(true);
          }}
        >
          <Power className="w-4 h-4 mr-1" />
          批量状态
        </Button>
        <Button
          size="sm"
          variant="outline"
          onClick={() => {
            setAction('priority');
            setShowModal(true);
          }}
        >
          <ArrowUp className="w-4 h-4 mr-1" />
          批量优先级
        </Button>
        <Button
          size="sm"
          variant="outline"
          onClick={() => {
            setAction('delete');
            setShowModal(true);
          }}
        >
          <Trash2 className="w-4 h-4 mr-1" />
          批量删除
        </Button>
        <Button size="sm" variant="outline" onClick={onClearSelection}>
          取消选择
        </Button>
      </div>

      {/* 批量操作确认弹窗 */}
      <Modal
        isOpen={showModal}
        onClose={() => setShowModal(false)}
        title="批量操作确认"
      >
        <div className="space-y-4">
          <p className="text-sm text-gray-600">
            即将对 <strong>{selectedIds.length}</strong> 个渠道执行操作
          </p>

          {action === 'status' && (
            <div>
              <label className="block text-sm font-medium mb-2">目标状态</label>
              <Select
                value={statusValue}
                onChange={(e) => setStatusValue(parseInt(e.target.value))}
              >
                <option value={1}>启用</option>
                <option value={2}>禁用</option>
              </Select>
            </div>
          )}

          {action === 'priority' && (
            <div>
              <label className="block text-sm font-medium mb-2">优先级</label>
              <input
                type="number"
                className="w-full px-3 py-2 border rounded-lg"
                value={priorityValue}
                onChange={(e) => setPriorityValue(parseInt(e.target.value) || 10)}
                min="0"
                max="100"
              />
              <p className="text-xs text-gray-500 mt-1">值越大优先级越高</p>
            </div>
          )}

          {action === 'delete' && (
            <div className="p-4 bg-red-50 border border-red-200 rounded-lg">
              <p className="text-sm text-red-600">
                ⚠️ 警告：删除操作不可恢复，请确认是否继续
              </p>
            </div>
          )}

          <div className="flex justify-end gap-2 pt-4">
            <Button variant="outline" onClick={() => setShowModal(false)}>
              取消
            </Button>
            <Button onClick={executeBatchAction} disabled={loading}>
              {loading ? '处理中...' : '确认执行'}
            </Button>
          </div>
        </div>
      </Modal>
    </>
  );
}
