import { useState, useEffect, useRef } from 'react';
import PageHeader from '../../components/PageHeader';
import EmptyState from '../../components/EmptyState';
import LoadingRows from '../../components/LoadingRows';
import { Chip, Tooltip, Button, Input, Snippet, Select, SelectItem, Switch } from '../../components/ui';
import {
  Edit, Trash2, Plus, RefreshCw, Search, X,
  KeyRound, Clock, Wallet, Shield, Power, Users,
  ChevronDown, ChevronUp,
} from 'lucide-react';
import { useAuthStore } from '../../store/auth';
import { toast } from '../../store/toast';
import { confirm } from '../../store/confirm';
import { formatQuota, moneyToQuota, getQuotaPerUnit, getQuotaDisplayHint } from '../../lib/quota';

/* ───── 类型 ───── */
interface Token {
  id: number;
  name: string;
  key: string;
  status: number;
  remain_quota: number;
  unlimited_quota: boolean;
  used_quota: number;
  created_time: number;
  expired_time: number;
  allowed_ips: string;
  allowed_models: string;
  group: string;
  cross_group_retry: boolean;
}

/* ───── 常量 ───── */
const EXPIRY_PRESETS = [
  { label: '永不过期', days: -1 },
  { label: '1 个月',  days: 30  },
  { label: '3 个月',  days: 90  },
  { label: '1 年',    days: 365 },
];

/* ───── 工具函数 ───── */
function addDays(days: number): string {
  const d = new Date(Date.now() + days * 86400000);
  return d.toISOString().slice(0, 16);
}

/* ══════════════════════════════════════
   抽屉：新建 / 编辑令牌
══════════════════════════════════════ */
interface DrawerProps {
  isOpen: boolean;
  onClose: () => void;
  editingToken: Token | null;
  authToken: string | null;
  onSuccess: () => void;
}

