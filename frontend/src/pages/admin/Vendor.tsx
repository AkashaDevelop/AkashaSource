import { useEffect, useState } from 'react';
import PageHeader from '../../components/PageHeader';
import { Button, Input, Modal, ModalBody, ModalContent, ModalFooter, ModalHeader, useDisclosure } from '../../components/ui';
import { Pagination } from '../../components/ui/Pagination';
import { useAuthStore } from '../../store/auth';
import { confirm } from '../../store/confirm';

interface Vendor {
  id: number;
  name: string;
  code: string;
  base_url: string;
  api_version: string;
  status: number;
  description: string;
}

export default function VendorManagement() {
  const { token } = useAuthStore();
  const { isOpen, onOpen, onOpenChange } = useDisclosure();
  const [items, setItems] = useState<Vendor[]>([]);
  const [keyword, setKeyword] = useState('');
  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [editing, setEditing] = useState<Vendor | null>(null);
  const [form, setForm] = useState<Partial<Vendor>>({ name: '', code: '', base_url: '', api_version: '', status: 1, description: '' });

  const fetchData = async (targetPage = 1) => {
    const pageSize = 20;
    const q = keyword.trim();
    const base = q ? `/api/vendors/search?keyword=${encodeURIComponent(q)}` : '/api/vendors';
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
    setForm({ name: '', code: '', base_url: '', api_version: '', status: 1, description: '' });
    onOpen();
  };

  const openEdit = (v: Vendor) => {
    setEditing(v);
    setForm(v);
    onOpen();
  };

  const submit = async (onClose: () => void) => {
    const method = editing ? 'PUT' : 'POST';
    const body = editing ? { ...form, id: editing.id } : form;
    const res = await fetch('/api/vendors', {
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
    if (!await confirm({ title: '删除供应商', message: '确认删除该供应商？', danger: true })) return;
    const res = await fetch(`/api/vendors/${id}`, { method: 'DELETE', headers: { Authorization: `Bearer ${token}` } });
    const data = await res.json();
    if (data.code === 0) fetchData(page);
  };

  return (
    <div className="space-y-6">
      <PageHeader title="供应商管理" description="Vendors 元数据维护"
        actions={<div className="flex gap-2"><Input size="sm" placeholder="搜索" value={keyword} onValueChange={setKeyword} onKeyDown={(e) => e.key === 'Enter' && fetchData(1)} /><Button variant="flat" onPress={() => fetchData(1)}>查询</Button><Button color="primary" onPress={openCreate}>新增</Button></div>} />
      <div className="data-table-wrap">
        <table className="data-table">
          <thead><tr><th>ID</th><th>Name</th><th>Code</th><th>Base URL</th><th>Version</th><th>状态</th><th>操作</th></tr></thead>
          <tbody>
            {items.map(v => (
              <tr key={v.id}>
                <td>{v.id}</td><td>{v.name}</td><td>{v.code}</td><td>{v.base_url}</td><td>{v.api_version || '-'}</td><td>{v.status === 1 ? '启用' : '禁用'}</td>
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
              <ModalHeader>{editing ? '编辑供应商' : '新增供应商'}</ModalHeader>
              <ModalBody className="gap-3">
                <Input label="名称" value={form.name || ''} onValueChange={(v) => setForm({ ...form, name: v })} />
                <Input label="代码" value={form.code || ''} onValueChange={(v) => setForm({ ...form, code: v })} />
                <Input label="Base URL" value={form.base_url || ''} onValueChange={(v) => setForm({ ...form, base_url: v })} />
                <Input label="API 版本" value={form.api_version || ''} onValueChange={(v) => setForm({ ...form, api_version: v })} />
                <Input label="状态(1启用/2禁用)" value={String(form.status ?? 1)} onValueChange={(v) => setForm({ ...form, status: Number(v) || 1 })} />
                <Input label="描述" value={form.description || ''} onValueChange={(v) => setForm({ ...form, description: v })} />
              </ModalBody>
              <ModalFooter><Button variant="light" onPress={onClose}>取消</Button><Button color="primary" onPress={() => submit(onClose)}>保存</Button></ModalFooter>
            </>
          )}
        </ModalContent>
      </Modal>
    </div>
  );
}
