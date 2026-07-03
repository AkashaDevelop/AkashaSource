import { useState, useEffect } from 'react';
import PageHeader from '../../components/PageHeader';
import EmptyState from '../../components/EmptyState';
import LoadingRows from '../../components/LoadingRows';
import { Card, CardBody, Button, Input, Chip, Pagination } from '../../components/ui';
import { RefreshCw, Filter, CheckCircle, BarChart2 } from 'lucide-react';
import { useAuthStore } from '../../store/auth';
import { toast } from '../../store/toast';
import { confirm } from '../../store/confirm';

interface PayOrder {
  id: number; user_id: number; amount: number; quota_added: number;
  status: number; provider: string; trade_no: string;
  created_at: number; completed_at: number;
}
interface Stats { count: number; total: number; }

const STATUS_MAP: Record<number, { label: string; color: 'warning'|'success'|'danger'|'default' }> = {
  0: { label: '待支付', color: 'warning' },
  1: { label: '成功',   color: 'success' },
  2: { label: '失败',   color: 'danger'  },
  3: { label: '已过期', color: 'default' },
};

// ～管理员的专属小账本，一眼看清所有充值的来龙去脉～
export default function PaymentManagement() {
  const { token } = useAuthStore();
  const [orders, setOrders]     = useState<PayOrder[]>([]);
  const [total, setTotal]       = useState(0);
  const [page, setPage]         = useState(1);
  const [loading, setLoading]   = useState(false);
  const [keyword, setKeyword]   = useState('');
  const [stats, setStats]       = useState<Record<string, Stats>>({});
  const [completing, setCompleting] = useState<number | null>(null);

  const fetchOrders = async (p = page, kw = keyword) => {
    setLoading(true);
    try {
      const res = await fetch(`/api/admin/payment/orders?page=${p}&size=20&keyword=${encodeURIComponent(kw)}`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      const d = await res.json();
      if (d.code === 0) { setOrders(d.data.data || []); setTotal(d.data.total || 0); setPage(p); }
    } finally { setLoading(false); }
  };

  const fetchStats = async () => {
    const res = await fetch('/api/admin/payment/stats', { headers: { Authorization: `Bearer ${token}` } });
    const d = await res.json();
    if (d.code === 0) setStats(d.data);
  };

  useEffect(() => { if (token) { fetchOrders(1); fetchStats(); } }, [token]); // eslint-disable-line

  const handleComplete = async (id: number) => {
    if (!await confirm({ title: '手动补单', message: `确认对订单 #${id} 进行手动补单并入账？此操作不可撤销。`, danger: true })) return;
    setCompleting(id);
    try {
      const res = await fetch(`/api/admin/payment/orders/${id}/complete`, { method: 'POST', headers: { Authorization: `Bearer ${token}` } });
      const d = await res.json();
      if (d.code === 0) { toast.success('补单成功'); fetchOrders(); fetchStats(); }
      else toast.error(d.msg || '补单失败');
    } finally { setCompleting(null); }
  };

  const statCards = [
    { key: 'pending', label: '待支付', colorStyle: { color: '#d97706', background: 'rgba(217,119,6,0.1)' } },
    { key: 'paid',    label: '已成功', colorStyle: { color: '#059669', background: 'rgba(5,150,105,0.1)' } },
    { key: 'failed',  label: '失败',   colorStyle: { color: '#dc2626', background: 'rgba(220,38,38,0.1)'  } },
  ];

  return (
    <div className="space-y-5">
      <PageHeader title="支付管理" description="查看所有充值订单，对卡单进行手动补单" actions={
        <Button size="sm" variant="flat" startContent={<RefreshCw size={14} />} onPress={() => { fetchOrders(1); fetchStats(); }} isLoading={loading}>刷新</Button>
      }/>

      {/* 统计卡片 */}
      <div className="grid grid-cols-3 gap-4">
        {statCards.map(({ key, label, colorStyle }) => {
          const s = (stats[key] as Stats) ?? { count: 0, total: 0 };
          return (
            <Card key={key}>
              <CardBody className="p-4 flex items-center gap-3">
                <div style={{ padding: '10px', borderRadius: 'var(--radius-lg)', ...colorStyle }}>
                  <BarChart2 size={18} />
                </div>
                <div>
                  <p style={{ fontSize: '11px', color: 'var(--text-muted)', margin: 0, fontWeight: 600, textTransform: 'uppercase' }}>{label}</p>
                  <p style={{ fontSize: '18px', fontWeight: 700, margin: 0 }}>{s.count} 笔</p>
                  <p style={{ fontSize: '12px', color: 'var(--text-secondary)', margin: 0 }}>${(s.total || 0).toFixed(2)}</p>
                </div>
              </CardBody>
            </Card>
          );
        })}
      </div>

      {/* 搜索栏 */}
      <div className="flex gap-2">
        <Input size="sm" placeholder="搜索订单号 / 订单 ID" value={keyword} onValueChange={setKeyword} startContent={<Filter size={14} />} className="max-w-xs" />
        <Button size="sm" color="primary" onPress={() => fetchOrders(1, keyword)}>搜索</Button>
        {keyword && <Button size="sm" variant="light" onPress={() => { setKeyword(''); fetchOrders(1, ''); }}>清除</Button>}
      </div>

      {/* 订单表格 */}
      <div className="data-table-wrap">
        <table className="data-table">
          <thead>
            <tr><th>ID</th><th>用户</th><th>渠道</th><th>金额</th><th>到账额度</th><th>交易号</th><th>状态</th><th>创建时间</th><th>操作</th></tr>
          </thead>
          <tbody>
            {loading ? <LoadingRows cols={9} rows={8} /> : orders.length === 0 ? (
              <tr><td colSpan={9}><EmptyState icon="💳" title="暂无订单" description="没有符合条件的充值记录" /></td></tr>
            ) : orders.map(o => {
              const st = STATUS_MAP[o.status] || STATUS_MAP[0];
              return (
                <tr key={o.id}>
                  <td style={{ fontFamily: 'monospace', fontSize: '12px' }}>#{o.id}</td>
                  <td style={{ fontSize: '12px' }}>{o.user_id}</td>
                  <td><Chip size="sm" variant="dot">{o.provider}</Chip></td>
                  <td style={{ fontWeight: 600 }}>${o.amount.toFixed(2)}</td>
                  <td style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>
                    {o.quota_added > 0 ? `$${(o.quota_added/500000).toFixed(4)}` : '-'}
                  </td>
                  <td style={{ fontFamily: 'monospace', fontSize: '11px', color: 'var(--text-muted)', maxWidth: '120px', overflow: 'hidden', textOverflow: 'ellipsis' }}>{o.trade_no || '-'}</td>
                  <td><Chip size="sm" variant="flat" color={st.color}>{st.label}</Chip></td>
                  <td style={{ fontSize: '12px', whiteSpace: 'nowrap' }}>{new Date(o.created_at * 1000).toLocaleString()}</td>
                  <td>
                    {o.status === 0 && (
                      <Button size="sm" color="success" variant="flat" startContent={<CheckCircle size={13} />}
                        isLoading={completing === o.id} onPress={() => handleComplete(o.id)}>
                        补单
                      </Button>
                    )}
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>

      {total > 20 && (
        <div className="flex items-center justify-between gap-3 mt-1">
          <span className="text-sm" style={{ color: 'var(--text-muted)' }}>
            共 <strong style={{ color: 'var(--text-primary)' }}>{total}</strong> 条，第 {page}/{Math.ceil(total / 20)} 页
          </span>
          <Pagination page={page} total={Math.ceil(total / 20)} onChange={p => fetchOrders(p)} />
        </div>
      )}
    </div>
  );
}