function TokenDrawer({ isOpen, onClose, editingToken, authToken, onSuccess }: DrawerProps) {
  const isEdit = !!editingToken;

  /* 表单字段 */
  const [name,           setName]           = useState('');
  const [status,         setStatus]         = useState(1);
  const [batchCount,     setBatchCount]     = useState(1);
  const [neverExpire,    setNeverExpire]    = useState(true);
  const [expiredTime,    setExpiredTime]    = useState('');
  const [unlimitedQuota, setUnlimitedQuota] = useState(true);
  const [remainQuota,    setRemainQuota]    = useState('10');
  const [allowedIps,     setAllowedIps]     = useState('');
  const [allowedModels,  setAllowedModels]  = useState<string[]>([]);
  const [,               setNote]           = useState('');
  const [group,          setGroup]          = useState('');
  const [crossGroupRetry, setCrossGroupRetry] = useState(false);

  /* 分组选项：拉取当前用户能用的分组 + 展示倍率 */
  const [groupOptions, setGroupOptions] = useState<Record<string, { ratio: number | string; desc: string }>>({});

  /* 模型选择 */
  const [models,       setModels]       = useState<string[]>([]);
  const [modelSearch,  setModelSearch]  = useState('');
  const [modelPickerOpen, setModelPickerOpen] = useState(false);

  /* 高级设置折叠 */
  const [advancedOpen, setAdvancedOpen] = useState(false);

  const [saving,   setSaving]  = useState(false);
  const [nameErr,  setNameErr] = useState('');

  const pickerRef = useRef<HTMLDivElement>(null);

  /* 关闭 model picker（点外） */
  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (pickerRef.current && !pickerRef.current.contains(e.target as Node)) {
        setModelPickerOpen(false);
      }
    };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, []);

  /* 拉取可用模型 */
  useEffect(() => {
    if (!isOpen || !authToken) return;
    fetch('/v1/models', { headers: { Authorization: `Bearer ${authToken}` } })
      .then(r => r.json())
      .then(d => { if (d.data?.length) setModels(d.data.map((m: any) => m.id).sort()); })
      .catch(() => {});
  }, [isOpen, authToken]);

  /* 拉取当前用户可用的分组，给令牌单独挑一个分组用～ */
  useEffect(() => {
    if (!isOpen || !authToken) return;
    fetch('/api/user/self/groups', { headers: { Authorization: `Bearer ${authToken}` } })
      .then(r => r.json())
      .then(d => { if (d.code === 0 && d.data) setGroupOptions(d.data); })
      .catch(() => {});
  }, [isOpen, authToken]);

  /* 重置 / 填充表单 */
  useEffect(() => {
    if (!isOpen) return;
    if (isEdit && editingToken) {
      setName(editingToken.name);
      setStatus(editingToken.status);
      setNeverExpire(editingToken.expired_time === -1);
      setExpiredTime(
        editingToken.expired_time === -1
          ? ''
          : new Date(editingToken.expired_time * 1000).toISOString().slice(0, 16)
      );
      setUnlimitedQuota(editingToken.unlimited_quota);
      setRemainQuota(editingToken.unlimited_quota ? '10' : (editingToken.remain_quota / getQuotaPerUnit()).toFixed(2));
      setAllowedIps(editingToken.allowed_ips || '');
      setAllowedModels(editingToken.allowed_models ? editingToken.allowed_models.split(',').filter(Boolean) : []);
      setGroup(editingToken.group || '');
      setCrossGroupRetry(editingToken.cross_group_retry || false);
      setNote('');
    } else {
      setName(''); setStatus(1); setBatchCount(1);
      setNeverExpire(true); setExpiredTime('');
      setUnlimitedQuota(true); setRemainQuota('10');
      setAllowedIps(''); setAllowedModels([]); setNote('');
      setGroup(''); setCrossGroupRetry(false);
    }
    setNameErr(''); setAdvancedOpen(false); setModelSearch('');
  }, [isOpen, isEdit, editingToken]);

  /* 快速设置过期时间 */
  const applyPreset = (days: number) => {
    if (days === -1) {
      setNeverExpire(true);
      setExpiredTime('');
    } else {
      setNeverExpire(false);
      setExpiredTime(addDays(days));
    }
  };

  /* 提交 */
  const handleSubmit = async () => {
    if (!name.trim()) { setNameErr('令牌名称不能为空'); return; }
    setNameErr('');
    setSaving(true);

    const expTime = neverExpire ? -1 : (expiredTime ? Math.floor(new Date(expiredTime).getTime() / 1000) : -1);
    const payload: any = {
      name: name.trim(),
      status,
      expired_time: expTime,
      unlimited_quota: unlimitedQuota,
      remain_quota: unlimitedQuota ? 0 : moneyToQuota(parseFloat(remainQuota || '0')),
      allowed_ips: allowedIps.trim(),
      allowed_models: allowedModels.join(','),
      group: group,
      cross_group_retry: group === 'auto' ? crossGroupRetry : false,
    };

    try {
      if (isEdit) {
        payload.id = editingToken!.id;
        const res  = await fetch('/api/token', {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${authToken}` },
          body: JSON.stringify(payload),
        });
        const data = await res.json();
        if (data.code !== 0) throw new Error(data.msg || '更新失败');
        toast.success('令牌已更新');
      } else {
        const count = Math.max(1, Math.min(batchCount, 20));
        const names = count === 1
          ? [payload.name]
          : Array.from({ length: count }, (_, i) => `${payload.name}-${String(i + 1).padStart(2, '0')}`);

        await Promise.all(names.map(n =>
          fetch('/api/token', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${authToken}` },
            body: JSON.stringify({ ...payload, name: n }),
          }).then(r => r.json())
        ));
        toast.success(count > 1 ? `成功创建 ${count} 个令牌` : '令牌已创建');
      }
      onSuccess();
      onClose();
    } catch (err: any) {
      toast.error(err.message || '操作失败');
    } finally {
      setSaving(false);
    }
  };

  const filteredModels = models.filter(m =>
    m.toLowerCase().includes(modelSearch.toLowerCase()) && !allowedModels.includes(m)
  );

  const toggleModel = (m: string) => {
    setAllowedModels(prev =>
      prev.includes(m) ? prev.filter(x => x !== m) : [...prev, m]
    );
  };

  /* ── 渲染 ── */
  return (
    <>
      {/* 背景遮罩 */}
      {isOpen && (
        <div className="token-drawer-backdrop" onClick={onClose} aria-hidden="true" />
      )}

      {/* 抽屉面板 */}
      <div className={`token-drawer${isOpen ? ' open' : ''}`} role="dialog" aria-modal="true">

        {/* 标题栏 */}
        <div className="token-drawer-header">
          <div className="flex items-center gap-2.5">
            <div style={{ width: '32px', height: '32px', borderRadius: 'var(--radius-lg)', background: 'var(--color-info-bg)', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
              <KeyRound size={16} style={{ color: 'var(--accent-primary)' }} />
            </div>
            <div>
              <h2 style={{ fontSize: '15px', fontWeight: 700, color: 'var(--text-primary)', margin: 0 }}>
                {isEdit ? '编辑令牌' : '新建令牌'}
              </h2>
              <p style={{ fontSize: '11px', color: 'var(--text-muted)', margin: 0 }}>
                {isEdit ? `ID: ${editingToken?.id}` : 'API 访问密钥配置'}
              </p>
            </div>
          </div>
          <button onClick={onClose} style={{ background: 'none', border: 'none', cursor: 'pointer', color: 'var(--text-muted)', padding: '4px' }} aria-label="关闭">
            <X size={20} />
          </button>
        </div>

        {/* 滚动内容 */}
        <div className="token-drawer-body">

          {/* ── 基本信息 ── */}
          <div className="token-drawer-section">
            <div className="token-drawer-section-title">
              <KeyRound size={14} style={{ color: 'var(--accent-primary)' }} />
              基本信息
            </div>
            <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
              {/* 令牌名称 */}
              <div>
                <Input
                  label="令牌名称"
                  placeholder="如：生产环境密钥"
                  value={name}
                  onValueChange={v => { setName(v); if (v.trim()) setNameErr(''); }}
                  isRequired
                  isInvalid={!!nameErr}
                />
                {nameErr && <p className="field-error">{nameErr}</p>}
              </div>

              {/* 编辑时显示状态开关 */}
              {isEdit && (
                <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '10px 14px', borderRadius: 'var(--radius-lg)', background: 'var(--bg-elevated)', border: '1px solid var(--border-color)' }}>
                  <div>
                    <p style={{ fontSize: '13px', fontWeight: 500, color: 'var(--text-primary)', margin: 0 }}>启用令牌</p>
                    <p style={{ fontSize: '11px', color: 'var(--text-muted)', margin: 0 }}>禁用后立即停止服务</p>
                  </div>
                  <button
                    onClick={() => setStatus(s => s === 1 ? 2 : 1)}
                    style={{
                      width: '44px', height: '24px', borderRadius: '12px', border: 'none', cursor: 'pointer', position: 'relative',
                      background: status === 1 ? 'linear-gradient(135deg, var(--accent-primary), var(--accent-cosmic))' : 'var(--bg-elevated)',
                      transition: 'background 0.2s',
                    }}
                    aria-checked={status === 1}
                    role="switch"
                  >
                    <span style={{
                      position: 'absolute', top: '2px', width: '20px', height: '20px', borderRadius: '50%',
                      background: 'white', boxShadow: '0 1px 4px rgba(0,0,0,0.3)',
                      transition: 'left 0.2s cubic-bezier(0.34,1.56,0.64,1)',
                      left: status === 1 ? '22px' : '2px',
                    }} />
                  </button>
                </div>
              )}

              {/* 新建时显示批量创建 */}
              {!isEdit && (
                <div style={{ display: 'flex', alignItems: 'center', gap: '12px', padding: '10px 14px', borderRadius: 'var(--radius-lg)', background: 'var(--bg-elevated)', border: '1px solid var(--border-color)' }}>
                  <div style={{ flex: 1 }}>
                    <p style={{ fontSize: '13px', fontWeight: 500, color: 'var(--text-primary)', margin: 0 }}>批量创建</p>
                    <p style={{ fontSize: '11px', color: 'var(--text-muted)', margin: 0 }}>多个令牌自动添加编号后缀</p>
                  </div>
                  <div className="flex items-center gap-2">
                    <button onClick={() => setBatchCount(v => Math.max(1, v - 1))} style={{ width: '28px', height: '28px', borderRadius: 'var(--radius-md)', border: '1px solid var(--border-color)', background: 'var(--bg-surface)', cursor: 'pointer', color: 'var(--text-primary)', fontSize: '16px', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>−</button>
                    <span style={{ width: '32px', textAlign: 'center', fontSize: '14px', fontWeight: 600, color: 'var(--text-primary)' }}>{batchCount}</span>
                    <button onClick={() => setBatchCount(v => Math.min(20, v + 1))} style={{ width: '28px', height: '28px', borderRadius: 'var(--radius-md)', border: '1px solid var(--border-color)', background: 'var(--bg-surface)', cursor: 'pointer', color: 'var(--text-primary)', fontSize: '16px', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>+</button>
                  </div>
                </div>
              )}
            </div>
          </div>

          {/* ── 有效期 ── */}
          <div className="token-drawer-section">
            <div className="token-drawer-section-title">
              <Clock size={14} style={{ color: 'var(--accent-cosmic)' }} />
              有效期
            </div>
            {/* 快速预设 */}
            <div className="flex flex-wrap gap-2 mb-3">
              {EXPIRY_PRESETS.map(p => (
                <button
                  key={p.label}
                  className={`expiry-preset-btn${(p.days === -1 ? neverExpire : (!neverExpire && expiredTime === addDays(p.days))) ? ' active' : ''}`}
                  onClick={() => applyPreset(p.days)}
                >
                  {p.label}
                </button>
              ))}
            </div>
            {/* 自定义时间 */}
            {!neverExpire && (
              <div>
                <label style={{ fontSize: '12px', color: 'var(--text-muted)', display: 'block', marginBottom: '6px' }}>自定义过期时间</label>
                <input
                  type="datetime-local"
                  className="akasha-date-input"
                  style={{ width: '100%' }}
                  value={expiredTime}
                  onChange={e => setExpiredTime(e.target.value)}
                />
              </div>
            )}
            {!neverExpire && expiredTime && (
              <p style={{ fontSize: '11px', color: 'var(--text-muted)', marginTop: '6px' }}>
                到期：{new Date(expiredTime).toLocaleString('zh-CN')}
              </p>
            )}
          </div>

          {/* ── 额度配置 ── */}
          <div className="token-drawer-section">
            <div className="token-drawer-section-title">
              <Wallet size={14} style={{ color: 'var(--accent-star)' }} />
              额度配置
            </div>
            {/* 无限额度开关 */}
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '10px 14px', borderRadius: 'var(--radius-lg)', background: 'var(--bg-elevated)', border: '1px solid var(--border-color)', marginBottom: '12px' }}>
              <div>
                <p style={{ fontSize: '13px', fontWeight: 500, color: 'var(--text-primary)', margin: 0 }}>无限额度</p>
                <p style={{ fontSize: '11px', color: 'var(--text-muted)', margin: 0 }}>不限制此令牌的用量</p>
              </div>
              <button
                onClick={() => setUnlimitedQuota(v => !v)}
                style={{
                  width: '44px', height: '24px', borderRadius: '12px', border: 'none', cursor: 'pointer', position: 'relative',
                  background: unlimitedQuota ? 'linear-gradient(135deg, var(--accent-primary), var(--accent-cosmic))' : 'var(--bg-elevated)',
                  transition: 'background 0.2s',
                }}
                aria-checked={unlimitedQuota} role="switch"
              >
                <span style={{
                  position: 'absolute', top: '2px', width: '20px', height: '20px', borderRadius: '50%',
                  background: 'white', boxShadow: '0 1px 4px rgba(0,0,0,0.3)',
                  transition: 'left 0.2s cubic-bezier(0.34,1.56,0.64,1)',
                  left: unlimitedQuota ? '22px' : '2px',
                }} />
              </button>
            </div>

            {/* 固定额度输入 */}
            {!unlimitedQuota && (
              <Input
                type="number"
                label="额度上限（USD）"
                placeholder="例如：10"
                value={remainQuota}
                onValueChange={setRemainQuota}
                description={`该令牌可消耗的最大金额，${getQuotaDisplayHint()}`}
                startContent={<span style={{ fontSize: '13px', color: 'var(--text-muted)' }}>$</span>}
              />
            )}
          </div>

          {/* ── 分组 ── */}
          <div className="token-drawer-section">
            <div className="token-drawer-section-title">
              <Users size={14} style={{ color: 'var(--accent-primary)' }} />
              分组
            </div>
            <Select
              label="令牌分组"
              placeholder="跟随账号默认分组"
              selectedKeys={group ? [group] : []}
              onSelectionChange={keys => setGroup([...keys][0] || '')}
            >
              {Object.entries(groupOptions).map(([name, info]) => (
                <SelectItem key={name} value={name}>
                  {name === 'auto' ? '自动（按顺序尝试候选分组）' : `${name}（倍率 ${info.ratio}）`}
                </SelectItem>
              ))}
            </Select>
            {group === 'auto' && (
              <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '10px 14px', marginTop: '10px', borderRadius: 'var(--radius-lg)', background: 'var(--bg-elevated)', border: '1px solid var(--border-color)' }}>
                <div>
                  <p style={{ fontSize: '13px', fontWeight: 500, color: 'var(--text-primary)', margin: 0 }}>跨分组重试</p>
                  <p style={{ fontSize: '11px', color: 'var(--text-muted)', margin: 0 }}>当前候选分组重试用尽后，是否自动换下一个分组接着试</p>
                </div>
                <Switch isSelected={crossGroupRetry} onValueChange={setCrossGroupRetry} size="sm" />
              </div>
            )}
          </div>

          {/* ── 访问控制（高级，可折叠） ── */}
          <div className="token-drawer-section">
            <button
              className="token-drawer-section-title w-full"
              style={{ justifyContent: 'space-between', background: 'none', border: 'none', cursor: 'pointer' }}
              onClick={() => setAdvancedOpen(v => !v)}
            >
              <span className="flex items-center gap-2" style={{ fontSize: '13px', fontWeight: 600, color: 'var(--text-primary)' }}>
                <Shield size={14} style={{ color: 'var(--color-success-fg)' }} />
                访问控制
                {(allowedModels.length > 0 || allowedIps.trim()) && (
                  <span style={{ fontSize: '11px', padding: '1px 6px', borderRadius: 'var(--radius-full)', background: 'var(--color-info-bg)', color: 'var(--accent-primary)' }}>
                    已配置
                  </span>
                )}
              </span>
              {advancedOpen ? <ChevronUp size={16} style={{ color: 'var(--text-muted)' }} /> : <ChevronDown size={16} style={{ color: 'var(--text-muted)' }} />}
            </button>

            {advancedOpen && (
              <div style={{ display: 'flex', flexDirection: 'column', gap: '16px', marginTop: '4px' }}>

                {/* 模型限制 */}
                <div>
                  <label style={{ fontSize: '12px', fontWeight: 500, color: 'var(--text-primary)', display: 'block', marginBottom: '6px' }}>
                    模型限制
                    <span style={{ fontSize: '11px', color: 'var(--text-muted)', fontWeight: 400, marginLeft: '6px' }}>留空则允许全部模型</span>
                  </label>

                  {/* 已选标签 */}
                  {allowedModels.length > 0 && (
                    <div className="flex flex-wrap gap-1.5 mb-2">
                      {allowedModels.map(m => (
                        <span key={m} className="model-tag" onClick={() => toggleModel(m)} title="点击移除">
                          {m} <X size={10} />
                        </span>
                      ))}
                    </div>
                  )}

                  {/* 模型搜索 + 下拉 */}
                  <div ref={pickerRef} style={{ position: 'relative' }}>
                    <input
                      className="akasha-date-input"
                      style={{ width: '100%', paddingLeft: '36px' }}
                      placeholder={models.length ? '搜索并选择模型...' : '加载模型中...'}
                      value={modelSearch}
                      onChange={e => setModelSearch(e.target.value)}
                      onFocus={() => setModelPickerOpen(true)}
                    />
                    <Search size={13} style={{ position: 'absolute', left: '12px', top: '50%', transform: 'translateY(-50%)', color: 'var(--text-muted)', pointerEvents: 'none' }} />

                    {modelPickerOpen && (
                      <div style={{
                        position: 'absolute', top: 'calc(100% + 4px)', left: 0, right: 0, zIndex: 10,
                        background: 'var(--bg-elevated)', border: '1px solid var(--border-color)',
                        borderRadius: 'var(--radius-lg)', boxShadow: 'var(--shadow-hover)',
                        maxHeight: '200px', overflowY: 'auto',
                      }}>
                        {filteredModels.length === 0 ? (
                          <p style={{ padding: '16px', textAlign: 'center', fontSize: '12px', color: 'var(--text-muted)', margin: 0 }}>
                            {modelSearch ? '无匹配模型' : '全部模型已选'}
                          </p>
                        ) : filteredModels.map(m => (
                          <button
                            key={m}
                            style={{
                              width: '100%', textAlign: 'left', padding: '8px 14px',
                              fontSize: '12px', fontFamily: 'monospace',
                              color: 'var(--text-primary)', background: 'none', border: 'none',
                              cursor: 'pointer', display: 'block',
                            }}
                            onMouseEnter={e => { (e.currentTarget as HTMLElement).style.background = 'var(--nav-hover-bg)'; }}
                            onMouseLeave={e => { (e.currentTarget as HTMLElement).style.background = 'none'; }}
                            onMouseDown={e => {
                              e.preventDefault();
                              toggleModel(m);
                              setModelSearch('');
                            }}
                          >
                            {m}
                          </button>
                        ))}
                      </div>
                    )}
                  </div>
                </div>

                {/* IP 白名单 */}
                <div>
                  <label style={{ fontSize: '12px', fontWeight: 500, color: 'var(--text-primary)', display: 'block', marginBottom: '6px' }}>
                    IP 白名单
                    <span style={{ fontSize: '11px', color: 'var(--text-muted)', fontWeight: 400, marginLeft: '6px' }}>每行一个，留空则不限制</span>
                  </label>
                  <textarea
                    value={allowedIps}
                    onChange={e => setAllowedIps(e.target.value)}
                    placeholder={'192.168.1.0/24\n10.0.0.1'}
                    rows={4}
                    style={{
                      width: '100%', padding: '8px 12px',
                      borderRadius: 'var(--radius-md)', border: '1px solid var(--border-color)',
                      background: 'var(--bg-elevated)', color: 'var(--text-primary)',
                      fontSize: '12px', fontFamily: 'monospace', lineHeight: 1.6, resize: 'vertical',
                      outline: 'none', transition: 'border-color 0.15s',
                    }}
                    onFocus={e => { e.currentTarget.style.borderColor = 'var(--accent-primary)'; }}
                    onBlur={e => { e.currentTarget.style.borderColor = 'var(--border-color)'; }}
                  />
                  {allowedIps.trim() && (
                    <div style={{ marginTop: '6px', padding: '7px 10px', borderRadius: 'var(--radius-md)', background: 'var(--color-warning-bg)', border: '1px solid var(--color-warning-bg)' }}>
                      <p style={{ fontSize: '11px', color: 'var(--color-warning-fg)', margin: 0 }}>
                        ⚠ 启用 IP 限制后，仅白名单内的地址可使用该令牌
                      </p>
                    </div>
                  )}
                </div>
              </div>
            )}
          </div>
        </div>

        {/* 底部操作栏 */}
        <div className="token-drawer-footer">
          <button
            onClick={onClose}
            style={{ padding: '8px 18px', borderRadius: 'var(--radius-lg)', border: '1px solid var(--border-color)', background: 'var(--bg-elevated)', color: 'var(--text-secondary)', fontSize: '13px', cursor: 'pointer', fontWeight: 500 }}
          >
            取消
          </button>
          <button
            onClick={handleSubmit}
            disabled={saving}
            style={{
              padding: '8px 20px', borderRadius: 'var(--radius-lg)', border: 'none',
              background: saving ? 'var(--bg-elevated)' : 'linear-gradient(135deg, var(--accent-primary), var(--accent-cosmic))',
              color: saving ? 'var(--text-muted)' : 'white',
              fontSize: '13px', fontWeight: 600, cursor: saving ? 'not-allowed' : 'pointer',
              display: 'flex', alignItems: 'center', gap: '6px',
              transition: 'opacity 0.15s',
            }}
          >
            {saving && <span style={{ width: '14px', height: '14px', border: '2px solid rgba(255,255,255,0.3)', borderTopColor: 'white', borderRadius: '50%', display: 'inline-block', animation: 'spin 0.8s linear infinite' }} />}
            {saving ? '保存中...' : (isEdit ? '保存修改' : `创建令牌${batchCount > 1 ? ` (${batchCount})` : ''}`)}
          </button>
        </div>
      </div>
    </>
  );
}

