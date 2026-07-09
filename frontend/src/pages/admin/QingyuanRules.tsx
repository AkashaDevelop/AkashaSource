import { useEffect, useMemo, useState } from 'react';
import PageHeader from '../../components/PageHeader';
import EmptyState from '../../components/EmptyState';
import LoadingRows from '../../components/LoadingRows';
import {
  Button, Input, Select, SelectItem, Switch, Chip, Textarea, Pagination,
  Modal, ModalContent, ModalHeader, ModalBody, ModalFooter, useDisclosure,
} from '../../components/ui';
import { RefreshCw, Plus, Pencil, Trash2, FlaskConical, Upload, Download, Search } from 'lucide-react';
import { useAuthStore } from '../../store/auth';
import { toast } from '../../store/toast';
import { confirm } from '../../store/confirm';

// ～宸汐清源 · 规则库管理：超管可以在这里自定义添加/修改检测规则，30秒内自动生效，
// 也可以点"立即刷新"让改动秒级生效喵～

interface RuleRow {
  id: number;
  category: string;
  name: string;
  description: string;
  score: number;
  keywords: string; // JSON 字符串数组
  context_required: string;
  match_mode: string; // any/all/regex
  enabled: boolean;
  language: string;
  sort_order: number;
  created_at: number;
  updated_at: number;
  created_by: number;
}

interface CategoryRow {
  id: number;
  category_key: string;
  display_name: string;
  parent_category: string;
  sort_order: number;
  description: string;
}

interface TestMatch {
  rule_id: number;
  category: string;
  name: string;
  score: number;
  matched_keyword: string;
}

const emptyRuleForm = {
  category: '',
  name: '',
  description: '',
  score: 50,
  keywordsText: '',
  context_required: '',
  match_mode: 'any',
  enabled: true,
  language: 'all',
  sort_order: 0,
};

const matchModeLabel = (m: string) => ({
  any: '任意匹配',
  all: '全部匹配',
  regex: '正则匹配',
}[m] || m);

const scoreColor = (score: number): 'default' | 'warning' | 'danger' => {
  if (score >= 70) return 'danger';
  if (score >= 45) return 'warning';
  return 'default';
};

function parseKeywords(raw: string): string[] {
  try {
    const arr = JSON.parse(raw);
    return Array.isArray(arr) ? arr : [];
  } catch {
    return [];
  }
}

