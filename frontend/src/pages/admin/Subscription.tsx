import { useEffect, useMemo, useState } from 'react';
import PageHeader from '../../components/PageHeader';
import EmptyState from '../../components/EmptyState';
import LoadingRows from '../../components/LoadingRows';
import {
  Button,
  Chip,
  Input,
  Modal,
  ModalBody,
  ModalContent,
  ModalFooter,
  ModalHeader,
  Select,
  SelectItem,
  useDisclosure,
} from '../../components/ui';
import {
  CircleDashed,
  Pencil,
  Plus,
  RefreshCw,
  Search,
  Trash2,
} from 'lucide-react';
import { useAuthStore } from '../../store/auth';
import { toast } from '../../store/toast';
import { confirm } from '../../store/confirm';

interface Plan {
  id: number;
  name: string;
  description: string;
  price: number;
  duration_days: number;
  type: string;
  group_name: string;
  quota: number;
  rpm: number;
  enabled: boolean;
  sort_order: number;
}

interface PlanResponseItem {
  plan?: Plan;
  id?: number;
  name?: string;
  description?: string;
  price?: number;
  duration_days?: number;
  type?: string;
  group_name?: string;
  quota?: number;
  rpm?: number;
  enabled?: boolean;
  sort_order?: number;
}

const TYPE_LABELS: Record<string, string> = {
  group: '分组订阅',
  quota: '额度包',
  rpm: '高速 RPM',
  combo: '组合套餐',
};

const TYPE_COLORS: Record<string, 'default' | 'primary' | 'secondary' | 'success' | 'warning' | 'danger'> = {
  group: 'secondary',
  quota: 'success',
  rpm: 'warning',
  combo: 'primary',
};

const emptyPlan: Omit<Plan, 'id'> = {
  name: '',
  description: '',
  price: 9.9,
  duration_days: 30,
  type: 'quota',
  group_name: '',
  quota: 500000,
  rpm: 0,
  enabled: true,
  sort_order: 0,
};

const normalizePlan = (item: PlanResponseItem): Plan => {
  const p = item.plan || item;
  return {
    id: Number(p.id || 0),
    name: p.name || '',
    description: p.description || '',
    price: Number(p.price || 0),
    duration_days: Number(p.duration_days || 0),
    type: p.type || 'quota',
    group_name: p.group_name || '',
    quota: Number(p.quota || 0),
    rpm: Number(p.rpm || 0),
    enabled: Boolean(p.enabled),
    sort_order: Number(p.sort_order || 0),
  };
};

const formatQuotaUsd = (quota: number) => `$${(quota / 500000).toFixed(2)}`;

