import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Card, CardBody, CardHeader, Button, Input, Select, SelectItem, Form, Alert, Switch } from '../components/ui';

type DBDriver = 'sqlite' | 'mysql' | 'postgres';

function buildDSN(driver: DBDriver, fields: Record<string, string>): string {
  if (driver === 'sqlite') return fields.sqlitePath || 'akasha.db';
  if (driver === 'mysql') {
    return `${fields.user}:${fields.password}@tcp(${fields.host}:${fields.port})/${fields.dbname}?charset=utf8mb4&parseTime=True&loc=Local`;
  }
  return `host=${fields.host} user=${fields.user} password=${fields.password} dbname=${fields.dbname} port=${fields.port} sslmode=disable`;
}

export default function Setup() {
  const navigate = useNavigate();
  const [step, setStep] = useState<'loading' | 'database' | 'account' | 'done'>('loading');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  // Step 1: 数据库连接
  const [driver, setDriver] = useState<DBDriver>('sqlite');
  const [dbFields, setDbFields] = useState({ sqlitePath: 'akasha.db', host: '127.0.0.1', port: driver === 'postgres' ? '5432' : '3306', user: '', password: '', dbname: '' });

  // Step 2: 超管账号
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [selfUseMode, setSelfUseMode] = useState(true);
  const [demoSite, setDemoSite] = useState(false);

  const checkStatus = async () => {
    try {
      const res = await fetch('/api/setup');
      const data = await res.json();
      const payload = data.code === 0 ? data.data : data;
      if (payload.root_init) {
        setStep('done');
        navigate('/login');
        return;
      }
      setStep(payload.db_connected ? 'account' : 'database');
    } catch {
      setStep('database');
    }
  };

  useEffect(() => { checkStatus(); }, []);

  const handleDriverChange = (d: DBDriver) => {
    setDriver(d);
    setDbFields(f => ({ ...f, port: d === 'postgres' ? '5432' : '3306' }));
  };

  const submitDatabase = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError('');
    try {
      const dsn = buildDSN(driver, dbFields);
      const res = await fetch('/api/setup/database', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ driver, dsn }),
      });
      const data = await res.json();
      if (data.code !== 0) throw new Error(data.msg || '连接失败');
      setStep('account');
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  const submitAccount = async (e: React.FormEvent) => {
    e.preventDefault();
    if (password !== confirmPassword) {
      setError('两次输入的密码不一致');
      return;
    }
    setLoading(true);
    setError('');
    try {
      const res = await fetch('/api/setup', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          username, password, confirmPassword,
          SelfUseModeEnabled: selfUseMode,
          DemoSiteEnabled: demoSite,
        }),
      });
      const data = await res.json();
      if (data.code !== 0) throw new Error(data.msg || '创建管理员账号失败');
      navigate('/login');
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex items-center justify-center min-h-screen p-4 relative overflow-hidden star-bg" style={{ background: 'var(--bg-base)' }}>
      <span className="auth-deco-1">✿</span>
      <span className="auth-deco-2">✦</span>
      <span className="auth-deco-3">❋</span>

      <div className="animate-fade-in-up w-full max-w-lg">
        <Card className="w-full" style={{
          background: 'var(--bg-surface)',
          backdropFilter: 'blur(16px)',
          border: '1px solid var(--border-color)',
          borderRadius: '24px',
          boxShadow: 'var(--shadow-card)',
        }}>
          <CardHeader className="flex flex-col gap-1 items-center pb-0 pt-8">
            <div className="text-4xl mb-2" style={{ color: 'var(--accent-primary)' }}>✿</div>
            <h1 className="text-2xl font-bold gradient-text">初始化向导</h1>
            <p className="text-xs mt-1" style={{ color: 'var(--text-secondary)' }}>
              {step === 'database' ? '第一步：连接数据库' : step === 'account' ? '第二步：创建超级管理员账号' : '正在检查系统状态...'}
            </p>
          </CardHeader>

          <CardBody className="overflow-visible py-6 px-8">
            {error && <Alert color="danger" className="mb-4">{error}</Alert>}

            {step === 'database' && (
              <Form className="flex flex-col gap-4" onSubmit={submitDatabase}>
                <Select label="数据库类型" selectedKeys={[driver]} onSelectionChange={keys => handleDriverChange(([...keys][0] as DBDriver) || 'sqlite')}>
                  <SelectItem key="sqlite">SQLite（单文件，适合个人/小规模使用）</SelectItem>
                  <SelectItem key="mysql">MySQL</SelectItem>
                  <SelectItem key="postgres">PostgreSQL</SelectItem>
                </Select>

                {driver === 'sqlite' ? (
                  <Input label="数据库文件路径" placeholder="akasha.db" value={dbFields.sqlitePath}
                    onValueChange={v => setDbFields({ ...dbFields, sqlitePath: v })} />
                ) : (
                  <>
                    <div className="grid grid-cols-2 gap-4">
                      <Input isRequired label="主机地址" placeholder="127.0.0.1" value={dbFields.host}
                        onValueChange={v => setDbFields({ ...dbFields, host: v })} />
                      <Input isRequired label="端口" placeholder={driver === 'postgres' ? '5432' : '3306'} value={dbFields.port}
                        onValueChange={v => setDbFields({ ...dbFields, port: v })} />
                    </div>
                    <div className="grid grid-cols-2 gap-4">
                      <Input isRequired label="用户名" value={dbFields.user}
                        onValueChange={v => setDbFields({ ...dbFields, user: v })} />
                      <Input isRequired label="密码" type="password" value={dbFields.password}
                        onValueChange={v => setDbFields({ ...dbFields, password: v })} />
                    </div>
                    <Input isRequired label="数据库名" value={dbFields.dbname}
                      onValueChange={v => setDbFields({ ...dbFields, dbname: v })} />
                  </>
                )}

                <Button type="submit" isLoading={loading} className="w-full font-bold h-11 mt-1"
                  style={{ background: 'linear-gradient(135deg, var(--accent-primary), var(--accent-cosmic))', color: 'white', borderRadius: '12px' }}>
                  测试并连接
                </Button>
              </Form>
            )}

            {step === 'account' && (
              <Form className="flex flex-col gap-4" onSubmit={submitAccount}>
                <Input isRequired label="用户名" placeholder="最多 12 个字符" value={username} onValueChange={setUsername} />
                <Input isRequired label="密码" type="password" placeholder="至少 8 位" value={password} onValueChange={setPassword} />
                <Input isRequired label="确认密码" type="password" value={confirmPassword} onValueChange={setConfirmPassword} />
                <Switch isSelected={selfUseMode} onValueChange={setSelfUseMode}>自用模式（关闭注册与充值等公共功能）</Switch>
                <Switch isSelected={demoSite} onValueChange={setDemoSite}>演示站点模式</Switch>

                <Button type="submit" isLoading={loading} className="w-full font-bold h-11 mt-1"
                  style={{ background: 'linear-gradient(135deg, var(--accent-primary), var(--accent-cosmic))', color: 'white', borderRadius: '12px' }}>
                  创建管理员账号
                </Button>
              </Form>
            )}

            {step === 'loading' && (
              <div className="text-center py-6" style={{ color: 'var(--text-secondary)' }}>正在检查系统状态...</div>
            )}
          </CardBody>
        </Card>
      </div>
    </div>
  );
}
