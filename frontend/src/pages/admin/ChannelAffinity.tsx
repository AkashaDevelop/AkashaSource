import { useState, useEffect } from 'react';
import { Card, CardBody, CardHeader, Button, Input, Select, Switch, Textarea } from '../../components/ui';
import { Plus, Trash2, Save, RefreshCw, Info } from 'lucide-react';
import { useAuthStore } from '../../store/auth';
import PageHeader from '../../components/PageHeader';

/**
 * 渠道亲和性规则接口
 *
 * 渠道亲和性是一种智能路由机制，通过规则匹配将特定用户/会话的请求
 * 固定路由到同一个渠道，从而提升 Prompt Cache 命中率，降低成本
 */
interface AffinityRule {
  name: string;                    // 规则名称（唯一标识）
  model_regex: string;             // 模型名称正则表达式（必填）
  path_regex: string;              // 请求路径正则表达式（可选）
  user_agent_include: string[];    // User-Agent 包含关键词（可选）
  key_source_type: string;         // 亲和性键来源类型
  key_source_key: string;          // 键名称（context_int/context_string/header/query）
  key_source_path: string;         // JSON 路径（gjson 格式，暂不支持）
  value_regex: string;             // 亲和性值正则表达式（可选）
  ttl_seconds: number;             // 缓存 TTL（秒）
  include_rule_name: boolean;      // 缓存键是否包含规则名
  include_using_group: boolean;    // 缓存键是否包含用户组
  skip_retry_on_fail: boolean;     // 失败时是否跳过重试
  param_override: Record<string, any>; // 参数覆盖模板（JSON）
}

/**
 * 缓存统计接口
 */
interface CacheStats {
  enabled: boolean;
  total: number;
  unknown: number;
  by_rule_name: Record<string, number>;
  last_seen_at?: number;
}