/* ══════════════════════════════════════
   主页面
══════════════════════════════════════ */
export default function TokenManagement() {
  const [tokens,       setTokens]       = useState<Token[]>([]);
  const [loading,      setLoading]      = useState(false);
  const [search,       setSearch]       = useState('');
  const [drawerOpen,   setDrawerOpen]   = useState(false);
  const [editingToken, setEditingToken] = useState<Token | null>(null);
  const { token: authToken } = useAuthStore();

  const fetchTokens = async () => {
    setLoading(true);
    try {
      const res  = await fetch('/api/token', { headers: { Authorization: `Bearer ${authToken}` } });
      const data = await res.json();
      if (data.code === 0) setTokens(data.data || []);
    } catch (e) { console.error(e); }
    finally { setLoading(false); }
  };

  useEffect(() => { if (authToken) fetchTokens(); }, [authToken]);

  const handleAdd = () => {
    setEditingToken(null);
    setDrawerOpen(true);
  };

  const handleEdit = (token: Token) => {
    setEditingToken(token);
    setDrawerOpen(true);
  };

  const handleDelete = async (id: number) => {
    if (!await confirm({ title: '删除令牌', message: '确定要删除这个令牌吗？删除后无法恢复，所有使用该密钥的请求将立即失败。', danger: true })) return;
    try {
      const res  = await fetch(`/api/token/${id}`, { method: 'DELETE', headers: { Authorization: `Bearer ${authToken}` } });
      const data = await res.json();
      if (data.code === 0) { fetchTokens(); toast.success('令牌已删除'); }
      else toast.error(data.msg || '删除失败');
    } catch { toast.error('请求失败'); }
  };

  const handleStatusToggle = async (token: Token) => {
    const newStatus = token.status === 1 ? 2 : 1;
    setTokens(prev => prev.map(t => t.id === token.id ? { ...t, status: newStatus } : t));
    try {
      await fetch('/api/token', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${authToken}` },
        body: JSON.stringify({ ...token, status: newStatus }),
      });
    } catch { fetchTokens(); }
  };

  const filtered = tokens.filter(t => !search || t.name.toLowerCase().includes(search.toLowerCase()));

  return (
    <div className="space-y-5">
      <PageHeader
        title="令牌管理"
        description="管理您的 API 访问密钥"
        actions={
          <div className="flex gap-2 items-center">
            <Input
              placeholder="搜索令牌..."
              size="sm"
              value={search}
              onValueChange={setSearch}
              startContent={<Search size={14} />}
              className="w-36"
            />
            <Button startContent={<RefreshCw size={15} />} onPress={fetchTokens} variant="flat" size="sm" isLoading={loading}>
              刷新
            </Button>
            <Button
              startContent={<Plus size={15} />}
              color="primary"
              size="sm"
              onPress={handleAdd}
              style={{ background: 'linear-gradient(135deg, var(--accent-primary), var(--accent-cosmic))', color: 'white' }}
            >
              新建令牌
            </Button>
          </div>
        }
      />

      <div className="data-table-wrap">
        <table className="data-table">
          <thead>
            <tr>
              <th>名称</th>
              <th>状态</th>
              <th>分组</th>
              <th>已用</th>
              <th>剩余</th>
              <th>模型限制</th>
              <th>过期时间</th>
              <th>密钥</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <LoadingRows cols={9} rows={5} />
            ) : filtered.length === 0 ? (
              <tr><td colSpan={9}>
                <EmptyState
                  icon={<KeyRound size={24} />}
                  title={search ? '未找到匹配的令牌' : '暂无令牌'}
                  description={search ? undefined : '点击「新建令牌」创建您的第一个 API 访问密钥'}
                  variant={search ? 'search' : 'default'}
                  action={!search ? <Button size="sm" color="primary" onPress={handleAdd} startContent={<Plus size={14} />}>新建令牌</Button> : undefined}
                />
              </td></tr>
            ) : filtered.map(token => (
              <tr key={token.id}>
                <td style={{ fontWeight: 500 }}>{token.name}</td>
                <td>
                  <Chip
                    size="sm"
                    variant="flat"
                    color={token.status === 1 ? 'success' : 'danger'}
                    className="cursor-pointer"
                    onClick={() => handleStatusToggle(token)}
                    startContent={<Power size={10} />}
                  >
                    {token.status === 1 ? '启用' : '禁用'}
                  </Chip>
                </td>
                <td>
                  {token.group
                    ? <Chip size="sm" variant="dot">{token.group}</Chip>
                    : <span style={{ fontSize: '11px', color: 'var(--text-faint)' }}>默认</span>
                  }
                </td>
                <td style={{ fontSize: '12px' }}>{formatQuota(token.used_quota)}</td>
                <td>
                  {token.unlimited_quota
                    ? <Chip size="sm" variant="dot" color="success">无限</Chip>
                    : <span style={{ fontSize: '12px' }}>{formatQuota(token.remain_quota)}</span>
                  }
                </td>
                <td style={{ maxWidth: '120px' }}>
                  {token.allowed_models
                    ? <span style={{ fontSize: '11px', color: 'var(--text-muted)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', display: 'block' }}>
                        {token.allowed_models.split(',').length} 个模型
                      </span>
                    : <span style={{ fontSize: '11px', color: 'var(--text-faint)' }}>不限制</span>
                  }
                </td>
                <td style={{ whiteSpace: 'nowrap', fontSize: '12px' }}>
                  {token.expired_time === -1
                    ? <Chip size="sm" variant="flat">永不过期</Chip>
                    : <span style={{ color: Date.now() > token.expired_time * 1000 ? 'var(--color-danger-fg)' : 'inherit' }}>
                        {new Date(token.expired_time * 1000).toLocaleDateString()}
                      </span>
                  }
                </td>
                <td>
                  <Snippet symbol="" codeString={token.key} color="default" size="sm">
                    {token.key.slice(0, 8) + '…' + token.key.slice(-4)}
                  </Snippet>
                </td>
                <td>
                  <div className="flex items-center gap-2">
                    <Tooltip content="编辑配置">
                      <span className="cursor-pointer" style={{ color: 'var(--text-muted)' }} onClick={() => handleEdit(token)}>
                        <Edit size={15} />
                      </span>
                    </Tooltip>
                    <Tooltip content="删除令牌">
                      <span className="cursor-pointer" style={{ color: 'var(--color-danger-fg)' }} onClick={() => handleDelete(token.id)}>
                        <Trash2 size={15} />
                      </span>
                    </Tooltip>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* 令牌配置抽屉 */}
      <TokenDrawer
        isOpen={drawerOpen}
        onClose={() => setDrawerOpen(false)}
        editingToken={editingToken}
        authToken={authToken}
        onSuccess={fetchTokens}
      />
    </div>
  );
}