export default function SubscriptionManagement() {
  const { token } = useAuthStore();
  const { isOpen, onOpen, onOpenChange } = useDisclosure();

  const [plans, setPlans] = useState<Plan[]>([]);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [keyword, setKeyword] = useState('');
  const [typeFilter, setTypeFilter] = useState('');
  const [statusFilter, setStatusFilter] = useState('');

  const [editing, setEditing] = useState<Partial<Plan> | null>(null);
  const [isNew, setIsNew] = useState(false);

  const fetchPlans = async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/subscription/admin/plans', {
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.code === 0) {
        const raw = Array.isArray(data.data) ? data.data : [];
        setPlans(raw.map((it: PlanResponseItem) => normalizePlan(it)).filter((it: Plan) => it.id > 0));
      } else {
        toast.error(data.msg || '拉取订阅套餐失败');
      }
    } catch (error) {
      console.error(error);
      toast.error('网络异常，拉取套餐失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (!token) return;
    fetchPlans();
  }, [token]);

  const filteredPlans = useMemo(() => {
    return plans.filter((plan) => {
      const hitKeyword = keyword.trim()
        ? [plan.name, plan.description, plan.group_name]
            .join(' ')
            .toLowerCase()
            .includes(keyword.trim().toLowerCase())
        : true;
      const hitType = typeFilter ? plan.type === typeFilter : true;
      const hitStatus = statusFilter ? String(Number(plan.enabled)) === statusFilter : true;
      return hitKeyword && hitType && hitStatus;
    });
  }, [plans, keyword, typeFilter, statusFilter]);

  const openNew = () => {
    setIsNew(true);
    setEditing({ ...emptyPlan });
    onOpen();
  };

  const openEdit = (plan: Plan) => {
    setIsNew(false);
    setEditing({ ...plan });
    onOpen();
  };

  const updateField = <K extends keyof Plan>(key: K, value: Plan[K]) => {
    setEditing((prev) => ({ ...(prev || {}), [key]: value }));
  };

  const savePlan = async (onClose: () => void) => {
    if (!editing?.name?.trim()) {
      toast.error('请填写套餐名称');
      return;
    }
    if ((editing.price ?? 0) < 0) {
      toast.error('价格不能为负数');
      return;
    }

    const payload: Partial<Plan> = {
      ...editing,
      name: editing.name.trim(),
      description: (editing.description || '').trim(),
      group_name: (editing.group_name || '').trim(),
      price: Number(editing.price || 0),
      duration_days: Number(editing.duration_days || 0),
      quota: Number(editing.quota || 0),
      rpm: Number(editing.rpm || 0),
      sort_order: Number(editing.sort_order || 0),
      enabled: Boolean(editing.enabled),
      type: editing.type || 'quota',
    };

    setSaving(true);
    try {
      const method = isNew ? 'POST' : 'PUT';
      const url = isNew
        ? '/api/subscription/admin/plans'
        : `/api/subscription/admin/plans/${editing.id}`;

      const res = await fetch(url, {
        method,
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({ plan: payload }),
      });
      const data = await res.json();

      if (data.code === 0) {
        toast.success(isNew ? '套餐创建成功' : '套餐更新成功');
        onClose();
        setEditing(null);
        fetchPlans();
      } else {
        toast.error(data.msg || '保存失败');
      }
    } catch (error) {
      console.error(error);
      toast.error('网络异常，保存失败');
    } finally {
      setSaving(false);
    }
  };

  const toggleEnabled = async (plan: Plan) => {
    try {
      const res = await fetch(`/api/subscription/admin/plans/${plan.id}`, {
        method: 'PATCH',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({ enabled: !plan.enabled }),
      });
      const data = await res.json();
      if (data.code === 0) {
        toast.success(!plan.enabled ? '已启用套餐' : '已禁用套餐');
        fetchPlans();
      } else {
        toast.error(data.msg || '状态更新失败');
      }
    } catch (error) {
      console.error(error);
      toast.error('网络异常，状态更新失败');
    }
  };

  const deletePlan = async (plan: Plan) => {
    const ok = await confirm({
      title: '删除套餐',
      message: `确认删除套餐「${plan.name}」吗？该操作不可恢复。`,
      danger: true,
    });
    if (!ok) return;

    try {
      const res = await fetch(`/api/subscription/plan/${plan.id}`, {
        method: 'DELETE',
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.code === 0) {
        toast.success('删除成功');
        fetchPlans();
      } else {
        toast.error(data.msg || '删除失败');
      }
    } catch (error) {
      console.error(error);
      toast.error('网络异常，删除失败');
    }
  };

  return (
    <div className="space-y-4">
      <PageHeader
        title="订阅管理"
        description="管理订阅套餐、价格、额度与生效状态"
        actions={
          <div className="flex gap-2 flex-wrap">
            <Input
              placeholder="搜索名称/描述/分组"
              value={keyword}
              onValueChange={setKeyword}
              className="w-44"
              startContent={<Search size={14} />}
            />
            <Select
              className="w-32"
              label="套餐类型"
              placeholder="全部"
              selectedKeys={typeFilter ? [typeFilter] : []}
              onSelectionChange={(keys) => setTypeFilter(([...keys][0] as string) || '')}
            >
              <SelectItem key="quota">额度包</SelectItem>
              <SelectItem key="group">分组订阅</SelectItem>
              <SelectItem key="rpm">高速 RPM</SelectItem>
              <SelectItem key="combo">组合套餐</SelectItem>
            </Select>
            <Select
              className="w-28"
              label="状态"
              placeholder="全部"
              selectedKeys={statusFilter ? [statusFilter] : []}
              onSelectionChange={(keys) => setStatusFilter(([...keys][0] as string) || '')}
            >
              <SelectItem key="1">启用</SelectItem>
              <SelectItem key="0">禁用</SelectItem>
            </Select>
            <Button variant="flat" startContent={<RefreshCw size={16} />} onPress={fetchPlans}>
              刷新
            </Button>
            <Button color="primary" startContent={<Plus size={16} />} onPress={openNew}>
              新建套餐
            </Button>
          </div>
        }
      />

      <div className="data-table-wrap">
        <table className="data-table">
          <thead>
            <tr>
              <th>ID</th>
              <th>套餐</th>
              <th>价格</th>
              <th>时长</th>
              <th>权益</th>
              <th>排序</th>
              <th>状态</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <LoadingRows cols={8} rows={6} />
            ) : filteredPlans.length === 0 ? (
              <tr>
                <td colSpan={8}>
                  <EmptyState icon="💳" title="暂无订阅套餐" />
                </td>
              </tr>
            ) : (
              filteredPlans.map((plan) => (
                <tr key={plan.id}>
                  <td>{plan.id}</td>
                  <td>
                    <div className="flex flex-col gap-1">
                      <div className="flex items-center gap-2 flex-wrap">
                        <span className="font-medium">{plan.name}</span>
                        <Chip size="sm" color={TYPE_COLORS[plan.type] || 'default'}>
                          {TYPE_LABELS[plan.type] || plan.type}
                        </Chip>
                      </div>
                      {plan.description ? (
                        <span className="text-xs" style={{ color: 'var(--text-secondary)' }}>
                          {plan.description}
                        </span>
                      ) : null}
                    </div>
                  </td>
                  <td>¥{Number(plan.price || 0).toFixed(2)}</td>
                  <td>{Number(plan.duration_days) > 0 ? `${plan.duration_days} 天` : '永久'}</td>
                  <td>
                    <div className="flex gap-2 flex-wrap text-xs" style={{ color: 'var(--text-secondary)' }}>
                      {(plan.type === 'group' || plan.type === 'combo') && plan.group_name ? (
                        <span>分组: {plan.group_name}</span>
                      ) : null}
                      {(plan.type === 'quota' || plan.type === 'combo') && plan.quota > 0 ? (
                        <span>额度: {formatQuotaUsd(plan.quota)}</span>
                      ) : null}
                      {(plan.type === 'rpm' || plan.type === 'combo') && plan.rpm > 0 ? (
                        <span>RPM: {plan.rpm}</span>
                      ) : null}
                      {plan.type === 'quota' && plan.quota <= 0 ? <span>-</span> : null}
                    </div>
                  </td>
                  <td>{plan.sort_order}</td>
                  <td>
                    <Chip size="sm" color={plan.enabled ? 'success' : 'danger'}>
                      {plan.enabled ? '启用中' : '已禁用'}
                    </Chip>
                  </td>
                  <td>
                    <div className="flex items-center gap-2">
                      <Button
                        size="sm"
                        variant="flat"
                        color={plan.enabled ? 'warning' : 'success'}
                        startContent={<CircleDashed size={14} />}
                        onPress={() => toggleEnabled(plan)}
                      >
                        {plan.enabled ? '禁用' : '启用'}
                      </Button>
                      <Button
                        size="sm"
                        variant="flat"
                        startContent={<Pencil size={14} />}
                        onPress={() => openEdit(plan)}
                      >
                        编辑
                      </Button>
                      <Button
                        size="sm"
                        variant="flat"
                        color="danger"
                        startContent={<Trash2 size={14} />}
                        onPress={() => deletePlan(plan)}
                      >
                        删除
                      </Button>
                    </div>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      <Modal isOpen={isOpen} onOpenChange={onOpenChange}>
        <ModalContent>
          {(onClose) => (
            <>
              <ModalHeader>{isNew ? '新建订阅套餐' : '编辑订阅套餐'}</ModalHeader>
              <ModalBody>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                  <Input
                    label="套餐名称"
                    placeholder="例如：月度会员"
                    value={editing?.name || ''}
                    onValueChange={(v) => updateField('name', v)}
                    isRequired
                  />
                  <Input
                    label="价格（CNY）"
                    type="number"
                    value={String(editing?.price ?? 0)}
                    onValueChange={(v) => updateField('price', Number(v || 0))}
                  />

                  <Select
                    label="套餐类型"
                    selectedKeys={[editing?.type || 'quota']}
                    onSelectionChange={(keys) => updateField('type', ([...keys][0] as string) || 'quota')}
                  >
                    <SelectItem key="quota">额度包</SelectItem>
                    <SelectItem key="group">分组订阅</SelectItem>
                    <SelectItem key="rpm">高速 RPM</SelectItem>
                    <SelectItem key="combo">组合套餐</SelectItem>
                  </Select>

                  <Input
                    label="有效天数（0=永久）"
                    type="number"
                    value={String(editing?.duration_days ?? 30)}
                    onValueChange={(v) => updateField('duration_days', Number(v || 0))}
                  />

                  {(editing?.type === 'group' || editing?.type === 'combo') && (
                    <Input
                      label="目标分组"
                      placeholder="例如：vip"
                      value={editing?.group_name || ''}
                      onValueChange={(v) => updateField('group_name', v)}
                    />
                  )}

                  {(editing?.type === 'quota' || editing?.type === 'combo') && (
                    <Input
                      label="赠送额度（单位）"
                      type="number"
                      value={String(editing?.quota ?? 0)}
                      onValueChange={(v) => updateField('quota', Number(v || 0))}
                      description="500000 额度 = 1 美元"
                    />
                  )}

                  {(editing?.type === 'rpm' || editing?.type === 'combo') && (
                    <Input
                      label="RPM 上限"
                      type="number"
                      value={String(editing?.rpm ?? 0)}
                      onValueChange={(v) => updateField('rpm', Number(v || 0))}
                    />
                  )}

                  <Input
                    label="排序（小的在前）"
                    type="number"
                    value={String(editing?.sort_order ?? 0)}
                    onValueChange={(v) => updateField('sort_order', Number(v || 0))}
                  />

                  <Select
                    label="状态"
                    selectedKeys={[String(Number(editing?.enabled ?? true))]}
                    onSelectionChange={(keys) => updateField('enabled', ([...keys][0] as string) === '1')}
                  >
                    <SelectItem key="1">启用</SelectItem>
                    <SelectItem key="0">禁用</SelectItem>
                  </Select>
                </div>

                <Input
                  label="描述"
                  placeholder="套餐描述（可选）"
                  value={editing?.description || ''}
                  onValueChange={(v) => updateField('description', v)}
                />
              </ModalBody>
              <ModalFooter>
                <Button variant="flat" onPress={onClose}>取消</Button>
                <Button color="primary" isLoading={saving} onPress={() => savePlan(onClose)}>
                  保存
                </Button>
              </ModalFooter>
            </>
          )}
        </ModalContent>
      </Modal>
    </div>
  );
}
