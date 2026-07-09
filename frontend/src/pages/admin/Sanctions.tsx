import { useEffect, useMemo, useState } from 'react';
import PageHeader from '../../components/PageHeader';
import EmptyState from '../../components/EmptyState';
import LoadingRows from '../../components/LoadingRows';
import {
  Button, Input, Select, SelectItem, Chip,
  Modal, ModalContent, ModalHeader, ModalBody, ModalFooter, useDisclosure,
} from '../../components/ui';
import { RefreshCw, Plus, Trash2, ShieldAlert } from 'lucide-react';
import { useAuthStore } from '../../store/auth';
import { toast } from '../../store/toast';
import { confirm } from '../../store/confirm';

// ～宸汐处置台：玄鉴自动处置和管理员手动处置都汇总在这里，可查看/解除/新增～

interface SanctionRow {
  id: number;
  target_type: string; // token / user / ip
  target_key: string;
  action: string;
  factor: number;
  reason: string;
  source: string; // xuanjian_auto / admin_manual
  enabled: boolean;
  expires_at: number; // 0 = 永久
  created_at: number;
  updated_at: number;
}

const targetLabel = (t: string) => ({ token: '令牌', user: '用户', ip: 'IP' }[t] || t);

const actionLabel = (a: string) => ({
  throttle: '高倍率限速',
  rpm_limit: '固定低RPM',
  billing_penalty: '高倍率计费',
  suspend_token: '短暂停用',
  disable_token: '停用令牌',
  ban_ip: '封禁IP',
  ban_user: '封禁用户',
}[a] || a);

const actionColor = (a: string): 'default' | 'warning' | 'danger' => ({
  throttle: 'warning',
  rpm_limit: 'warning',
  billing_penalty: 'warning',
  suspend_token: 'warning',
  disable_token: 'danger',
  ban_ip: 'danger',
  ban_user: 'danger',
}[a] as any || 'default');

// 每种目标类型允许的处置动作
const actionsByTarget: Record<string, { key: string; label: string }[]> = {
  token: [
    { key: 'throttle', label: '高倍率限速' },
    { key: 'rpm_limit', label: '固定低RPM' },
    { key: 'billing_penalty', label: '高倍率计费' },
    { key: 'suspend_token', label: '短暂停用' },
    { key: 'disable_token', label: '停用令牌' },
  ],
  user: [
    { key: 'billing_penalty', label: '高倍率计费' },
    { key: 'ban_user', label: '封禁用户' },
  ],
  ip: [
    { key: 'ban_ip', label: '封禁IP' },
  ],
};

// factor 语义提示
const factorHint = (action: string) => ({
  throttle: '降速倍率 0~1（如 0.3 表示降到 30%）',
  rpm_limit: '目标 RPM 绝对值（如 5）',
  billing_penalty: '计费倍率（如 5 表示 5 倍收费）',
}[action] || '此动作无需 factor');

const emptyForm = {
  target_type: 'token',
  target_key: '',
  action: 'throttle',
  factor: 0.3,
  reason: '',
  duration_minutes: 60,
};

