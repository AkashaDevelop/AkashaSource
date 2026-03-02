import { useEffect, useMemo, useState } from 'react';
import PageHeader from '../../components/PageHeader';
import { Button, Input, Modal, ModalBody, ModalContent, ModalFooter, ModalHeader, useDisclosure } from '../../components/ui';
import { useAuthStore } from '../../store/auth';
import { confirm } from '../../store/confirm';

interface DeploymentItem {
  id: string;
  deployment_name: string;
  status: string;
  hardware_info?: string;
  completed_percent?: number;
  time_remaining?: string;
  provider?: string;
}

interface CreateDeploymentForm {
  resource_private_name: string;
  duration_hours: number;
  gpus_per_container: number;
  hardware_id: number;
  location_ids: string;
  image_url: string;
  replica_count: number;
}

export default function DeploymentManagement() {
  const { token } = useAuthStore();
  const { isOpen, onOpen, onOpenChange } = useDisclosure();

  const [items, setItems] = useState<DeploymentItem[]>([]);
  const [keyword, setKeyword] = useState('');
  const [loading, setLoading] = useState(false);
  const [editing, setEditing] = useState<DeploymentItem | null>(null);

  const [createForm, setCreateForm] = useState<CreateDeploymentForm>({
    resource_private_name: '',
    duration_hours: 24,
    gpus_per_container: 1,
    hardware_id: 1,
    location_ids: '1',
    image_url: '',
    replica_count: 1,
  });
  const [rename, setRename] = useState('');

  const modalTitle = useMemo(() => {
    return editing ? '重命名部署' : '创建部署';
  }, [editing]);

  const parseItems = (payload: any): DeploymentItem[] => {
    if (!payload) return [];
    if (Array.isArray(payload?.items)) return payload.items;
    if (Array.isArray(payload)) return payload;
    return [];
  };

  const fetchData = async () => {
    setLoading(true);
    try {
      const q = keyword.trim();
      const url = q
        ? `/api/deployments/search?keyword=${encodeURIComponent(q)}&page=1&page_size=50`
        : '/api/deployments?page=1&page_size=50';
      const res = await fetch(url, { headers: { Authorization: `Bearer ${token}` } });
      const data = await res.json();
      if (data.code === 0) {
        setItems(parseItems(data.data));
      }
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
  }, []);

  const openCreate = () => {
    setEditing(null);
    setRename('');
    setCreateForm({
      resource_private_name: '',
      duration_hours: 24,
      gpus_per_container: 1,
      hardware_id: 1,
      location_ids: '1',
      image_url: '',
      replica_count: 1,
    });
    onOpen();
  };

  const openRename = (v: DeploymentItem) => {
    setEditing(v);
    setRename(v.deployment_name || '');
    onOpen();
  };

  const submit = async (onClose: () => void) => {
    if (editing) {
      const res = await fetch(`/api/deployments/${editing.id}/name`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({ name: rename.trim() }),
      });
      const data = await res.json();
      if (data.code === 0) {
        await fetchData();
        onClose();
      }
      return;
    }

    const locationIds = createForm.location_ids
      .split(',')
      .map(v => Number(v.trim()))
      .filter(v => Number.isFinite(v) && v > 0);

    const payload = {
      resource_private_name: createForm.resource_private_name.trim(),
      duration_hours: Number(createForm.duration_hours) || 24,
      gpus_per_container: Number(createForm.gpus_per_container) || 1,
      hardware_id: Number(createForm.hardware_id) || 1,
      location_ids: locationIds,
      container_config: {
        replica_count: Number(createForm.replica_count) || 1,
      },
      registry_config: {
        image_url: createForm.image_url.trim(),
      },
    };

    const res = await fetch('/api/deployments', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify(payload),
    });
    const data = await res.json();
    if (data.code === 0) {
      await fetchData();
      onClose();
    }
  };

  const remove = async (id: string) => {
    if (!await confirm({ title: '删除部署', message: '确认终止该部署？', danger: true })) return;
    const res = await fetch(`/api/deployments/${id}`, {
      method: 'DELETE',
      headers: { Authorization: `Bearer ${token}` },
    });
    const data = await res.json();
    if (data.code === 0) fetchData();
  };

  return (
    <div className="space-y-6">
      <PageHeader
        title="部署管理"
        description="io.net Deployments"
        actions={(
          <div className="flex gap-2">
            <Input
              size="sm"
              placeholder="搜索部署名"
              value={keyword}
              onValueChange={setKeyword}
              onKeyDown={(e) => e.key === 'Enter' && fetchData()}
            />
            <Button variant="flat" onPress={fetchData} isLoading={loading}>查询</Button>
            <Button color="primary" onPress={openCreate}>创建</Button>
          </div>
        )}
      />

      <div className="data-table-wrap">
        <table className="data-table">
          <thead>
            <tr>
              <th>ID</th>
              <th>Name</th>
              <th>Status</th>
              <th>Hardware</th>
              <th>Remaining</th>
              <th>Progress</th>
              <th>Provider</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <tr><td colSpan={8}>加载中...</td></tr>
            ) : items.length === 0 ? (
              <tr><td colSpan={8}>暂无数据</td></tr>
            ) : items.map(v => (
              <tr key={v.id}>
                <td>{v.id}</td>
                <td>{v.deployment_name || '-'}</td>
                <td>{v.status || '-'}</td>
                <td>{v.hardware_info || '-'}</td>
                <td>{v.time_remaining || '-'}</td>
                <td>{typeof v.completed_percent === 'number' ? `${v.completed_percent.toFixed(1)}%` : '-'}</td>
                <td>{v.provider || '-'}</td>
                <td>
                  <div className="flex gap-2">
                    <Button size="sm" variant="flat" onPress={() => openRename(v)}>重命名</Button>
                    <Button size="sm" color="danger" variant="flat" onPress={() => remove(v.id)}>终止</Button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <Modal isOpen={isOpen} onOpenChange={onOpenChange}>
        <ModalContent>
          {(onClose) => (
            <>
              <ModalHeader>{modalTitle}</ModalHeader>
              <ModalBody className="gap-3">
                {editing ? (
                  <Input
                    label="新名称"
                    value={rename}
                    onValueChange={setRename}
                  />
                ) : (
                  <>
                    <Input
                      label="资源名称(resource_private_name)"
                      value={createForm.resource_private_name}
                      onValueChange={(v) => setCreateForm({ ...createForm, resource_private_name: v })}
                    />
                    <Input
                      label="镜像地址(registry_config.image_url)"
                      value={createForm.image_url}
                      onValueChange={(v) => setCreateForm({ ...createForm, image_url: v })}
                    />
                    <Input
                      label="硬件ID(hardware_id)"
                      value={String(createForm.hardware_id)}
                      onValueChange={(v) => setCreateForm({ ...createForm, hardware_id: Number(v) || 1 })}
                    />
                    <Input
                      label="位置ID列表(location_ids，逗号分隔)"
                      value={createForm.location_ids}
                      onValueChange={(v) => setCreateForm({ ...createForm, location_ids: v })}
                    />
                    <Input
                      label="GPU/容器(gpus_per_container)"
                      value={String(createForm.gpus_per_container)}
                      onValueChange={(v) => setCreateForm({ ...createForm, gpus_per_container: Number(v) || 1 })}
                    />
                    <Input
                      label="副本数(container_config.replica_count)"
                      value={String(createForm.replica_count)}
                      onValueChange={(v) => setCreateForm({ ...createForm, replica_count: Number(v) || 1 })}
                    />
                    <Input
                      label="运行时长小时(duration_hours)"
                      value={String(createForm.duration_hours)}
                      onValueChange={(v) => setCreateForm({ ...createForm, duration_hours: Number(v) || 24 })}
                    />
                  </>
                )}
              </ModalBody>
              <ModalFooter>
                <Button variant="light" onPress={onClose}>取消</Button>
                <Button color="primary" onPress={() => submit(onClose)}>保存</Button>
              </ModalFooter>
            </>
          )}
        </ModalContent>
      </Modal>
    </div>
  );
}
