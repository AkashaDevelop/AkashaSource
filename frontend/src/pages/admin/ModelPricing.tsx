// 模型定价管理喵～一个模型一个面板，把倍率、缓存、图像、音频、按次计费全部集中在这里编辑
// 保存后后端会自动重新生成计费引擎需要的全部倍率表，不用再手动点"同步定价"啦～
import { useState, useEffect, useMemo } from 'react';
import { Card, CardBody, Button, Input, Switch, Chip, Checkbox } from '../../components/ui';
import EmptyState from '../../components/EmptyState';
import { Search, Save, RefreshCw, Copy, AlertCircle } from 'lucide-react';
import { useAuthStore } from '../../store/auth';
import { toast } from '../../store/toast';

interface ModelConfig {
  id: number;
  model_name: string;
  display_name: string;
  category: string;
  input_ratio: number;
  output_ratio: number;
  cache_ratio: number;
  image_ratio: number;
  audio_ratio: number;
  audio_completion_ratio: number;
  is_fixed_price: boolean;
  fixed_price: number;
  upstream_input_price: number;
  upstream_output_price: number;
  max_context: number;
  enabled: boolean;
}

// 定价字段仍是默认值，大概率是"从没配置过"，用来驱动"仅显示未配置价格"筛选～
function isUnsetPricing(m: ModelConfig): boolean {
  return (
    m.input_ratio === 1 &&
    m.output_ratio === 1 &&
    m.cache_ratio === 0.5 &&
    m.image_ratio === 1 &&
    m.audio_ratio === 1 &&
    m.audio_completion_ratio === 1 &&
    !m.is_fixed_price
  );
}

type PricingForm = {
  input_ratio: string;
  output_ratio: string;
  cache_ratio: string;
  image_ratio: string;
  audio_ratio: string;
  audio_completion_ratio: string;
  is_fixed_price: boolean;
  fixed_price: string;
  upstream_input_price: string;
  upstream_output_price: string;
};

function toForm(m: ModelConfig): PricingForm {
  return {
    input_ratio: String(m.input_ratio),
    output_ratio: String(m.output_ratio),
    cache_ratio: String(m.cache_ratio),
    image_ratio: String(m.image_ratio),
    audio_ratio: String(m.audio_ratio),
    audio_completion_ratio: String(m.audio_completion_ratio),
    is_fixed_price: m.is_fixed_price,
    fixed_price: String(m.fixed_price),
    upstream_input_price: String(m.upstream_input_price),
    upstream_output_price: String(m.upstream_output_price),
  };
}

