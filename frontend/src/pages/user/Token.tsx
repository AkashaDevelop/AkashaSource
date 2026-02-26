import { useState, useEffect } from 'react';
import {
  Table,
  TableHeader,
  TableColumn,
  TableBody,
  TableRow,
  TableCell,
  Chip,
  Tooltip,
  Button,
  Modal,
  ModalContent,
  ModalHeader,
  ModalBody,
  ModalFooter,
  useDisclosure,
  Input,
  Snippet,
  Form,
  Select,
  SelectItem,
  Checkbox,
} from '@heroui/react';
import { Edit, Trash2, Plus, RefreshCw } from 'lucide-react';
import { useAuthStore } from '../../store/auth';

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
}

const STATUS_OPTIONS = [
  { key: '1', label: '启用' },
  { key: '2', label: '禁用' },
];

export default function TokenManagement() {
  const [tokens, setTokens] = useState<Token[]>([]);
  const [loading, setLoading] = useState(false);
  const { token: authToken } = useAuthStore();
  const { isOpen, onOpen, onOpenChange } = useDisclosure();
  const [editingToken, setEditingToken] = useState<Token | null>(null);

  // Form State
  const [name, setName] = useState('');
  const [status, setStatus] = useState('1');
  const [expiredTime, setExpiredTime] = useState('');
  const [neverExpire, setNeverExpire] = useState(true);
  const [remainQuota, setRemainQuota] = useState('0');
  const [unlimitedQuota, setUnlimitedQuota] = useState(true);
  const [allowedIps, setAllowedIps] = useState('');
  const [allowedModels, setAllowedModels] = useState('');

  const fetchTokens = async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/token', {
        headers: { Authorization: `Bearer ${authToken}` },
      });
      const data = await res.json();
      if (res.ok) {
        setTokens(data.data || []);
      }
    } catch (error) {
      console.error('获取令牌失败:', error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (authToken) fetchTokens();
  }, [authToken]);

  const handleEdit = (token: Token) => {
    setEditingToken(token);
    setName(token.name);
    setStatus(token.status.toString());
    
    if (token.expired_time === -1) {
        setNeverExpire(true);
        setExpiredTime('');
    } else {
        setNeverExpire(false);
        // Convert timestamp to datetime-local string (YYYY-MM-DDTHH:mm)
        const date = new Date(token.expired_time * 1000);
        const iso = date.toISOString().slice(0, 16);
        setExpiredTime(iso); // Simplification, timezone might be an issue but ok for now
    }

    setUnlimitedQuota(token.unlimited_quota);
    setRemainQuota(token.unlimited_quota ? '0' : (token.remain_quota / 500000).toString());
    setAllowedIps(token.allowed_ips || '');
    setAllowedModels(token.allowed_models || '');
    
    onOpen();
  };

  const handleAdd = () => {
    setEditingToken(null);
    setName('');
    setStatus('1');
    setNeverExpire(true);
    setExpiredTime('');
    setUnlimitedQuota(true);
    setRemainQuota('0');
    setAllowedIps('');
    setAllowedModels('');
    onOpen();
  };

  const handleSubmit = async (onClose: () => void) => {
    const url = '/api/token';
    const method = editingToken ? 'PUT' : 'POST';
    
    let expTime = -1;
    if (!neverExpire && expiredTime) {
        expTime = Math.floor(new Date(expiredTime).getTime() / 1000);
    }

    const body = {
      id: editingToken?.id,
      name,
      status: parseInt(status),
      expired_time: expTime,
      unlimited_quota: unlimitedQuota,
      remain_quota: unlimitedQuota ? 0 : parseFloat(remainQuota) * 500000,
      allowed_ips: allowedIps,
      allowed_models: allowedModels,
    };

    try {
      const res = await fetch(url, {
        method,
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${authToken}`,
        },
        body: JSON.stringify(body),
      });

      if (res.ok) {
        fetchTokens();
        onClose();
      } else {
        alert('操作失败');
      }
    } catch (error) {
      console.error('提交失败:', error);
    }
  };

  const handleDelete = async (id: number) => {
    if (!confirm('确定要删除这个令牌吗?')) return;
    try {
      const res = await fetch(`/api/token/${id}`, {
        method: 'DELETE',
        headers: { Authorization: `Bearer ${authToken}` },
      });
      if (res.ok) fetchTokens();
    } catch (error) {
      console.error('删除失败:', error);
    }
  };

  const handleStatusToggle = async (token: Token) => {
      const newStatus = token.status === 1 ? 2 : 1;
      // Optimistic update
      setTokens(prev => prev.map(t => t.id === token.id ? { ...t, status: newStatus } : t));
      
      try {
        await fetch('/api/token', {
            method: 'PUT',
            headers: {
                'Content-Type': 'application/json',
                Authorization: `Bearer ${authToken}`,
            },
            body: JSON.stringify({ ...token, status: newStatus }),
        });
      } catch (error) {
          console.error('更新状态失败:', error);
          fetchTokens();
      }
  };

  const formatQuota = (quota: number) => {
    return `$${(quota / 500000).toFixed(2)}`;
  };

  const renderStatus = (status: number) => {
      if (status === 1) return <Chip color="success" size="sm" variant="flat">启用</Chip>;
      if (status === 2) return <Chip color="danger" size="sm" variant="flat">禁用</Chip>;
      return <Chip color="default" size="sm" variant="flat">未知</Chip>;
  };

  return (
    <div className="space-y-6 p-6">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-2xl font-bold">令牌管理</h1>
          <p className="text-default-500">管理您的 API 访问密钥</p>
        </div>
        <div className="flex gap-2">
          <Button startContent={<RefreshCw size={18} />} onPress={fetchTokens} variant="flat">
            刷新
          </Button>
          <Button startContent={<Plus size={18} />} color="primary" onPress={handleAdd}>
            添加令牌
          </Button>
        </div>
      </div>

      <Table aria-label="令牌表格">
        <TableHeader>
          <TableColumn>名称</TableColumn>
          <TableColumn>状态</TableColumn>
          <TableColumn>已用额度</TableColumn>
          <TableColumn>剩余额度</TableColumn>
          <TableColumn>IP 限制</TableColumn>
          <TableColumn>模型限制</TableColumn>
          <TableColumn>过期时间</TableColumn>
          <TableColumn>创建时间</TableColumn>
          <TableColumn>密钥</TableColumn>
          <TableColumn>操作</TableColumn>
        </TableHeader>
        <TableBody emptyContent="暂无令牌" isLoading={loading}>
          {tokens.map((token) => (
            <TableRow key={token.id}>
              <TableCell className="font-medium">{token.name}</TableCell>
              <TableCell>
                  <div className="cursor-pointer" onClick={() => handleStatusToggle(token)}>
                    {renderStatus(token.status)}
                  </div>
              </TableCell>
              <TableCell>{formatQuota(token.used_quota)}</TableCell>
              <TableCell>
                {token.unlimited_quota ? (
                  <Chip size="sm" variant="dot" color="success">无限</Chip>
                ) : (
                  formatQuota(token.remain_quota)
                )}
              </TableCell>
              <TableCell className="max-w-[180px] truncate">{token.allowed_ips || '-'}</TableCell>
              <TableCell className="max-w-[180px] truncate">{token.allowed_models || '-'}</TableCell>
              <TableCell>
                  {token.expired_time === -1 ? (
                      <Chip size="sm" variant="flat">永不过期</Chip>
                  ) : (
                      new Date(token.expired_time * 1000).toLocaleString()
                  )}
              </TableCell>
              <TableCell>{new Date(token.created_time * 1000).toLocaleString()}</TableCell>
              <TableCell>
                <Snippet symbol="" codeString={token.key} color="default" size="sm">
                  {token.key.substring(0, 8) + '***' + token.key.substring(token.key.length - 4)}
                </Snippet>
              </TableCell>
              <TableCell>
                <div className="flex items-center gap-2">
                  <Tooltip content="编辑">
                    <span 
                      className="text-lg text-default-400 cursor-pointer active:opacity-50"
                      onClick={() => handleEdit(token)}
                    >
                      <Edit size={18} />
                    </span>
                  </Tooltip>
                  <Tooltip color="danger" content="删除">
                    <span 
                      className="text-lg text-danger cursor-pointer active:opacity-50"
                      onClick={() => handleDelete(token.id)}
                    >
                      <Trash2 size={18} />
                    </span>
                  </Tooltip>
                </div>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>

      <Modal isOpen={isOpen} onOpenChange={onOpenChange}>
        <ModalContent>
          {(onClose) => (
            <>
              <ModalHeader>{editingToken ? '编辑令牌' : '添加新令牌'}</ModalHeader>
              <ModalBody>
                <Form className="flex flex-col gap-4">
                  <Input
                    label="名称"
                    placeholder="请输入名称"
                    value={name}
                    onValueChange={setName}
                    isRequired
                  />
                  
                  <Select 
                    label="状态" 
                    selectedKeys={[status]} 
                    onChange={(e) => setStatus(e.target.value)}
                  >
                    {STATUS_OPTIONS.map((opt) => (
                      <SelectItem key={opt.key}>{opt.label}</SelectItem>
                    ))}
                  </Select>

                  <div className="flex flex-col gap-2 p-2 border rounded-lg border-default-200">
                    <Checkbox isSelected={neverExpire} onValueChange={setNeverExpire}>
                        永不过期
                    </Checkbox>
                    {!neverExpire && (
                        <Input 
                            type="datetime-local" 
                            label="过期时间" 
                            value={expiredTime}
                            onValueChange={setExpiredTime}
                        />
                    )}
                  </div>

                  <div className="flex flex-col gap-2 p-2 border rounded-lg border-default-200">
                    <Checkbox isSelected={unlimitedQuota} onValueChange={setUnlimitedQuota}>
                        无限额度
                    </Checkbox>
                    {!unlimitedQuota && (
                        <Input 
                            type="number" 
                            label="剩余额度 ($)" 
                            value={remainQuota}
                            onValueChange={setRemainQuota}
                            description="1 美元 = 500000 额度"
                        />
                    )}
                  </div>

                  <Input
                    label="允许的 IP（逗号分隔）"
                    value={allowedIps}
                    onValueChange={setAllowedIps}
                    placeholder="例如：1.1.1.1,2.2.2.2"
                  />
                  <Input
                    label="允许的模型（逗号分隔）"
                    value={allowedModels}
                    onValueChange={setAllowedModels}
                    placeholder="例如：gpt-4o-mini,qwen-max"
                  />

                </Form>
              </ModalBody>
              <ModalFooter>
                <Button color="danger" variant="light" onPress={onClose}>
                  取消
                </Button>
                <Button color="primary" onPress={() => handleSubmit(onClose)}>
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
