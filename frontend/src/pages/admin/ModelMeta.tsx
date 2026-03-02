import { useEffect, useState } from 'react';
import PageHeader from '../../components/PageHeader';
import { Button, Input, Modal, ModalBody, ModalContent, ModalFooter, ModalHeader, Switch, useDisclosure } from '../../components/ui';
import { Pagination } from '../../components/ui/Pagination';
import { useAuthStore } from '../../store/auth';
import { confirm } from '../../store/confirm';

interface ModelMeta {
  id: number;
  vendor_id: number;
  model_name: string;
  display_name: string;
  model_type: string;
  context_length: number;
  input_price: number;
  output_price: number;
  enabled: boolean;
}

export default function ModelMetaManagement() {
  const { token } = useAuthStore();
  const { isOpen, onOpen, onOpenChange } = useDisclosure();
  const [items, setItems] = useState<ModelMeta[]>([]);
  const [keyword, setKeyword] = useState('');
  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [editing, setEditing] = useState<ModelMeta | null>(null);
  const [form, setForm] = useState<Partial<ModelMeta>>({ vendor_id: 0, model_name: '', display_name: '', model_type: 'chat', context_length: 0, input_price: 0, output_price: 0, enabled: true });

  const fetchData = async (targetPage = 1) => {
    const pageSize = 20;
    const q = keyword.trim();
    const base = q ? `/api/models/search?keyword=${encodeURIComponent(q)}` : '/api/models';
    const url = `${base}${base.includes('?') ? '&' : '?'}page=${targetPage}&page_size=${pageSize}`;
    const res = await fetch(url, { headers: { Authorization: `Bearer ${token}` } });
    const data = await res.json();
    if (data.code !== 0) return;

    const payload = data.data;
    if (Array.isArray(payload)) {
      setItems(payload);
      setPage(1);
      setTotalPages(1);
      return;
    }

    const nextItems = payload?.items || [];
    const total = Number(payload?.total ?? nextItems.length);
    const ps = Number(payload?.page_size ?? pageSize) || pageSize;
    const nextPage = Number(payload?.page ?? targetPage) || targetPage;
    setItems(nextItems);
    setPage(nextPage);
    setTotalPages(Math.max(1, Math.ceil(total / ps)));
  };

  useEffect(() => { fetchData(1); }, []);

  const openCreate = () => {
    setEditing(null);
    setForm({ vendor_id: 0, model_name: '', display_name: '', model_type: 'chat', context_length: 0, input_price: 0, output_price: 0, enabled: true });
    onOpen();
  };

  const openEdit = (v: ModelMeta) => {
    setEditing(v);
    setForm(v);
    onOpen();
  };

  const submit = async (onClose: () => void) => {
    const method = editing ? 'PUT' : 'POST';
    const body = editing ? { ...form, id: editing.id } : form;
    const res = await fetch('/api/models', {
      method,
      headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
      body: JSON.stringify(body),
    });
    const data = await res.json();
    if (data.code === 0) {
      await fetchData(page);
      onClose();
    }
  };

  const remove = async (id: number) => {
    if (!await confirm({ title: '删除模型元数据', message: '确认删除该记录？', danger: true })) return;
    const res = await fetch(`/api/models/${id}`, { method: 'DELETE', headers: { Authorization: `Bearer ${token}` } });
    const data = await res.json();
    if (data.code === 0) fetchData(page);
  };

  return (
    <div className="space-y-6">
      <PageHeader title="模型元数据" description="Models Meta 管理"
        actions={<div className="flex gap-2"><Input size="sm" placeholder="搜索" value={keyword} onValueChange={setKeyword} onKeyDown={(e) => e.key === 'Enter' && fetchData(1)} /><Button variant="flat" onPress={() => fetchData(1)}>查询</Button><Button color="primary" onPress={openCreate}>新增</Button></div>} />
      <div className="data-table-wrap">
        <table className="data-table">
          <thead><tr><th>ID</th><th>Model</th><th>Type</th><th>Vendor</th><th>Input</th><th>Output</th><th>Enabled</th><th>操作</th></tr></thead>
          <tbody>
            {items.map(v => (
              <tr key={v.id}>
                <td>{v.id}</td><td>{v.model_name}</td><td>{v.model_type}</td><td>{v.vendor_id}</td><td>{v.input_price}</td><td>{v.output_price}</td><td>{v.enabled ? 'Y' : 'N'}</td>
                <td><div className="flex gap-2"><Button size="sm" variant="flat" onPress={() => openEdit(v)}>编辑</Button><Button size="sm" color="danger" variant="flat" onPress={() => remove(v.id)}>删除</Button></div></td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <div className="flex justify-end">
        <Pagination total={totalPages} page={page} onChange={(p) => fetchData(p)} />
      </div>

      <Modal isOpen={isOpen} onOpenChange={onOpenChange}>
        <ModalContent>
          {(onClose) => (
            <>
              <ModalHeader>{editing ? '编辑模型元数据' : '新增模型元数据'}</ModalHeader>
              <ModalBody className="gap-3">
                <Input label="Vendor ID" value={String(form.vendor_id ?? 0)} onValueChange={(v) => setForm({ ...form, vendor_id: Number(v) || 0 })} />
                <Input label="模型名" value={form.model_name || ''} onValueChange={(v) => setForm({ ...form, model_name: v })} />
                <Input label="显示名" value={form.display_name || ''} onValueChange={(v) => setForm({ ...form, display_name: v })} />
                <Input label="类型" value={form.model_type || ''} onValueChange={(v) => setForm({ ...form, model_type: v })} />
                <Input label="上下文长度" value={String(form.context_length ?? 0)} onValueChange={(v) => setForm({ ...form, context_length: Number(v) || 0 })} />
                <Input label="输入单价" value={String(form.input_price ?? 0)} onValueChange={(v) => setForm({ ...form, input_price: Number(v) || 0 })} />
                <Input label="输出单价" value={String(form.output_price ?? 0)} onValueChange={(v) => setForm({ ...form, output_price: Number(v) || 0 })} />
                <Switch isSelected={!!form.enabled} onValueChange={(v) => setForm({ ...form, enabled: v })}>启用</Switch>
              </ModalBody>
              <ModalFooter><Button variant="light" onPress={onClose}>取消</Button><Button color="primary" onPress={() => submit(onClose)}>保存</Button></ModalFooter>
            </>
          )}
        </ModalContent>
      </Modal>
    </div>
  );
}
