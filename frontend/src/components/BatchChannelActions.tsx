import { useState } from 'react';
import {
  Button,
  Modal,
  ModalBody,
  ModalContent,
  ModalFooter,
  ModalHeader,
  Select,
  SelectItem,
} from './ui';
import { CheckSquare, Trash2, Power, ArrowUp } from 'lucide-react';
import { toast } from '../store/toast';

interface BatchChannelActionsProps {
  selectedIds: number[];
  onSuccess: () => void;
  onClearSelection: () => void;
  token: string;
}

export default function BatchChannelActions({
  selectedIds,
  onSuccess,
  onClearSelection,
  token,
}: BatchChannelActionsProps) {
  const [showModal, setShowModal] = useState(false);
  const [action, setAction] = useState<'status' | 'priority' | 'delete'>('status');
  const [statusValue, setStatusValue] = useState<'1' | '2'>('1');
  const [priorityValue, setPriorityValue] = useState('10');
  const [loading, setLoading] = useState(false);

  if (selectedIds.length === 0) return null;

  const openActionModal = (nextAction: 'status' | 'priority' | 'delete') => {
    setAction(nextAction);
    setShowModal(true);
  };

  const executeBatchAction = async () => {
    setLoading(true);
    try {
      let endpoint = '';
      const body: Record<string, unknown> = { channel_ids: selectedIds };

      if (action === 'status') {
        endpoint = '/api/admin/channels/batch-status';
        body.status = parseInt(statusValue, 10);
      } else if (action === 'priority') {
        endpoint = '/api/admin/channels/batch-priority';
        body.priority = parseInt(priorityValue, 10) || 10;
      } else {
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
        toast.success(`操作成功，已处理 ${selectedIds.length} 个渠道`);
        setShowModal(false);
        onClearSelection();
        onSuccess();
      } else {
        toast.error(data.msg || data.message || '批量操作失败');
      }
    } catch (error) {
      console.error(error);
      toast.error('批量操作失败');
    } finally {
      setLoading(false);
    }
  };

  return (
    <>
      <div
        className="flex items-center gap-2 p-4 rounded-xl flex-wrap"
        style={{ background: 'var(--color-info-bg)', border: '1px solid var(--border-color)' }}
      >
        <CheckSquare className="w-5 h-5" style={{ color: 'var(--accent-primary)' }} />
        <span className="font-medium" style={{ color: 'var(--text-primary)' }}>
          已选择 {selectedIds.length} 个渠道
        </span>
        <div className="flex-1" />
        <Button size="sm" variant="bordered" color="primary" onPress={() => openActionModal('status')}>
          <Power className="w-4 h-4 mr-1" />
          批量状态
        </Button>
        <Button size="sm" variant="bordered" color="secondary" onPress={() => openActionModal('priority')}>
          <ArrowUp className="w-4 h-4 mr-1" />
          批量优先级
        </Button>
        <Button size="sm" variant="bordered" color="danger" onPress={() => openActionModal('delete')}>
          <Trash2 className="w-4 h-4 mr-1" />
          批量删除
        </Button>
        <Button size="sm" variant="light" onPress={onClearSelection}>
          取消选择
        </Button>
      </div>

      <Modal isOpen={showModal} onOpenChange={setShowModal} size="md">
        <ModalContent>
          {(onClose) => (
            <>
              <ModalHeader onClose={onClose}>批量操作确认</ModalHeader>
              <ModalBody>
                <div className="space-y-4">
                  <p className="text-sm" style={{ color: 'var(--text-secondary)' }}>
                    即将对 <strong>{selectedIds.length}</strong> 个渠道执行操作。
                  </p>

                  {action === 'status' && (
                    <Select
                      label="目标状态"
                      selectedKeys={[statusValue]}
                      onSelectionChange={(keys) => setStatusValue((Array.from(keys)[0] as '1' | '2') || '1')}
                    >
                      <SelectItem key="1">启用</SelectItem>
                      <SelectItem key="2">禁用</SelectItem>
                    </Select>
                  )}

                  {action === 'priority' && (
                    <div className="space-y-2">
                      <InputLabel label="优先级" />
                      <input
                        type="number"
                        className="w-full px-3 py-2 rounded-xl border text-sm bg-[var(--bg-elevated)] border-[var(--border-color)] text-[var(--text-primary)]"
                        value={priorityValue}
                        onChange={(e) => setPriorityValue(e.target.value)}
                        min="0"
                        max="100"
                      />
                      <p className="text-xs" style={{ color: 'var(--text-faint)' }}>值越大优先级越高</p>
                    </div>
                  )}

                  {action === 'delete' && (
                    <div
                      className="p-4 rounded-xl"
                      style={{ background: 'var(--color-danger-bg)', border: '1px solid rgba(239,68,68,0.2)' }}
                    >
                      <p className="text-sm" style={{ color: 'var(--color-danger-fg)' }}>
                        ⚠️ 删除操作不可恢复，请确认是否继续。
                      </p>
                    </div>
                  )}
                </div>
              </ModalBody>
              <ModalFooter>
                <Button variant="light" onPress={onClose}>取消</Button>
                <Button color={action === 'delete' ? 'danger' : 'primary'} isLoading={loading} onPress={executeBatchAction}>
                  确认执行
                </Button>
              </ModalFooter>
            </>
          )}
        </ModalContent>
      </Modal>
    </>
  );
}

function InputLabel({ label }: { label: string }) {
  return <label className="text-xs font-medium text-[var(--text-secondary)]">{label}</label>;
}
