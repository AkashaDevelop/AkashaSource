import { useState, useEffect } from 'react';
import {
  Card,
  CardBody,
  CardHeader,
  Input,
  Button,
  Divider,
} from '@heroui/react';
import { CreditCard, DollarSign } from 'lucide-react';
import { useAuthStore } from '../../store/auth';

export default function TopupPage() {
  const [code, setCode] = useState('');
  const [loading, setLoading] = useState(false);
  const { token, user, updateUser } = useAuthStore();
  const [message, setMessage] = useState<{ type: 'success' | 'danger'; text: string } | null>(null);

  const fetchUser = async () => {
    try {
        const res = await fetch('/api/user/self', {
            headers: { Authorization: `Bearer ${token}` },
        });
        const data = await res.json();
        if (res.ok) {
            updateUser(data.data);
        }
    } catch (e) {
        console.error(e);
    }
  }

  useEffect(() => {
    fetchUser();
  }, []);

  const handleRedeem = async () => {
    if (!code) return;
    setLoading(true);
    setMessage(null);

    try {
      const res = await fetch('/api/user/redemption', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({ code }),
      });
      const data = await res.json();
      
      if (res.ok) {
        setMessage({ type: 'success', text: `兑换成功！增加额度 $${(data.quota / 500000).toFixed(2)}` });
        setCode('');
        fetchUser(); // Refresh user balance
      } else {
        setMessage({ type: 'danger', text: data.error || '兑换失败' });
      }
    } catch (error) {
      console.error('Redeem error:', error);
      setMessage({ type: 'danger', text: '请求失败' });
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="space-y-6 max-w-2xl mx-auto p-6">
      <div className="flex flex-col gap-2">
        <h1 className="text-2xl font-bold">充值中心</h1>
        <p className="text-default-500">使用兑换码增加账户额度</p>
      </div>

      <Card className="p-4">
        <CardHeader className="flex gap-3">
          <div className="p-2 bg-primary/10 rounded-lg text-primary">
            <DollarSign size={24} />
          </div>
          <div className="flex flex-col">
            <p className="text-md font-semibold">当前余额</p>
            <p className="text-small text-default-500">1 USD = 500000 Quota</p>
          </div>
        </CardHeader>
        <Divider />
        <CardBody className="py-8">
          <div className="flex flex-col items-center justify-center gap-2">
            <span className="text-4xl font-bold text-success">
              ${user ? (user.quota / 500000).toFixed(2) : '0.00'}
            </span>
            <span className="text-small text-default-400">可用额度</span>
          </div>
        </CardBody>
      </Card>

      <Card className="p-4">
        <CardHeader>
          <h3 className="text-lg font-semibold">兑换码充值</h3>
        </CardHeader>
        <CardBody className="gap-4">
          <Input
            label="兑换码"
            placeholder="请输入您的兑换码"
            value={code}
            onValueChange={setCode}
            startContent={<CreditCard className="text-default-400" size={18} />}
          />
          {message && (
            <div className={`text-small ${message.type === 'success' ? 'text-success' : 'text-danger'}`}>
              {message.text}
            </div>
          )}
          <Button 
            color="primary" 
            fullWidth 
            onPress={handleRedeem}
            isLoading={loading}
            isDisabled={!code}
          >
            立即兑换
          </Button>
        </CardBody>
      </Card>
    </div>
  );
}