export default function ChannelAffinity() {
  const [rules, setRules] = useState<AffinityRule[]>([]);
  const [stats, setStats] = useState<CacheStats | null>(null);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const { token } = useAuthStore();

  /**
   * 获取亲和性规则配置
   */
  const fetchRules = async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/admin/channel-affinity/rules', {
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.code === 0 && data.data) {
        setRules(data.data.rules || []);
      }
    } catch (error) {
      console.error('获取亲和性规则失败:', error);
    } finally {
      setLoading(false);
    }
  };

  /**
   * 获取缓存统计信息
   */
  const fetchStats = async () => {
    try {
      const res = await fetch('/api/admin/channel-affinity/stats', {
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.code === 0) {
        setStats(data.data);
      }
    } catch (error) {
      console.error('获取缓存统计失败:', error);
    }
  };

  useEffect(() => {
    if (token) {
      fetchRules();
      fetchStats();
    }
  }, [token]);

  /**
   * 保存规则配置
   */
  const saveRules = async () => {
    setSaving(true);
    try {
      const res = await fetch('/api/admin/channel-affinity/rules', {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ rules }),
      });
      const data = await res.json();
      if (data.code === 0) {
        alert('保存成功');
        fetchStats();
      } else {
        alert(`保存失败: ${data.message}`);
      }
    } catch (error) {
      alert('保存失败');
      console.error(error);
    } finally {
      setSaving(false);
    }
  };

  /**
   * 清空所有缓存
   */
  const clearAllCache = async () => {
    if (!confirm('确定要清空所有亲和性缓存吗？')) return;
    try {
      const res = await fetch('/api/admin/channel-affinity/cache/clear', {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.code === 0) {
        alert(`已清空 ${data.data.count} 条缓存`);
        fetchStats();
      }
    } catch (error) {
      console.error('清空缓存失败:', error);
    }
  };

  /**
   * 添加新规则
   */
  const addRule = () => {
    setRules([...rules, {
      name: `rule_${Date.now()}`,
      model_regex: '.*',
      path_regex: '',
      user_agent_include: [],
      key_source_type: 'context_int',
      key_source_key: 'user_id',
      key_source_path: '',
      value_regex: '',
      ttl_seconds: 3600,
      include_rule_name: true,
      include_using_group: true,
      skip_retry_on_fail: false,
      param_override: {},
    }]);
  };

  /**
   * 删除规则
   */
  const deleteRule = (index: number) => {
    if (!confirm('确定要删除此规则吗？')) return;
    setRules(rules.filter((_, i) => i !== index));
  };

  /**
   * 更新规则字段
   */
  const updateRule = (index: number, field: keyof AffinityRule, value: any) => {
    const newRules = [...rules];
    newRules[index] = { ...newRules[index], [field]: value };
    setRules(newRules);
  };

  return (
    <div className="space-y-6">
      <PageHeader
        title="渠道亲和性配置"
        description="配置智能路由规则，提升 Prompt Cache 命中率"
        actions={
          <div className="flex gap-2">
            <Button onClick={clearAllCache} variant="outline">
              <Trash2 className="w-4 h-4 mr-2" />
              清空缓存
            </Button>
            <Button onClick={addRule} variant="outline">
              <Plus className="w-4 h-4 mr-2" />
              添加规则
            </Button>
            <Button onClick={saveRules} disabled={saving}>
              <Save className="w-4 h-4 mr-2" />
              {saving ? '保存中...' : '保存配置'}
            </Button>
          </div>
        }
      />

      {/* 缓存统计卡片 */}
      {stats && (
        <Card>
          <CardHeader>
            <h3 className="text-lg font-semibold">缓存统计</h3>
          </CardHeader>
          <CardBody>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
              <div>
                <div className="text-sm text-gray-500">状态</div>
                <div className="text-xl font-bold">
                  {stats.enabled ? '已启用' : '已禁用'}
                </div>
              </div>
              <div>
                <div className="text-sm text-gray-500">缓存总数</div>
                <div className="text-xl font-bold">{stats.total}</div>
              </div>
              <div>
                <div className="text-sm text-gray-500">最后更新</div>
                <div className="text-xl font-bold">
                  {stats.last_seen_at ? new Date(stats.last_seen_at * 1000).toLocaleString() : '-'}
                </div>
              </div>
            </div>
            {stats.by_rule_name && Object.keys(stats.by_rule_name).length > 0 && (
              <div className="mt-4">
                <div className="text-sm font-medium mb-2">各规则缓存数量</div>
                <div className="space-y-1">
                  {Object.entries(stats.by_rule_name).map(([name, count]) => (
                    <div key={name} className="flex justify-between text-sm">
                      <span>{name}</span>
                      <span className="font-mono">{count}</span>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </CardBody>
        </Card>
      )}

      {/* 功能说明卡片 */}
      <Card>
        <CardHeader>
          <div className="flex items-center gap-2">
            <Info className="w-5 h-5 text-blue-500" />
            <h3 className="text-lg font-semibold">功能说明</h3>
          </div>
        </CardHeader>
        <CardBody>
          <div className="space-y-2 text-sm text-gray-600">
            <p><strong>渠道亲和性</strong>是一种智能路由机制，将特定用户/会话的请求固定路由到同一渠道。</p>
            <p><strong>核心优势：</strong></p>
            <ul className="list-disc list-inside space-y-1 ml-4">
              <li>提升 Prompt Cache 命中率（相同上下文重复使用）</li>
              <li>降低 API 调用成本（缓存命中可节省 90% 费用）</li>
              <li>改善响应速度（缓存响应更快）</li>
            </ul>
            <p><strong>配置要点：</strong></p>
            <ul className="list-disc list-inside space-y-1 ml-4">
              <li><code>model_regex</code>: 匹配模型名称（必填，如 <code>gpt-4.*</code>）</li>
              <li><code>key_source_type</code>: 选择亲和性键来源（user_id、session_id 等）</li>
              <li><code>ttl_seconds</code>: 缓存有效期（建议 3600-7200 秒）</li>
              <li><code>include_using_group</code>: 按用户组隔离缓存</li>
            </ul>
          </div>
        </CardBody>
      </Card>

      {/* 规则列表 */}
      <div className="space-y-4">
        {rules.map((rule, index) => (
          <Card key={index}>
            <CardHeader>
              <div className="flex items-center justify-between">
                <h3 className="text-lg font-semibold">规则 #{index + 1}: {rule.name}</h3>
                <Button
                  onClick={() => deleteRule(index)}
                  variant="outline"
                  size="sm"
                >
                  <Trash2 className="w-4 h-4" />
                </Button>
              </div>
            </CardHeader>
            <CardBody>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                {/* 规则名称 */}
                <div>
                  <label className="block text-sm font-medium mb-1">
                    规则名称 <span className="text-red-500">*</span>
                  </label>
                  <Input
                    value={rule.name}
                    onChange={(e) => updateRule(index, 'name', e.target.value)}
                    placeholder="rule_name"
                  />
                  <p className="text-xs text-gray-500 mt-1">唯一标识，用于缓存键和统计</p>
                </div>

                {/* 模型正则 */}
                <div>
                  <label className="block text-sm font-medium mb-1">
                    模型正则表达式 <span className="text-red-500">*</span>
                  </label>
                  <Input
                    value={rule.model_regex}
                    onChange={(e) => updateRule(index, 'model_regex', e.target.value)}
                    placeholder="gpt-4.*"
                  />
                  <p className="text-xs text-gray-500 mt-1">匹配模型名称，如 <code>gpt-4.*</code> 或 <code>claude-.*</code></p>
                </div>

                {/* 路径正则 */}
                <div>
                  <label className="block text-sm font-medium mb-1">路径正则表达式</label>
                  <Input
                    value={rule.path_regex}
                    onChange={(e) => updateRule(index, 'path_regex', e.target.value)}
                    placeholder="/v1/chat/completions"
                  />
                  <p className="text-xs text-gray-500 mt-1">可选，匹配请求路径</p>
                </div>

                {/* 键来源类型 */}
                <div>
                  <label className="block text-sm font-medium mb-1">
                    亲和性键来源 <span className="text-red-500">*</span>
                  </label>
                  <Select
                    value={rule.key_source_type}
                    onChange={(e) => updateRule(index, 'key_source_type', e.target.value)}
                  >
                    <option value="context_int">Context Int (如 user_id)</option>
                    <option value="context_string">Context String (如 session_id)</option>
                    <option value="header">HTTP Header</option>
                    <option value="query">Query 参数</option>
                  </Select>
                  <p className="text-xs text-gray-500 mt-1">从哪里提取亲和性值</p>
                </div>

                {/* 键名称 */}
                <div>
                  <label className="block text-sm font-medium mb-1">
                    键名称 <span className="text-red-500">*</span>
                  </label>
                  <Input
                    value={rule.key_source_key}
                    onChange={(e) => updateRule(index, 'key_source_key', e.target.value)}
                    placeholder="user_id"
                  />
                  <p className="text-xs text-gray-500 mt-1">
                    {rule.key_source_type === 'context_int' && 'Context 中的整数键名'}
                    {rule.key_source_type === 'context_string' && 'Context 中的字符串键名'}
                    {rule.key_source_type === 'header' && 'HTTP Header 名称'}
                    {rule.key_source_type === 'query' && 'Query 参数名称'}
                  </p>
                </div>

                {/* TTL */}
                <div>
                  <label className="block text-sm font-medium mb-1">缓存 TTL (秒)</label>
                  <Input
                    type="number"
                    value={rule.ttl_seconds}
                    onChange={(e) => updateRule(index, 'ttl_seconds', parseInt(e.target.value) || 3600)}
                    placeholder="3600"
                  />
                  <p className="text-xs text-gray-500 mt-1">建议 3600-7200 秒</p>
                </div>
              </div>

              {/* 开关选项 */}
              <div className="mt-4 space-y-3 border-t pt-4">
                <div className="flex items-center justify-between">
                  <div>
                    <div className="font-medium">包含规则名</div>
                    <div className="text-xs text-gray-500">缓存键是否包含规则名（用于隔离不同规则）</div>
                  </div>
                  <Switch
                    checked={rule.include_rule_name}
                    onChange={(checked) => updateRule(index, 'include_rule_name', checked)}
                  />
                </div>

                <div className="flex items-center justify-between">
                  <div>
                    <div className="font-medium">包含用户组</div>
                    <div className="text-xs text-gray-500">缓存键是否包含用户组（按组隔离缓存）</div>
                  </div>
                  <Switch
                    checked={rule.include_using_group}
                    onChange={(checked) => updateRule(index, 'include_using_group', checked)}
                  />
                </div>

                <div className="flex items-center justify-between">
                  <div>
                    <div className="font-medium">失败跳过重试</div>
                    <div className="text-xs text-gray-500">匹配此规则的请求失败时不重试其他渠道</div>
                  </div>
                  <Switch
                    checked={rule.skip_retry_on_fail}
                    onChange={(checked) => updateRule(index, 'skip_retry_on_fail', checked)}
                  />
                </div>
              </div>

              {/* 高级选项 */}
              <div className="mt-4 border-t pt-4">
                <details className="group">
                  <summary className="cursor-pointer font-medium text-sm">高级选项</summary>
                  <div className="mt-3 space-y-3">
                    <div>
                      <label className="block text-sm font-medium mb-1">值正则表达式</label>
                      <Input
                        value={rule.value_regex}
                        onChange={(e) => updateRule(index, 'value_regex', e.target.value)}
                        placeholder="^\d+$"
                      />
                      <p className="text-xs text-gray-500 mt-1">可选，对提取的亲和性值进行二次过滤</p>
                    </div>

                    <div>
                      <label className="block text-sm font-medium mb-1">参数覆盖模板 (JSON)</label>
                      <Textarea
                        value={JSON.stringify(rule.param_override, null, 2)}
                        onChange={(e) => {
                          try {
                            const parsed = JSON.parse(e.target.value);
                            updateRule(index, 'param_override', parsed);
                          } catch {}
                        }}
                        placeholder='{"temperature": 0.7}'
                        rows={3}
                      />
                      <p className="text-xs text-gray-500 mt-1">可选，动态覆盖请求参数</p>
                    </div>
                  </div>
                </details>
              </div>
            </CardBody>
          </Card>
        ))}
      </div>

      {rules.length === 0 && (
        <Card>
          <CardBody>
            <div className="text-center py-12 text-gray-500">
              <p className="mb-4">暂无规则配置</p>
              <Button onClick={addRule}>
                <Plus className="w-4 h-4 mr-2" />
                添加第一条规则
              </Button>
            </div>
          </CardBody>
        </Card>
      )}
    </div>
  );
}
