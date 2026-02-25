import { useState, useEffect } from 'react';
import {
  Table,
  TableHeader,
  TableColumn,
  TableBody,
  TableRow,
  TableCell,
  Chip,
  Button,
  Modal,
  ModalContent,
  ModalHeader,
  ModalBody,
  ModalFooter,
  useDisclosure,
  Input,
  Pagination,
} from '@heroui/react';
import { Plus, RefreshCw, Copy } from 'lucide-react';
import { useAuthStore } from '../../store/auth';

interface Redemption {
  id: number;
  name: string;
  code: string;
  quota: number;
  status: number;
  created_at: number;
  used_by: number;
}

export default function RedemptionManagement() {
  const [codes, setCodes] = useState<Redemption[]>([]);
  const [loading, setLoading] = useState(false);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const { token } = useAuthStore();
  const { isOpen, onOpen, onOpenChange } = useDisclosure();
  
  const [formData, setFormData] = useState({
    name: '',
    quota: '1', // Quota in USD (will convert to int64)
    count: '1',
  });

  const fetchCodes = async () => {
    setLoading(true);
    try {
      const res = await fetch(`/api/redemption?p=${page}&size=10`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (res.ok) {
        setCodes(data.data || []);
        setTotal(data.total || 0);
      }
    } catch (error) {
      console.error('Failed to fetch codes:', error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (token) fetchCodes();
  }, [page, token]);

  const handleGenerate = async (onClose: () => void) => {
    try {
      const res = await fetch('/api/redemption', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({
          name: formData.name,
          quota: parseInt(formData.quota) * 500000, // Convert $1 to 500000
          count: parseInt(formData.count),
        }),
      });
      
      if (res.ok) {
        fetchCodes();
        onClose();
      } else {
        alert('Failed to generate codes');
      }
    } catch (error) {
      console.error('Generate error:', error);
    }
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    // Could show toast
  };

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-2xl font-bold">兑换码管理</h1>
          <p className="text-default-500">生成和管理额度兑换码</p>
        </div>
        <div className="flex gap-2">
          <Button startContent={<RefreshCw size={18} />} onPress={fetchCodes} variant="flat">
            刷新
          </Button>
          <Button startContent={<Plus size={18} />} color="primary" onPress={onOpen}>
            生成兑换码
          </Button>
        </div>
      </div>

      <Table 
        aria-label="Redemption table"
        bottomContent={
          <div className="flex w-full justify-center">
            <Pagination
              isCompact
              showControls
              showShadow
              color="primary"
              page={page}
              total={Math.ceil(total / 10) || 1}
              onChange={(page) => setPage(page)}
            />
          </div>
        }
      >
        <TableHeader>
          <TableColumn>ID</TableColumn>
          <TableColumn>名称</TableColumn>
          <TableColumn>兑换码</TableColumn>
          <TableColumn>额度</TableColumn>
          <TableColumn>状态</TableColumn>
          <TableColumn>创建时间</TableColumn>
          <TableColumn>使用者ID</TableColumn>
        </TableHeader>
        <TableBody emptyContent="暂无兑换码" isLoading={loading}>
          {codes.map((item) => (
            <TableRow key={item.id}>
              <TableCell>{item.id}</TableCell>
              <TableCell>{item.name}</TableCell>
              <TableCell>
                <div className="flex items-center gap-2">
                  <span className="font-mono">{item.code}</span>
                  <Copy size={14} className="cursor-pointer text-default-400" onClick={() => copyToClipboard(item.code)} />
                </div>
              </TableCell>
              <TableCell>${(item.quota / 500000).toFixed(2)}</TableCell>
              <TableCell>
                <Chip size="sm" color={item.status === 1 ? "success" : "default"}>
                  {item.status === 1 ? "未使用" : "已使用"}
                </Chip>
              </TableCell>
              <TableCell>{new Date(item.created_at * 1000).toLocaleString()}</TableCell>
              <TableCell>{item.used_by || '-'}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>

      <Modal isOpen={isOpen} onOpenChange={onOpenChange}>
        <ModalContent>
          {(onClose) => (
            <>
              <ModalHeader>生成兑换码</ModalHeader>
              <ModalBody>
                <Input
                  label="名称"
                  placeholder="例如: 活动赠送"
                  value={formData.name}
                  onValueChange={(v) => setFormData({...formData, name: v})}
                />
                <Input
                  label="额度 ($)"
                  type="number"
                  placeholder="1"
                  value={formData.quota}
                  onValueChange={(v) => setFormData({...formData, quota: v})}
                  description="1 USD = 500000 Quota"
                />
                <Input
                  label="生成数量"
                  type="number"
                  value={formData.count}
                  onValueChange={(v) => setFormData({...formData, count: v})}
                />
              </ModalBody>
              <ModalFooter>
                <Button color="danger" variant="light" onPress={onClose}>
                  取消
                </Button>
                <Button color="primary" onPress={() => handleGenerate(onClose)}>
                  生成
                </Button>
              </ModalFooter>
            </>
          )}
        </ModalContent>
      </Modal>
    </div>
  );
}
