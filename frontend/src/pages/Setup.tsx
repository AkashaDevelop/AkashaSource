import { useEffect, useState, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { Card, CardBody, CardHeader, Button, Input, Select, SelectItem, Form, Alert, Switch } from '../components/ui';

type DBDriver = 'sqlite' | 'mysql' | 'postgres';
type Step = 'loading' | 'database' | 'account' | 'done';

interface SetupStatus {
  root_init: boolean;
  db_connected: boolean;
  db_configured: boolean;
  database_type: string;
}

const DB_DEFAULT_PORTS: Record<DBDriver, string> = {
  sqlite: '',
  mysql: '3306',
  postgres: '5432',
};

const DB_LABELS: Record<DBDriver, string> = {
  sqlite: 'SQLite',
  mysql: 'MySQL',
  postgres: 'PostgreSQL',
};

function buildDSN(driver: DBDriver, f: Record<string, string>): string {
  if (driver === 'sqlite') return f.sqlitePath || 'akasha.db';
  if (driver === 'mysql') {
    return `${f.user}:${f.password}@tcp(${f.host}:${f.port})/${f.dbname}?charset=utf8mb4&parseTime=True&loc=Local`;
  }
  return `host=${f.host} user=${f.user} password=${f.password} dbname=${f.dbname} port=${f.port} sslmode=disable`;
}

export default function Setup() {
  const navigate = useNavigate();
  const [step, setStep] = useState<Step>('loading');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [setupStatus, setSetupStatus] = useState<SetupStatus | null>(null);

  // Step 1: 数据库
  const [driver, setDriver] = useState<DBDriver>('sqlite');
  const [dbFields, setDbFields] = useState({
    sqlitePath: 'akasha.db',
    host: '127.0.0.1',
    port: '3306',
    user: '',
    password: '',
    dbname: '',
  });
  const [dbConnected, setDbConnected] = useState(false);
  const [showDbForm, setShowDbForm] = useState(false);

  // Step 2: 超管账号
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [selfUseMode, setSelfUseMode] = useState(true);
  const [demoSite, setDemoSite] = useState(false);

  const checkStatus = useCallback(async () => {
    try {
      const res = await fetch('/api/setup');
      const data = await res.json();
      const payload: SetupStatus = data.code === 0 ? data.data : data;
      setSetupStatus(payload);

      if (payload.root_init) {
        setStep('done');
        navigate('/login');
        return;
      }
      // 始终从数据库配置步骤开始，即使 DB 已通过默认 SQLite 连接
      setStep('database');
      setDbConnected(payload.db_connected);
      // 只有用户显式配置过数据库，才跳过表单直接展示已连接状态
      setShowDbForm(!payload.db_configured);
      if (payload.database_type === 'mysql' || payload.database_type === 'postgres') {
        setDriver(payload.database_type as DBDriver);
        setDbFields(f => ({ ...f, port: DB_DEFAULT_PORTS[payload.database_type as DBDriver] }));
      }
    } catch {
      setStep('database');
      setShowDbForm(true);
    }
  }, [navigate]);

  useEffect(() => { checkStatus(); }, [checkStatus]);

  const handleDriverChange = (d: DBDriver) => {
    setDriver(d);
    setDbFields(f => ({ ...f, port: DB_DEFAULT_PORTS[d] }));
  };

  const updateField = (key: string, value: string) => {
    setDbFields(prev => ({ ...prev, [key]: value }));
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
      setDbConnected(true);
      setShowDbForm(false);
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  const proceedToAccount = () => {
    setError('');
    setStep('account');
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
      setStep('done');
      setTimeout(() => navigate('/login'), 1500);
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  const stepNumber = step === 'database' ? 1 : step === 'account' ? 2 : 3;
  const dbTypeLabel = setupStatus?.database_type
    ? DB_LABELS[setupStatus.database_type as DBDriver] || setupStatus.database_type
    : 'SQLite';

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

            {/* 步骤指示器 */}
            {step !== 'loading' && (
              <div className="flex items-center gap-2 mt-3 mb-1">
                {[1, 2, 3].map(n => (
                  <div key={n} className="flex items-center gap-2">
                    <div
                      className="w-7 h-7 rounded-full flex items-center justify-center text-xs font-bold transition-all duration-300"
                      style={{
                        background: n < stepNumber
                          ? 'var(--accent-primary)'
                          : n === stepNumber
                            ? 'var(--accent-primary)'
                            : 'var(--bg-elevated)',
                        color: n <= stepNumber ? '#fff' : 'var(--text-muted)',
                        border: n > stepNumber ? '1px solid var(--border-color)' : 'none',
                        opacity: n > stepNumber ? 0.5 : 1,
                      }}
                    >
                      {n < stepNumber ? '✓' : n}
                    </div>
                    {n < 3 && (
                      <div className="w-8 h-0.5 rounded-full transition-all duration-300"
                        style={{ background: n < stepNumber ? 'var(--accent-primary)' : 'var(--border-color)' }} />
                    )}
                  </div>
                ))}
              </div>
            )}

            <p className="text-xs mt-1" style={{ color: 'var(--text-secondary)' }}>
              {step === 'database' && '第一步：配置数据库'}
              {step === 'account' && '第二步：创建超级管理员'}
              {step === 'done' && '初始化完成'}
              {step === 'loading' && '正在检查系统状态...'}
            </p>
          </CardHeader>

          <CardBody className="overflow-visible py-6 px-8">
            {error && <Alert color="danger" className="mb-4">{error}</Alert>}

            {/* ── Step 1: 数据库配置 ── */}
            {step === 'database' && (
              <>
                {/* 已连接状态卡片（用户未展开重新配置表单时显示） */}
                {dbConnected && !showDbForm && (
                  <div className="flex flex-col gap-4">
                    <div className="p-4 rounded-2xl flex items-center gap-3" style={{
                      background: 'var(--bg-elevated)',
                      border: '1px solid var(--accent-primary)',
                    }}>
                      <div className="w-10 h-10 rounded-full flex items-center justify-center flex-shrink-0"
                        style={{ background: 'var(--accent-glow)' }}>
                        <span style={{ color: 'var(--accent-primary)' }}>✓</span>
                      </div>
                      <div className="flex-1 min-w-0">
                        <div className="text-sm font-semibold" style={{ color: 'var(--text-primary)' }}>
                          数据库已连接
                        </div>
                        <div className="text-xs" style={{ color: 'var(--text-secondary)' }}>
                          当前类型：{dbTypeLabel}
                          {setupStatus?.db_configured && '（已持久化配置）'}
                        </div>
                      </div>
                    </div>

                    <div className="flex gap-3">
                      <Button
                        className="flex-1 font-bold h-11"
                        style={{ background: 'linear-gradient(135deg, var(--accent-primary), var(--accent-cosmic))', color: 'white', borderRadius: '12px' }}
                        onPress={proceedToAccount}
                      >
                        使用此数据库
                      </Button>
                      <Button
                        variant="bordered"
                        className="h-11"
                        style={{ borderRadius: '12px' }}
                        onPress={() => { setShowDbForm(true); setError(''); }}
                      >
                        重新配置
                      </Button>
                    </div>
                  </div>
                )}

                {/* 数据库配置表单（首次配置或重新配置时显示） */}
                {(showDbForm || !dbConnected) && (
                  <Form className="flex flex-col gap-4" onSubmit={submitDatabase}>
                    <Select
                      label="数据库类型"
                      selectedKeys={[driver]}
                      onSelectionChange={keys => handleDriverChange(([...keys][0] as DBDriver) || 'sqlite')}
                    >
                      <SelectItem key="sqlite">SQLite（单文件，适合个人/小规模使用）</SelectItem>
                      <SelectItem key="mysql">MySQL 5.7+</SelectItem>
                      <SelectItem key="postgres">PostgreSQL 9.6+</SelectItem>
                    </Select>

                    {driver === 'sqlite' ? (
                      <Input
                        label="数据库文件路径"
                        placeholder="akasha.db"
                        value={dbFields.sqlitePath}
                        onValueChange={v => updateField('sqlitePath', v)}
                      />
                    ) : (
                      <>
                        <div className="grid grid-cols-2 gap-4">
                          <Input
                            isRequired
                            label="主机地址"
                            placeholder="127.0.0.1"
                            value={dbFields.host}
                            onValueChange={v => updateField('host', v)}
                          />
                          <Input
                            isRequired
                            label="端口"
                            placeholder={DB_DEFAULT_PORTS[driver]}
                            value={dbFields.port}
                            onValueChange={v => updateField('port', v)}
                          />
                        </div>
                        <div className="grid grid-cols-2 gap-4">
                          <Input
                            isRequired
                            label="用户名"
                            value={dbFields.user}
                            onValueChange={v => updateField('user', v)}
                          />
                          <Input
                            isRequired
                            label="密码"
                            type="password"
                            value={dbFields.password}
                            onValueChange={v => updateField('password', v)}
                          />
                        </div>
                        <Input
                          isRequired
                          label="数据库名"
                          placeholder="akasha"
                          value={dbFields.dbname}
                          onValueChange={v => updateField('dbname', v)}
                        />
                      </>
                    )}

                    <Button
                      type="submit"
                      isLoading={loading}
                      className="w-full font-bold h-11 mt-1"
                      style={{ background: 'linear-gradient(135deg, var(--accent-primary), var(--accent-cosmic))', color: 'white', borderRadius: '12px' }}
                    >
                      测试并连接
                    </Button>

                    {dbConnected && (
                      <Button
                        variant="light"
                        className="w-full"
                        style={{ color: 'var(--text-secondary)' }}
                        onPress={() => { setShowDbForm(false); setError(''); }}
                      >
                        返回已连接状态
                      </Button>
                    )}
                  </Form>
                )}
              </>
            )}

            {/* ── Step 2: 创建超管账号 ── */}
            {step === 'account' && (
              <Form className="flex flex-col gap-4" onSubmit={submitAccount}>
                <Input isRequired label="用户名" placeholder="最多 12 个字符" value={username} onValueChange={setUsername} />
                <Input isRequired label="密码" type="password" placeholder="至少 8 位" value={password} onValueChange={setPassword} />
                <Input isRequired label="确认密码" type="password" value={confirmPassword} onValueChange={setConfirmPassword} />
                <Switch isSelected={selfUseMode} onValueChange={setSelfUseMode}>自用模式（关闭注册与充值等公共功能）</Switch>
                <Switch isSelected={demoSite} onValueChange={setDemoSite}>演示站点模式</Switch>

                <div className="flex gap-3 mt-1">
                  <Button
                    variant="bordered"
                    className="h-11"
                    style={{ borderRadius: '12px' }}
                    onPress={() => setStep('database')}
                  >
                    上一步
                  </Button>
                  <Button
                    type="submit"
                    isLoading={loading}
                    className="flex-1 font-bold h-11"
                    style={{ background: 'linear-gradient(135deg, var(--accent-primary), var(--accent-cosmic))', color: 'white', borderRadius: '12px' }}
                  >
                    创建管理员账号
                  </Button>
                </div>
              </Form>
            )}

            {/* ── Step 3: 完成 ── */}
            {step === 'done' && (
              <div className="text-center py-8 flex flex-col items-center gap-4">
                <div className="w-16 h-16 rounded-full flex items-center justify-center"
                  style={{ background: 'var(--accent-glow)' }}>
                  <span className="text-3xl" style={{ color: 'var(--accent-primary)' }}>✓</span>
                </div>
                <div>
                  <div className="text-lg font-bold" style={{ color: 'var(--text-primary)' }}>初始化完成</div>
                  <div className="text-xs mt-1" style={{ color: 'var(--text-secondary)' }}>即将跳转到登录页...</div>
                </div>
              </div>
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