export default function Sanctions() {
  const { token } = useAuthStore();
  const { isOpen, onOpen, onOpenChange } = useDisclosure();
  const [rows, setRows] = useState<SanctionRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [targetFilter, setTargetFilter] = useState('');
  const [form, setForm] = useState(emptyForm);
  const [saving, setSaving] = useState(false);

  const authHeaders = useMemo(() => ({ Authorization: `Bearer ${token}` }), [token]);

  const fetchRows = async () => {
    setLoading(true);
    const params = new URLSearchParams();
    if (targetFilter) params.set('target_type', targetFilter);
    try {
      const res = await fetch(`/api/admin/sanctions?${params}`, { headers: authHeaders });
      const data = await res.json();
      if (data.code === 0) setRows(data.data?.list || []);
      else toast.error(data.msg || '查询失败');
    } catch (e) { console.error(e); toast.error('网络异常'); }
    finally { setLoading(false); }
  };

  useEffect(() => { if (token) fetchRows(); }, [token, targetFilter]);

  const openCreate = () => { setForm(emptyForm); onOpen(); };

  // 切换目标类型时，自动把 action 重置为该类型的第一个合法动作
  const onTargetTypeChange = (t: string) => {
    const acts = actionsByTarget[t] || [];
    setForm({ ...form, target_type: t, action: acts[0]?.key || '' });
  };

  const saveSanction = async (onClose: () => void) => {
    if (!form.target_key.trim()) {
      toast.error('请填写处置目标');
      return;
    }
    setSaving(true);
    try {
      const res = await fetch('/api/admin/sanctions', {
        method: 'POST',
        headers: { ...authHeaders, 'Content-Type': 'application/json' },
        body: JSON.stringify({
          target_type: form.target_type,
          target_key: form.target_key.trim(),
          action: form.action,
          factor: Number(form.factor) || 0,
          reason: form.reason.trim(),
          duration_minutes: Number(form.duration_minutes) || 0,
        }),
      });
      const data = await res.json();
      if (data.code === 0) { toast.success('处置已生效'); fetchRows(); onClose(); }
      else toast.error(data.msg || '施加处置失败');
    } catch { toast.error('网络异常'); }
    finally { setSaving(false); }
  };

  const revoke = async (r: SanctionRow) => {
    if (!await confirm({ title: '解除处置', message: `确定解除对 ${targetLabel(r.target_type)} ${r.target_key} 的「${actionLabel(r.action)}」处置？`, danger: true })) return;
    try {
      const res = await fetch(`/api/admin/sanctions/${r.id}`, { method: 'DELETE', headers: authHeaders });
      const data = await res.json();
      if (data.code === 0) { toast.success('已解除'); fetchRows(); }
      else toast.error(data.msg || '解除失败');
    } catch { toast.error('网络异常'); }
  };

  const fmtExpiry = (ts: number) => {
    if (ts === 0) return '永久';
    if (ts * 1000 < Date.now()) return '已过期';
    return new Date(ts * 1000).toLocaleString();
  };

  const availableActions = actionsByTarget[form.target_type] || [];
  const needsFactor = ['throttle', 'rpm_limit', 'billing_penalty'].includes(form.action);

  return (
    <div className="space-y-6">
      <PageHeader
        title="宸汐处置台"
        description="玄鉴自动处置与管理员手动处置的统一管理 · 限速/停用/封禁/高倍率计费均可查看与解除 · 仅超级管理员可访问"
        actions={
          <div className="flex gap-2">
            <Button variant="flat" onPress={fetchRows} startContent={<RefreshCw size={16} />}>刷新</Button>
            <Button color="primary" onPress={openCreate} startContent={<Plus size={16} />}>手动处置</Button>
          </div>
        }
      />

      <div className="flex gap-2 flex-wrap items-center">
        <Select placeholder="目标类型" className="w-44" selectedKeys={targetFilter ? [targetFilter] : []}
          onSelectionChange={keys => setTargetFilter([...keys][0] as string || '')}>
          <SelectItem key="token">令牌</SelectItem>
          <SelectItem key="user">用户</SelectItem>
          <SelectItem key="ip">IP</SelectItem>
        </Select>
      </div>

      <div className="data-table-wrap">
        <table className="data-table">
          <thead><tr><th>目标</th><th>动作</th><th>Factor</th><th>原因</th><th>来源</th><th>到期</th><th>操作</th></tr></thead>
          <tbody>
            {loading ? <LoadingRows cols={7} rows={5} /> :
              rows.length === 0 ? (
                <tr><td colSpan={7}><EmptyState icon="🛡️" title="暂无处置记录" /></td></tr>
              ) : rows.map(r => (
                <tr key={r.id}>
                  <td className="text-xs">
                    <Chip size="sm" variant="flat">{targetLabel(r.target_type)}</Chip>
                    <span className="ml-1">{r.target_key}</span>
                  </td>
                  <td><Chip size="sm" color={actionColor(r.action)} variant="flat">{actionLabel(r.action)}</Chip></td>
                  <td className="text-xs">{['throttle', 'rpm_limit', 'billing_penalty'].includes(r.action) ? r.factor : '-'}</td>
                  <td className="text-xs max-w-xs truncate" title={r.reason}>{r.reason || '-'}</td>
                  <td><Chip size="sm" variant="flat" color={r.source === 'admin_manual' ? 'primary' : 'default'}>{r.source === 'admin_manual' ? '手动' : '玄鉴自动'}</Chip></td>
                  <td className="text-xs whitespace-nowrap">{fmtExpiry(r.expires_at)}</td>
                  <td>
                    <span className="cursor-pointer text-danger" onClick={() => revoke(r)}><Trash2 size={16} /></span>
                  </td>
                </tr>
              ))
            }
          </tbody>
        </table>
      </div>

      <Modal isOpen={isOpen} onOpenChange={onOpenChange} size="lg">
        <ModalContent>{onClose => <>
          <ModalHeader className="flex items-center gap-2"><ShieldAlert size={18} /> 手动施加处置</ModalHeader>
          <ModalBody className="gap-4">
            <div className="grid grid-cols-2 gap-4">
              <Select label="目标类型" selectedKeys={[form.target_type]}
                onSelectionChange={keys => onTargetTypeChange([...keys][0] as string || 'token')}>
                <SelectItem key="token">令牌（Token ID）</SelectItem>
                <SelectItem key="user">用户（User ID）</SelectItem>
                <SelectItem key="ip">IP 地址</SelectItem>
              </Select>
              <Input
                label={form.target_type === 'ip' ? 'IP 地址' : `${targetLabel(form.target_type)} ID`}
                placeholder={form.target_type === 'ip' ? '如 1.2.3.4' : '数字 ID'}
                value={form.target_key}
                onValueChange={v => setForm({ ...form, target_key: v })}
              />
            </div>
            <Select label="处置动作" selectedKeys={form.action ? [form.action] : []}
              onSelectionChange={keys => setForm({ ...form, action: [...keys][0] as string || '' })}>
              {availableActions.map(a => <SelectItem key={a.key}>{a.label}</SelectItem>)}
            </Select>
            {needsFactor && (
              <Input type="number" label="Factor" description={factorHint(form.action)}
                value={String(form.factor)} onValueChange={v => setForm({ ...form, factor: parseFloat(v) || 0 })} />
            )}
            <Input type="number" label="持续时间（分钟，0=永久）"
              value={String(form.duration_minutes)} onValueChange={v => setForm({ ...form, duration_minutes: parseInt(v) || 0 })} />
            <Input label="处置原因（可选）" placeholder="便于事后审计"
              value={form.reason} onValueChange={v => setForm({ ...form, reason: v })} />
          </ModalBody>
          <ModalFooter>
            <Button variant="light" onPress={onClose}>取消</Button>
            <Button color="primary" isLoading={saving} onPress={() => saveSanction(onClose)}>施加处置</Button>
          </ModalFooter>
        </>}</ModalContent>
      </Modal>
    </div>
  );
}
