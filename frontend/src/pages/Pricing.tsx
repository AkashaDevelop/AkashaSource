import { useState, useEffect } from 'react';
import PageHeader from '../components/PageHeader';
import EmptyState from '../components/EmptyState';
import LoadingRows from '../components/LoadingRows';
import { Card, CardBody, Input, Chip } from '../components/ui';
import { Search } from 'lucide-react';

interface ModelPrice {
  model: string;
  input_ratio: number;
  output_ratio: number;
  upstream_input_price: number;
  upstream_output_price: number;
  actual_input_price: number;
  actual_output_price: number;
}

function getRatioColor(ratio: number): string {
  if (ratio <= 1) return '#34d399';
  if (ratio <= 5) return 'var(--accent-cosmic)';
  if (ratio <= 15) return '#fbbf24';
  return '#f87171';
}

export default function Pricing() {
  const [models, setModels] = useState<ModelPrice[]>([]);
  const [loading, setLoading] = useState(false);
  const [search, setSearch] = useState('');

  useEffect(() => {
    setLoading(true);
    fetch('/api/pricing')
      .then(r => r.json())
      .then(d => {
        if (d.code === 0 && d.data?.models) {
          setModels(d.data.models);
        } else if (d.models) {
          setModels(d.models);
        }
      })
      .catch(console.error)
      .finally(() => setLoading(false));
  }, []);

  const filtered = models.filter(m =>
    m.model.toLowerCase().includes(search.toLowerCase())
  );

  return (
    <div className="space-y-6">
      <PageHeader title="模型定价" description="所有可用模型的计费倍率一览" />

      <Card>
        <CardBody className="p-5">
          <div className="flex items-center gap-3 mb-4">
            <Input
              placeholder="搜索模型名称..."
              value={search}
              onValueChange={setSearch}
              startContent={<Search size={16} className="text-default-400" />}
              style={{ maxWidth: '320px' }}
            />
            {!loading && (
              <span style={{ fontSize: '13px', color: 'var(--text-muted)' }}>
                共 {filtered.length} 个模型
              </span>
            )}
          </div>

          {/* 倍率说明 */}
          <div style={{
            display: 'flex', gap: '16px', flexWrap: 'wrap',
            padding: '10px 14px', borderRadius: '10px',
            background: 'var(--bg-elevated)', border: '1px solid var(--border-color)',
            marginBottom: '16px', fontSize: '12px', color: 'var(--text-muted)',
          }}>
            <span>倍率说明：</span>
            {[
              { label: '≤1x 极低', color: '#34d399' },
              { label: '≤5x 低', color: 'var(--accent-cosmic)' },
              { label: '≤15x 中', color: '#fbbf24' },
              { label: '>15x 高', color: '#f87171' },
            ].map(item => (
              <span key={item.label} style={{ color: item.color, fontWeight: 600 }}>{item.label}</span>
            ))}
          </div>

          <div className="data-table-wrap">
            <table className="data-table">
              <thead>
                <tr>
                  <th>模型名称</th>
                  <th>输入倍率</th>
                  <th>输出倍率</th>
                  <th>上游输入价格</th>
                  <th>上游输出价格</th>
                  <th>实际输入价格</th>
                  <th>实际输出价格</th>
                </tr>
              </thead>
              <tbody>
                {loading ? (
                  <LoadingRows cols={7} rows={8} />
                ) : filtered.length === 0 ? (
                  <tr><td colSpan={7}><EmptyState icon="🤖" title="暂无定价数据" description={search ? `未找到包含「${search}」的模型` : '暂时没有可用的模型定价'} /></td></tr>
                ) : filtered.map((m) => (
                  <tr key={m.model}>
                    <td>
                      <span style={{ fontFamily: 'monospace', fontSize: '13px', fontWeight: 500 }}>{m.model}</span>
                    </td>
                    <td>
                      <Chip
                        size="sm"
                        variant="flat"
                        style={{ color: getRatioColor(m.input_ratio), background: `${getRatioColor(m.input_ratio)}18`, fontWeight: 600 }}
                      >
                        {m.input_ratio}x
                      </Chip>
                    </td>
                    <td>
                      <Chip
                        size="sm"
                        variant="flat"
                        style={{ color: getRatioColor(m.output_ratio), background: `${getRatioColor(m.output_ratio)}18`, fontWeight: 600 }}
                      >
                        {m.output_ratio}x
                      </Chip>
                    </td>
                    <td>
                      <span style={{ fontSize: '13px' }}>
                        {m.upstream_input_price > 0 ? `¥${m.upstream_input_price.toFixed(2)}/M` : '-'}
                      </span>
                    </td>
                    <td>
                      <span style={{ fontSize: '13px' }}>
                        {m.upstream_output_price > 0 ? `¥${m.upstream_output_price.toFixed(2)}/M` : '-'}
                      </span>
                    </td>
                    <td>
                      <span style={{ fontSize: '13px', fontWeight: 600, color: 'var(--accent-cosmic)' }}>
                        {m.actual_input_price > 0 ? `¥${m.actual_input_price.toFixed(2)}/M` : '-'}
                      </span>
                    </td>
                    <td>
                      <span style={{ fontSize: '13px', fontWeight: 600, color: 'var(--accent-cosmic)' }}>
                        {m.actual_output_price > 0 ? `¥${m.actual_output_price.toFixed(2)}/M` : '-'}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </CardBody>
      </Card>
    </div>
  );
}
