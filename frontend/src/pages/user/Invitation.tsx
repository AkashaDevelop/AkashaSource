import { useState, useEffect } from 'react';
import PageHeader from '../../components/PageHeader';
import EmptyState from '../../components/EmptyState';
import LoadingRows from '../../components/LoadingRows';
import { Button, Chip, Snippet, Input, Modal, ModalContent, ModalHeader, ModalBody, ModalFooter, useDisclosure } from '../../components/ui';
import { Plus, RefreshCw, ArrowRightLeft } from 'lucide-react';
import { useAuthStore } from '../../store/auth';
import { toast } from '../../store/toast';

interface Invitation {
  id: number;
  code: string;
  status: number;
  cost: number;
  created_at: number;
  used_at: number;
  invitee_id: number;
}

export default function UserInvitation() {
  const [invitations, setInvitations] = useState<Invitation[]>([]);
  const [loading, setLoading] = useState(false);
  const [generating, setGenerating] = useState(false);
  const [affQuota, setAffQuota] = useState(0);
  const [transferAmount, setTransferAmount] = useState('');
  const [transferring, setTransferring] = useState(false);
  const { token, updateUser } = useAuthStore();
  const { isOpen, onOpen, onOpenChange } = useDisclosure();

  const fetchInvitations = async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/user/invitation', { headers: { Authorization: `Bearer ${token}` } });
      const data = await res.json();
      if (data.code === 0) setInvitations(data.data || []);
      else toast.error(data.msg || '获取邀请码失败');
    } catch (e: any) {
      toast.error(e.message);
    } finally {
      setLoading(false);
    }
  };

  const fetchAffQuota = async () => {
    try {
      const res = await fetch('/api/user/self', { headers: { Authorization: `Bearer ${token}` } });
      const data = await res.json();
      if (data.code === 0) {
        setAffQuota(data.data?.aff_quota ?? 0);
        updateUser(data.data);
      }
    } catch (e: any) {
      console.error(e);
    }
  };

  useEffect(() => { fetchInvitations(); fetchAffQuota(); }, []);

  const handleGenerate = async () => {
    setGenerating(true);
    try {
      const res = await fetch('/api/user/invitation', {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.code === 0) {
        toast.success('邀请码生成成功');
        fetchInvitations();
      } else {
        toast.error(data.msg || '生成失败');
      }
    } catch (e: any) {
      toast.error(e.message);
    } finally {
      setGenerating(false);
    }
  };

  const handleTransfer = async () => {
    const quota = Math.floor(parseFloat(transferAmount) || 0);
    if (quota <= 0) { toast.error('请输入有效的转账额度'); return; }
    if (quota > affQuota) { toast.error('转账额度不能超过邀请返利余额'); return; }
    setTransferring(true);
    try {
      const res = await fetch('/api/user/aff_transfer', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify({ quota }),
      });
      const data = await res.json();
      if (data.code === 0) {
        toast.success('转账成功');
        setTransferAmount('');
        onOpenChange(false);
        fetchAffQuota();
      } else {
        toast.error(data.msg || '转账失败');
      }
    } catch (e: any) {
      toast.error(e.message);
    } finally {
      setTransferring(false);
    }
  };

  const unused = invitations.filter(i => i.status === 1).length;
  const used = invitations.filter(i => i.status === 2).length;

  return (
    <div className="space-y-6">
      <PageHeader
        title="邀请管理"
        description="生成并管理您的邀请码，邀请好友注册"
        actions={
          <div className="flex gap-2">
            <Button variant="flat" startContent={<RefreshCw size={16} />} onPress={() => { fetchInvitations(); fetchAffQuota(); }} isLoading={loading}>
              刷新
            </Button>
            <Button color="primary" startContent={<Plus size={16} />} isLoading={generating} onPress={handleGenerate}>
              生成邀请码
            </Button>
          </div>
        }
      />

      {/* 邀请返利余额 */}
      <div style={{
        padding: '16px', borderRadius: 'var(--radius-xl)',
        background: 'var(--bg-surface)', border: '1px solid var(--border-color)',
        display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexWrap: 'wrap', gap: '12px',
      }}>
        <div>
          <p style={{ fontSize: '12px', color: 'var(--text-muted)', margin: '0 0 6px' }}>邀请返利余额（待转入可用额度）</p>
          <p style={{ fontSize: '24px', fontWeight: 700, color: 'var(--accent-primary)', margin: 0 }}>{affQuota.toLocaleString()}</p>
        </div>
        <Button
          color="primary" variant="flat"
          startContent={<ArrowRightLeft size={16} />}
          isDisabled={affQuota <= 0}
          onPress={onOpen}
        >
          转入可用额度
        </Button>
      </div>

      {/* 统计 */}
      <div className="grid grid-cols-3 gap-3">
        {[
          { label: '总邀请码', value: invitations.length, color: 'var(--accent-primary)', bg: 'var(--nav-active-bg)' },
          { label: '未使用', value: unused, color: 'var(--color-success-fg)', bg: 'var(--color-success-bg)' },
          { label: '已使用', value: used, color: 'var(--text-muted)', bg: 'var(--bg-elevated)' },
        ].map(s => (
          <div key={s.label} style={{
            padding: '16px', borderRadius: 'var(--radius-xl)',
            background: 'var(--bg-surface)', border: '1px solid var(--border-color)',
          }}>
            <p style={{ fontSize: '12px', color: 'var(--text-muted)', margin: '0 0 6px' }}>{s.label}</p>
            <p style={{ fontSize: '24px', fontWeight: 700, color: s.color, margin: 0 }}>{s.value}</p>
          </div>
        ))}
      </div>

      <div className="data-table-wrap">
        <table className="data-table">
          <thead>
            <tr>
              <th>邀请码</th>
              <th>状态</th>
              <th>成本（额度）</th>
              <th>创建时间</th>
              <th>使用时间</th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <LoadingRows cols={5} rows={5} />
            ) : invitations.length === 0 ? (
              <tr><td colSpan={5}><EmptyState icon="🎁" title="暂无邀请码" description="点击「生成邀请码」创建您的第一个邀请码" /></td></tr>
            ) : invitations.map((item) => (
              <tr key={item.id}>
                <td>
                  <Snippet symbol="" hideSymbol>
                    {item.code}
                  </Snippet>
                </td>
                <td>
                  <Chip color={item.status === 1 ? 'success' : 'default'} size="sm" variant="flat">
                    {item.status === 1 ? '未使用' : '已使用'}
                  </Chip>
                </td>
                <td>{item.cost > 0 ? item.cost.toLocaleString() : <span style={{ color: 'var(--text-muted)' }}>免费</span>}</td>
                <td style={{ whiteSpace: 'nowrap' }}>{new Date(item.created_at * 1000).toLocaleString()}</td>
                <td style={{ whiteSpace: 'nowrap' }}>{item.used_at ? new Date(item.used_at * 1000).toLocaleString() : <span style={{ color: 'var(--text-muted)' }}>-</span>}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <Modal isOpen={isOpen} onOpenChange={onOpenChange} size="sm">
        <ModalContent>
          <ModalHeader>转入可用额度</ModalHeader>
          <ModalBody>
            <p style={{ fontSize: '13px', color: 'var(--text-muted)', margin: '0 0 12px' }}>
              当前邀请返利余额：{affQuota.toLocaleString()}
            </p>
            <Input
              type="number"
              label="转账额度"
              placeholder={`最多 ${affQuota}`}
              value={transferAmount}
              onValueChange={setTransferAmount}
            />
          </ModalBody>
          <ModalFooter>
            <Button variant="flat" onPress={() => onOpenChange(false)}>取消</Button>
            <Button color="primary" isLoading={transferring} onPress={handleTransfer}>确认转账</Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
    </div>
  );
}