export default function QingyuanRules() {
  const { token } = useAuthStore();
  const authHeaders = useMemo(() => ({ Authorization: `Bearer ${token}` }), [token]);

  const [rules, setRules] = useState<RuleRow[]>([]);
  const [categories, setCategories] = useState<CategoryRow[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const pageSize = 20;
  const [loading, setLoading] = useState(false);
  const [categoryFilter, setCategoryFilter] = useState('');
  const [enabledFilter, setEnabledFilter] = useState('');
  const [reloading, setReloading] = useState(false);
  const [lastReload, setLastReload] = useState<number | null>(null);

  const [editingRule, setEditingRule] = useState<RuleRow | null>(null);
  const [ruleForm, setRuleForm] = useState(emptyRuleForm);
  const [saving, setSaving] = useState(false);
  const { isOpen: isRuleModalOpen, onOpen: onRuleModalOpen, onOpenChange: onRuleModalOpenChange } = useDisclosure();

  // 规则测试器
  const [testText, setTestText] = useState('');
  const [testMatches, setTestMatches] = useState<TestMatch[] | null>(null);
  const [testing, setTesting] = useState(false);
  const { isOpen: isTestModalOpen, onOpen: onTestModalOpen, onOpenChange: onTestModalOpenChange } = useDisclosure();

  // 批量导入
  const [importText, setImportText] = useState('');
  const [importing, setImporting] = useState(false);
  const { isOpen: isImportModalOpen, onOpen: onImportModalOpen, onOpenChange: onImportModalOpenChange } = useDisclosure();

  const fetchCategories = async () => {
    try {
      const res = await fetch('/api/admin/qingyuan/categories', { headers: authHeaders });
      const data = await res.json();
      if (data.code === 0) setCategories(data.data || []);
    } catch (e) { console.error(e); }
  };

  const fetchRules = async () => {
    setLoading(true);
    const params = new URLSearchParams({ page: String(page), page_size: String(pageSize) });
    if (categoryFilter) params.set('category', categoryFilter);
    if (enabledFilter) params.set('enabled', enabledFilter);
    try {
      const res = await fetch(`/api/admin/qingyuan/rules?${params}`, { headers: authHeaders });
      const data = await res.json();
      if (data.code === 0) {
        setRules(data.data?.rules || []);
        setTotal(data.data?.total || 0);
      } else {
        toast.error(data.msg || '拉取规则失败');
      }
    } catch (e) {
      console.error(e);
      toast.error('网络异常，拉取规则失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (!token) return;
    fetchCategories();
  }, [token]);

  useEffect(() => {
    if (!token) return;
    fetchRules();
  }, [token, page, categoryFilter, enabledFilter]);

  const reloadCache = async () => {
    setReloading(true);
    try {
      const res = await fetch('/api/admin/qingyuan/rules/reload', { method: 'POST', headers: authHeaders });
      const data = await res.json();
      if (data.code === 0) {
        toast.success('规则缓存已刷新，立即生效');
        setLastReload(data.data?.reload_time || Date.now() / 1000);
      } else {
        toast.error(data.msg || '刷新失败');
      }
    } catch {
      toast.error('刷新失败');
    } finally {
      setReloading(false);
    }
  };

  const openCreateRule = () => {
    setEditingRule(null);
    setRuleForm(emptyRuleForm);
    onRuleModalOpen();
  };

  const openEditRule = (r: RuleRow) => {
    setEditingRule(r);
    const keywords = parseKeywords(r.keywords);
    setRuleForm({
      category: r.category,
      name: r.name,
      description: r.description || '',
      score: r.score,
      keywordsText: keywords.join('\n'),
      context_required: r.context_required || '',
      match_mode: r.match_mode || 'any',
      enabled: r.enabled,
      language: r.language || 'all',
      sort_order: r.sort_order || 0,
    });
    onRuleModalOpen();
  };

  const saveRule = async (onClose: () => void) => {
    if (!ruleForm.category.trim() || !ruleForm.name.trim()) {
      toast.error('分类和规则名称不能为空');
      return;
    }
    const keywords = ruleForm.keywordsText.split('\n').map(s => s.trim()).filter(Boolean);
    if (keywords.length === 0) {
      toast.error('至少需要一个关键词');
      return;
    }
    if (ruleForm.score < 0 || ruleForm.score > 100) {
      toast.error('风险分必须在 0-100 之间');
      return;
    }

    const body = {
      category: ruleForm.category.trim(),
      name: ruleForm.name.trim(),
      description: ruleForm.description.trim(),
      score: ruleForm.score,
      keywords: JSON.stringify(keywords),
      context_required: ruleForm.context_required.trim(),
      match_mode: ruleForm.match_mode,
      enabled: ruleForm.enabled,
      language: ruleForm.language,
      sort_order: ruleForm.sort_order,
    };

    setSaving(true);
    try {
      const res = await fetch(
        editingRule ? `/api/admin/qingyuan/rules/${editingRule.id}` : '/api/admin/qingyuan/rules',
        {
          method: editingRule ? 'PUT' : 'POST',
          headers: { ...authHeaders, 'Content-Type': 'application/json' },
          body: JSON.stringify(body),
        }
      );
      const data = await res.json();
      if (data.code === 0) {
        toast.success(editingRule ? '规则更新成功，已自动刷新缓存' : '规则创建成功，已自动刷新缓存');
        fetchRules();
        onClose();
      } else {
        toast.error(data.msg || '保存失败');
      }
    } catch {
      toast.error('保存失败');
    } finally {
      setSaving(false);
    }
  };

  const toggleRule = async (r: RuleRow) => {
    try {
      const res = await fetch(`/api/admin/qingyuan/rules/${r.id}/toggle`, { method: 'POST', headers: authHeaders });
      const data = await res.json();
      if (data.code === 0) {
        toast.success(data.data?.enabled ? '规则已启用' : '规则已禁用');
        fetchRules();
      } else {
        toast.error(data.msg || '切换失败');
      }
    } catch {
      toast.error('切换失败');
    }
  };

  const deleteRule = async (r: RuleRow) => {
    const ok = await confirm({
      title: '删除规则',
      message: `确定删除规则「${r.name}」吗？该操作不可恢复。`,
      danger: true,
    });
    if (!ok) return;
    try {
      const res = await fetch(`/api/admin/qingyuan/rules/${r.id}`, { method: 'DELETE', headers: authHeaders });
      const data = await res.json();
      if (data.code === 0) {
        toast.success('规则已删除');
        fetchRules();
      } else {
        toast.error(data.msg || '删除失败');
      }
    } catch {
      toast.error('删除失败');
    }
  };

  const runTest = async () => {
    if (!testText.trim()) {
      toast.error('请输入测试文本');
      return;
    }
    setTesting(true);
    try {
      const res = await fetch('/api/admin/qingyuan/rules/test', {
        method: 'POST',
        headers: { ...authHeaders, 'Content-Type': 'application/json' },
        body: JSON.stringify({ text: testText }),
      });
      const data = await res.json();
      if (data.code === 0) {
        setTestMatches(data.data?.matches || []);
      } else {
        toast.error(data.msg || '测试失败');
      }
    } catch {
      toast.error('测试失败');
    } finally {
      setTesting(false);
    }
  };

  const exportRules = async () => {
    try {
      const res = await fetch('/api/admin/qingyuan/rules/export', { headers: authHeaders });
      const data = await res.json();
      if (data.code === 0) {
        const blob = new Blob([JSON.stringify(data.data.rules, null, 2)], { type: 'application/json' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `qingyuan_rules_${new Date().toISOString().slice(0, 10)}.json`;
        a.click();
        URL.revokeObjectURL(url);
        toast.success(`已导出 ${data.data.count} 条规则`);
      } else {
        toast.error(data.msg || '导出失败');
      }
    } catch {
      toast.error('导出失败');
    }
  };

  const runImport = async () => {
    let parsed: any[];
    try {
      parsed = JSON.parse(importText);
      if (!Array.isArray(parsed)) throw new Error('not array');
    } catch {
      toast.error('JSON 格式错误，需要是规则数组');
      return;
    }

    setImporting(true);
    try {
      const res = await fetch('/api/admin/qingyuan/rules/import', {
        method: 'POST',
        headers: { ...authHeaders, 'Content-Type': 'application/json' },
        body: JSON.stringify({ rules: parsed }),
      });
      const data = await res.json();
      if (data.code === 0) {
        toast.success(`导入完成：成功 ${data.data.success}/${data.data.total} 条`);
        setImportText('');
        onImportModalOpenChange();
        fetchRules();
      } else {
        toast.error(data.msg || '导入失败');
      }
    } catch {
      toast.error('导入失败');
    } finally {
      setImporting(false);
    }
  };

  const categoryDisplayName = (key: string) => {
    const cat = categories.find(c => c.category_key === key);
    return cat ? cat.display_name : key;
  };

  const totalPages = Math.max(1, Math.ceil(total / pageSize));

  return (
    <div className="space-y-4">
      <PageHeader
        title="规则库管理"
        description="超管可自定义添加/修改检测规则，缓存每 30 秒自动刷新，或点击「立即刷新」秒级生效"
        actions={
          <div className="flex gap-2 flex-wrap items-center">
            {lastReload && (
              <span className="text-xs" style={{ color: 'var(--text-muted)' }}>
                上次刷新：{new Date(lastReload * 1000).toLocaleTimeString()}
              </span>
            )}
            <Button variant="flat" onPress={reloadCache} isLoading={reloading} startContent={<RefreshCw size={14} />}>
              立即刷新缓存
            </Button>
            <Button variant="flat" onPress={onTestModalOpen} startContent={<FlaskConical size={14} />}>
              规则测试器
            </Button>
            <Button variant="flat" onPress={exportRules} startContent={<Download size={14} />}>
              导出规则
            </Button>
            <Button variant="flat" onPress={onImportModalOpen} startContent={<Upload size={14} />}>
              批量导入
            </Button>
            <Button color="primary" onPress={openCreateRule} startContent={<Plus size={16} />}>
              新增规则
            </Button>
          </div>
        }
      />

      <div className="flex gap-2 flex-wrap items-center">
        <Select
          placeholder="按分类筛选"
          className="w-56"
          selectedKeys={categoryFilter ? [categoryFilter] : []}
          onSelectionChange={keys => { setCategoryFilter([...keys][0] as string || ''); setPage(1); }}
        >
          {categories.map(c => (
            <SelectItem key={c.category_key}>{c.display_name}</SelectItem>
          ))}
        </Select>
        <Select
          placeholder="启用状态"
          className="w-36"
          selectedKeys={enabledFilter ? [enabledFilter] : []}
          onSelectionChange={keys => { setEnabledFilter([...keys][0] as string || ''); setPage(1); }}
        >
          <SelectItem key="true">已启用</SelectItem>
          <SelectItem key="false">已禁用</SelectItem>
        </Select>
        <Button variant="flat" onPress={fetchRules} startContent={<RefreshCw size={14} />}>
          刷新列表
        </Button>
        <span className="text-xs ml-auto" style={{ color: 'var(--text-muted)' }}>
          共 {total} 条规则
        </span>
      </div>

      <div className="data-table-wrap">
        <table className="data-table">
          <thead>
            <tr>
              <th>分类</th>
              <th>规则名称</th>
              <th>关键词</th>
              <th>风险分</th>
              <th>匹配模式</th>
              <th>上下文要求</th>
              <th>启用</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <LoadingRows cols={8} rows={6} />
            ) : rules.length === 0 ? (
              <tr>
                <td colSpan={8}>
                  <EmptyState icon="🛡️" title="暂无规则" description="点击右上角新增规则，或批量导入现有规则库" />
                </td>
              </tr>
            ) : (
              rules.map(r => {
                const keywords = parseKeywords(r.keywords);
                return (
                  <tr key={r.id}>
                    <td><Chip size="sm" variant="flat">{categoryDisplayName(r.category)}</Chip></td>
                    <td>
                      <div className="flex flex-col gap-0.5">
                        <span className="text-sm font-medium">{r.name}</span>
                        {r.description && (
                          <span className="text-xs" style={{ color: 'var(--text-secondary)' }}>{r.description}</span>
                        )}
                      </div>
                    </td>
                    <td className="text-xs max-w-xs truncate" title={keywords.join('、')}>
                      {keywords.slice(0, 3).join('、')}{keywords.length > 3 ? ` 等${keywords.length}项` : ''}
                    </td>
                    <td><Chip size="sm" color={scoreColor(r.score)} variant="flat">{r.score}</Chip></td>
                    <td className="text-xs">{matchModeLabel(r.match_mode)}</td>
                    <td className="text-xs max-w-xs truncate">{r.context_required || '-'}</td>
                    <td><Switch size="sm" isSelected={r.enabled} onValueChange={() => toggleRule(r)} /></td>
                    <td>
                      <div className="flex gap-2">
                        <span className="cursor-pointer text-default-400" onClick={() => openEditRule(r)}>
                          <Pencil size={16} />
                        </span>
                        <span className="cursor-pointer text-danger" onClick={() => deleteRule(r)}>
                          <Trash2 size={16} />
                        </span>
                      </div>
                    </td>
                  </tr>
                );
              })
            )}
          </tbody>
        </table>
      </div>

      {totalPages > 1 && (
        <div className="flex justify-center">
          <Pagination total={totalPages} page={page} onChange={setPage} />
        </div>
      )}

      {/* ══ 新增/编辑规则弹窗 ══ */}
      <Modal isOpen={isRuleModalOpen} onOpenChange={onRuleModalOpenChange} size="2xl" scrollBehavior="inside">
        <ModalContent>
          {onClose => (
            <>
              <ModalHeader>{editingRule ? '编辑规则' : '新增规则'}</ModalHeader>
              <ModalBody className="gap-4">
                <div className="grid grid-cols-2 gap-4">
                  <Select
                    label="规则分类"
                    isRequired
                    selectedKeys={ruleForm.category ? [ruleForm.category] : []}
                    onSelectionChange={keys => setRuleForm({ ...ruleForm, category: [...keys][0] as string || '' })}
                  >
                    {categories.map(c => (
                      <SelectItem key={c.category_key}>{c.display_name}</SelectItem>
                    ))}
                  </Select>
                  <Input
                    label="规则名称"
                    placeholder="如：忽略之前指令"
                    isRequired
                    value={ruleForm.name}
                    onValueChange={v => setRuleForm({ ...ruleForm, name: v })}
                  />
                </div>

                <Input
                  label="规则描述（可选）"
                  placeholder="简要说明这条规则检测什么"
                  value={ruleForm.description}
                  onValueChange={v => setRuleForm({ ...ruleForm, description: v })}
                />

                <Textarea
                  label="关键词（每行一个，命中即视为匹配）"
                  minRows={5}
                  placeholder={'ignore previous\ndisregard above\n忽略之前'}
                  value={ruleForm.keywordsText}
                  onValueChange={v => setRuleForm({ ...ruleForm, keywordsText: v })}
                />

                <Input
                  label="上下文要求（可选，用 | 分隔，需与关键词同时出现才触发）"
                  placeholder="illegal|harmful|dangerous"
                  value={ruleForm.context_required}
                  onValueChange={v => setRuleForm({ ...ruleForm, context_required: v })}
                  description="留空表示关键词命中即触发，不需要额外上下文"
                />

                <div className="grid grid-cols-3 gap-4">
                  <Input
                    type="number"
                    label="风险分（0-100）"
                    value={String(ruleForm.score)}
                    onValueChange={v => setRuleForm({ ...ruleForm, score: parseInt(v) || 0 })}
                  />
                  <Select
                    label="匹配模式"
                    selectedKeys={[ruleForm.match_mode]}
                    onSelectionChange={keys => setRuleForm({ ...ruleForm, match_mode: [...keys][0] as string || 'any' })}
                  >
                    <SelectItem key="any">任意关键词匹配</SelectItem>
                    <SelectItem key="all">全部关键词匹配</SelectItem>
                    <SelectItem key="regex">正则表达式匹配</SelectItem>
                  </Select>
                  <Select
                    label="语言限制"
                    selectedKeys={[ruleForm.language]}
                    onSelectionChange={keys => setRuleForm({ ...ruleForm, language: [...keys][0] as string || 'all' })}
                  >
                    <SelectItem key="all">全部语言</SelectItem>
                    <SelectItem key="zh">仅中文</SelectItem>
                    <SelectItem key="en">仅英文</SelectItem>
                    <SelectItem key="ja">仅日文</SelectItem>
                  </Select>
                </div>

                <div className="grid grid-cols-2 gap-4">
                  <Input
                    type="number"
                    label="排序（小的在前）"
                    value={String(ruleForm.sort_order)}
                    onValueChange={v => setRuleForm({ ...ruleForm, sort_order: parseInt(v) || 0 })}
                  />
                  <div className="flex items-end pb-2">
                    <Switch isSelected={ruleForm.enabled} onValueChange={v => setRuleForm({ ...ruleForm, enabled: v })}>
                      立即启用
                    </Switch>
                  </div>
                </div>
              </ModalBody>
              <ModalFooter>
                <Button variant="flat" onPress={onClose}>取消</Button>
                <Button color="primary" isLoading={saving} onPress={() => saveRule(onClose)}>保存</Button>
              </ModalFooter>
            </>
          )}
        </ModalContent>
      </Modal>

      {/* ══ 规则测试器弹窗 ══ */}
      <Modal isOpen={isTestModalOpen} onOpenChange={onTestModalOpenChange} size="2xl" scrollBehavior="inside">
        <ModalContent>
          {() => (
            <>
              <ModalHeader>规则测试器</ModalHeader>
              <ModalBody className="gap-4">
                <Textarea
                  label="输入测试文本"
                  placeholder="粘贴要测试的文本，查看哪些已启用规则会触发"
                  minRows={5}
                  value={testText}
                  onValueChange={v => setTestText(v)}
                />
                <Button color="primary" onPress={runTest} isLoading={testing} startContent={<Search size={16} />}>
                  开始测试
                </Button>
                {testMatches !== null && (
                  <div className="space-y-2">
                    <div className="text-sm font-medium">
                      {testMatches.length === 0 ? '未匹配到任何规则' : `匹配到 ${testMatches.length} 条规则`}
                    </div>
                    {testMatches.map((m, idx) => (
                      <div key={idx} className="p-3 rounded-lg flex items-center justify-between" style={{ background: 'var(--bg-elevated)', border: '1px solid var(--border-color)' }}>
                        <div className="flex flex-col gap-0.5">
                          <span className="text-sm font-medium">{m.name}</span>
                          <span className="text-xs" style={{ color: 'var(--text-secondary)' }}>
                            {categoryDisplayName(m.category)} · 命中关键词: {m.matched_keyword}
                          </span>
                        </div>
                        <Chip size="sm" color={scoreColor(m.score)} variant="flat">{m.score}</Chip>
                      </div>
                    ))}
                  </div>
                )}
              </ModalBody>
              <ModalFooter />
            </>
          )}
        </ModalContent>
      </Modal>

      {/* ══ 批量导入弹窗 ══ */}
      <Modal isOpen={isImportModalOpen} onOpenChange={onImportModalOpenChange} size="2xl" scrollBehavior="inside">
        <ModalContent>
          {onClose => (
            <>
              <ModalHeader>批量导入规则</ModalHeader>
              <ModalBody className="gap-4">
                <div className="text-xs" style={{ color: 'var(--text-secondary)' }}>
                  粘贴规则数组 JSON（与「导出规则」的格式一致），每条规则需包含 category / name / keywords / score 等字段。
                </div>
                <Textarea
                  label="规则 JSON"
                  minRows={10}
                  placeholder='[{"category": "jailbreak_dan", "name": "示例规则", "score": 60, "keywords": "[\"keyword1\"]", "match_mode": "any", "enabled": true}]'
                  value={importText}
                  onValueChange={v => setImportText(v)}
                />
              </ModalBody>
              <ModalFooter>
                <Button variant="flat" onPress={onClose}>取消</Button>
                <Button color="primary" isLoading={importing} onPress={runImport}>开始导入</Button>
              </ModalFooter>
            </>
          )}
        </ModalContent>
      </Modal>
    </div>
  );
}