export default function ModelPricing() {
  const { token } = useAuthStore();
  const [models, setModels] = useState<ModelConfig[]>([]);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [syncing, setSyncing] = useState(false);
  const [keyword, setKeyword] = useState('');
  const [onlyUnset, setOnlyUnset] = useState(false);
  const [selectedId, setSelectedId] = useState<number | null>(null);
  const [form, setForm] = useState<PricingForm | null>(null);
  const [applyTargets, setApplyTargets] = useState<number[]>([]);
  const [applying, setApplying] = useState(false);

  const fetchModels = async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/model', { headers: { Authorization: `Bearer ${token}` } });
      const data = await res.json();
      if (res.ok) setModels(data.data || []);
    } catch (e) { console.error(e); }
    finally { setLoading(false); }
  };

  useEffect(() => { fetchModels(); }, []);

  const filteredModels = useMemo(() => {
    const kw = keyword.trim().toLowerCase();
    return models.filter((m) => {
      const hitKeyword = !kw
        || m.model_name.toLowerCase().includes(kw)
        || (m.display_name || '').toLowerCase().includes(kw);
      const hitUnset = !onlyUnset || isUnsetPricing(m);
      return hitKeyword && hitUnset;
    });
  }, [models, keyword, onlyUnset]);

  const selected = models.find(m => m.id === selectedId) || null;

  const handleSelect = (m: ModelConfig) => {
    setSelectedId(m.id);
    setForm(toForm(m));
    setApplyTargets([]);
  };

  const handleSave = async () => {
    if (!selected || !form) return;
    setSaving(true);
    try {
      const body = {
        id: selected.id,
        input_ratio: parseFloat(form.input_ratio) || 1,
        output_ratio: parseFloat(form.output_ratio) || 1,
        cache_ratio: parseFloat(form.cache_ratio) || 0.5,
        image_ratio: parseFloat(form.image_ratio) || 1,
        audio_ratio: parseFloat(form.audio_ratio) || 1,
        audio_completion_ratio: parseFloat(form.audio_completion_ratio) || 1,
        is_fixed_price: form.is_fixed_price,
        fixed_price: parseFloat(form.fixed_price) || 0,
        upstream_input_price: parseFloat(form.upstream_input_price) || 0,
        upstream_output_price: parseFloat(form.upstream_output_price) || 0,
      };
      const res = await fetch('/api/model', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify(body),
      });
      if (res.ok) {
        toast.success('定价已保存并生效');
        fetchModels();
      } else {
        toast.error('保存失败');
      }
    } catch (e) {
      console.error(e);
      toast.error('保存请求失败');
    } finally {
      setSaving(false);
    }
  };

  const handleSyncUpstream = async () => {
    setSyncing(true);
    try {
      const res = await fetch('/api/model/sync-upstream', {
        method: 'POST', headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.code === 0) {
        toast.success(`同步成功：创建 ${data.data.created} 个，更新 ${data.data.updated} 个`);
        fetchModels();
      } else {
        toast.error(data.msg || '同步失败');
      }
    } catch (e) {
      console.error(e);
      toast.error('同步请求失败');
    } finally {
      setSyncing(false);
    }
  };

  const handleApply = async () => {
    if (!selected || applyTargets.length === 0) {
      toast.error('请先勾选要应用到的模型');
      return;
    }
    setApplying(true);
    try {
      const res = await fetch('/api/model/batch-apply-pricing', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify({ source_id: selected.id, target_ids: applyTargets }),
      });
      const data = await res.json();
      if (data.code === 0) {
        toast.success(`已应用到 ${applyTargets.length} 个模型`);
        setApplyTargets([]);
        fetchModels();
      } else {
        toast.error(data.msg || '应用失败');
      }
    } catch (e) {
      console.error(e);
      toast.error('应用请求失败');
    } finally {
      setApplying(false);
    }
  };

  return (
    <div className="grid grid-cols-1 lg:grid-cols-[320px_1fr] gap-4">
      {/* 左侧模型列表 */}
      <Card>
        <CardBody className="space-y-3">
          <Input
            size="sm"
            placeholder="搜索模型名/显示名"
            value={keyword}
            onValueChange={setKeyword}
            startContent={<Search size={14} />}
          />
          <Switch isSelected={onlyUnset} onValueChange={setOnlyUnset}>
            仅显示未配置价格
          </Switch>
          <Button
            size="sm" variant="flat" color="warning" className="w-full"
            startContent={<RefreshCw size={14} />}
            isLoading={syncing}
            onPress={handleSyncUpstream}
          >
            从上游同步参考价
          </Button>
          <div className="max-h-[520px] overflow-y-auto space-y-1 pt-1">
            {loading ? (
              <p className="text-sm text-default-400 text-center py-6">加载中...</p>
            ) : filteredModels.length === 0 ? (
              <EmptyState icon="💰" title="暂无匹配模型" />
            ) : filteredModels.map((m) => (
              <div
                key={m.id}
                onClick={() => handleSelect(m)}
                className="p-2 rounded-lg cursor-pointer transition-all"
                style={{
                  background: m.id === selectedId ? 'var(--nav-active-bg)' : 'transparent',
                  border: `1px solid ${m.id === selectedId ? 'var(--accent-primary)' : 'transparent'}`,
                }}
              >
                <div className="flex items-center justify-between gap-2">
                  <span className="text-sm font-mono truncate">{m.model_name}</span>
                  {isUnsetPricing(m) && <AlertCircle size={13} className="text-warning flex-shrink-0" />}
                </div>
                <div className="flex items-center gap-1.5 mt-1">
                  <Chip size="sm" variant="flat" color={m.is_fixed_price ? 'secondary' : 'default'}>
                    {m.is_fixed_price ? '按次计费' : '按量计费'}
                  </Chip>
                </div>
              </div>
            ))}
          </div>
        </CardBody>
      </Card>

      {/* 右侧定价面板 */}
      {!selected || !form ? (
        <Card>
          <CardBody>
            <EmptyState icon="👈" title="请选择左侧的模型" description="点一个模型开始编辑它的定价" />
          </CardBody>
        </Card>
      ) : (
        <div className="space-y-4">
          <Card>
            <CardBody className="space-y-4">
              <div className="flex items-center justify-between">
                <div>
                  <h3 className="text-lg font-semibold">{selected.display_name || selected.model_name}</h3>
                  <p className="text-xs text-default-400 font-mono">{selected.model_name}</p>
                </div>
                <Switch
                  isSelected={form.is_fixed_price}
                  onValueChange={(v) => setForm({ ...form, is_fixed_price: v })}
                >
                  按次计费
                </Switch>
              </div>

              {form.is_fixed_price ? (
                <Input
                  label="单次价格（美元/次）"
                  type="number"
                  value={form.fixed_price}
                  onValueChange={(v) => setForm({ ...form, fixed_price: v })}
                  description="按次计费时忽略下面的倍率字段，每次调用固定收这个价"
                />
              ) : (
                <div className="grid grid-cols-2 gap-4">
                  <Input label="输入倍率" type="number" value={form.input_ratio}
                    onValueChange={(v) => setForm({ ...form, input_ratio: v })}
                    description="1 = 与基准价同价" />
                  <Input label="输出倍率" type="number" value={form.output_ratio}
                    onValueChange={(v) => setForm({ ...form, output_ratio: v })}
                    description="1 = 与输入同价" />
                  <Input label="缓存倍率" type="number" value={form.cache_ratio}
                    onValueChange={(v) => setForm({ ...form, cache_ratio: v })}
                    description="命中缓存的 token 打几折，0.5 = 五折" />
                  <Input label="图像倍率" type="number" value={form.image_ratio}
                    onValueChange={(v) => setForm({ ...form, image_ratio: v })}
                    description="图像 token 相对文本的倍率" />
                  <Input label="音频输入倍率" type="number" value={form.audio_ratio}
                    onValueChange={(v) => setForm({ ...form, audio_ratio: v })} />
                  <Input label="音频输出倍率" type="number" value={form.audio_completion_ratio}
                    onValueChange={(v) => setForm({ ...form, audio_completion_ratio: v })} />
                </div>
              )}
            </CardBody>
          </Card>

          <Card>
            <CardBody className="space-y-4">
              <h4 className="text-sm font-semibold text-default-500">上游成本价（仅展示比对，不参与计费）</h4>
              <div className="grid grid-cols-2 gap-4">
                <Input label="上游输入价格（元/百万 tokens）" type="number" value={form.upstream_input_price}
                  onValueChange={(v) => setForm({ ...form, upstream_input_price: v })} />
                <Input label="上游输出价格（元/百万 tokens）" type="number" value={form.upstream_output_price}
                  onValueChange={(v) => setForm({ ...form, upstream_output_price: v })} />
              </div>
            </CardBody>
          </Card>

          <Card>
            <CardBody className="space-y-3">
              <h4 className="text-sm font-semibold text-default-500 flex items-center gap-1.5">
                <Copy size={14} /> 应用到其他模型
              </h4>
              <div className="max-h-40 overflow-y-auto grid grid-cols-2 gap-1.5">
                {models.filter(m => m.id !== selected.id).map((m) => (
                  <Checkbox
                    key={m.id}
                    isSelected={applyTargets.includes(m.id)}
                    onValueChange={(v) => setApplyTargets(v
                      ? [...applyTargets, m.id]
                      : applyTargets.filter(id => id !== m.id))}
                  >
                    <span className="text-xs font-mono">{m.model_name}</span>
                  </Checkbox>
                ))}
              </div>
              <Button size="sm" variant="flat" isLoading={applying} onPress={handleApply}>
                应用到已选的 {applyTargets.length} 个模型
              </Button>
            </CardBody>
          </Card>

          <Button color="primary" startContent={<Save size={16} />} isLoading={saving} onPress={handleSave} className="w-full">
            保存并生效
          </Button>
        </div>
      )}
    </div>
  );
}
