import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Card, CardBody, CardHeader, Button, Input, Form, Alert, Modal, ModalContent, ModalHeader, ModalBody, ModalFooter, useDisclosure } from '../components/ui';
import { useAuthStore } from '../store/auth';

export default function Login() {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();
  const { login } = useAuthStore();
  const [initialized, setInitialized] = useState(true);
  const [systemName, setSystemName] = useState('Akasha');
  const [logoUrl, setLogoUrl] = useState('');
  const [githubEnabled, setGithubEnabled] = useState(false);
  const [linuxDOEnabled, setLinuxDOEnabled] = useState(false);
  const [discordEnabled, setDiscordEnabled] = useState(false);
  const [oidcEnabled, setOidcEnabled] = useState(false);
  const [telegramEnabled, setTelegramEnabled] = useState(false);
  const [telegramBotToken, setTelegramBotToken] = useState('');
  const [turnstileEnabled, setTurnstileEnabled] = useState(false);
  const [turnstileSiteKey, setTurnstileSiteKey] = useState('');
  const [turnstileToken, setTurnstileToken] = useState('');
  const [captchaProvider, setCaptchaProvider] = useState('');
  const [geetestEnabled, setGeetestEnabled] = useState(false);
  const [geetestId, setGeetestId] = useState('');
  const [geetestResult, setGeetestResult] = useState<any>(null);

  // 2FA state
  const [requires2FA, setRequires2FA] = useState(false);
  const [totpUserId, setTotpUserId] = useState<number | null>(null);
  const [totpCode, setTotpCode] = useState('');
  const [totpLoading, setTotpLoading] = useState(false);

  // Password reset state
  const { isOpen: isResetOpen, onOpen: onResetOpen, onOpenChange: onResetOpenChange } = useDisclosure();
  const [resetEmail, setResetEmail] = useState('');
  const [resetCode, setResetCode] = useState('');
  const [resetPassword, setResetPassword] = useState('');
  const [resetStep, setResetStep] = useState<'email' | 'code'>('email');
  const [resetLoading, setResetLoading] = useState(false);
  const [resetMsg, setResetMsg] = useState('');

  useEffect(() => {
    // Check if system is initialized
    fetch('/api/system/status')
      .then(res => res.json())
      .then(data => {
        const payload = data.code === 0 ? data.data : data;
        if (payload.initialized === false) {
          setInitialized(false);
        }
        if (payload.options) {
          if (payload.options.system_name) setSystemName(payload.options.system_name);
          if (payload.options.logo_url) setLogoUrl(payload.options.logo_url);
          if (payload.options.github_client_id) setGithubEnabled(true);
          if (payload.options.linuxdo_client_id) setLinuxDOEnabled(true);
          if (payload.options.discord_client_id) setDiscordEnabled(true);
          if (payload.options.oidc_client_id) setOidcEnabled(true);
          if (payload.options.telegram_bot_token) {
            setTelegramEnabled(true);
            setTelegramBotToken(payload.options.telegram_bot_token);
          }
          if (payload.options.captcha_provider) setCaptchaProvider(payload.options.captcha_provider);
          if (payload.options.turnstile_check_enabled === 'true') {
            setTurnstileEnabled(true);
            setTurnstileSiteKey(payload.options.turnstile_site_key || '');
          }
          if (payload.options.geetest_enabled === 'true') {
            setGeetestEnabled(true);
            setGeetestId(payload.options.geetest_id || '');
          }
        }
      });
  }, []);
  
  useEffect(() => {
    if (turnstileEnabled && turnstileSiteKey && (window as any).turnstile) {
        (window as any).turnstile.render('#turnstile-widget', {
            sitekey: turnstileSiteKey,
            callback: function(token: string) {
                setTurnstileToken(token);
            },
        });
    }
  }, [turnstileEnabled, turnstileSiteKey]);

  useEffect(() => {
    if (geetestEnabled && geetestId && (window as any).initGeetest4) {
      (window as any).initGeetest4({
        captchaId: geetestId,
        product: 'bind',
      }, (captchaObj: any) => {
        (window as any)._geetestCaptcha = captchaObj;
        captchaObj.onSuccess(() => {
          setGeetestResult(captchaObj.getValidate());
        });
      });
    }
  }, [geetestEnabled, geetestId]);

  const triggerGeetest = (): Promise<any> => {
    return new Promise((resolve) => {
      const captchaObj = (window as any)._geetestCaptcha;
      if (!captchaObj) { resolve(null); return; }
      captchaObj.onSuccess(() => {
        resolve(captchaObj.getValidate());
      });
      captchaObj.showCaptcha();
    });
  };

  const handleGithubLogin = () => {
    window.location.href = '/oauth/github';
  };

  const handleLinuxDOLogin = () => {
    window.location.href = '/oauth/linuxdo';
  };

  const handleDiscordLogin = () => {
    window.location.href = '/oauth/discord';
  };

  const handleOIDCLogin = () => {
    window.location.href = '/oauth/oidc';
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const useGeetest = captchaProvider === 'geetest' && geetestEnabled;
    const useTurnstile = captchaProvider === 'turnstile' ? turnstileEnabled : (!captchaProvider && turnstileEnabled);

    if (useTurnstile && !turnstileToken) {
      setError("请完成人机验证");
      return;
    }

    let geetestData = geetestResult;
    if (useGeetest && !geetestData) {
      geetestData = await triggerGeetest();
      if (!geetestData) {
        setError("请完成人机验证");
        return;
      }
    }

    setLoading(true);
    setError('');

    try {
      const body: any = { username, password };
      if (useTurnstile) body.turnstile = turnstileToken;
      if (useGeetest && geetestData) body.geetest = geetestData;

      const res = await fetch('/api/user/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      });

      const data = await res.json();
      if (data.code !== 0) {
        if (turnstileEnabled && (window as any).turnstile) {
          (window as any).turnstile.reset();
          setTurnstileToken('');
        }
        setGeetestResult(null);
        throw new Error(data.msg || 'Login failed');
      }

      // Check if 2FA is required
      if (data.data?.requires_2fa) {
        setRequires2FA(true);
        setTotpUserId(data.data.user_id);
        return;
      }

      login(data.data.user, data.data.token);

      if (data.data.user.role >= 10) {
        navigate('/admin');
      } else {
        navigate('/');
      }
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  const handle2FASubmit = async () => {
    if (!totpCode || !totpUserId) return;
    setTotpLoading(true);
    setError('');
    try {
      const res = await fetch('/api/user/login/2fa', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ user_id: totpUserId, code: totpCode }),
      });
      const data = await res.json();
      if (data.code !== 0) throw new Error(data.msg || '验证失败');
      login(data.data.user, data.data.token);
      if (data.data.user.role >= 10) navigate('/admin');
      else navigate('/');
    } catch (err: any) {
      setError(err.message);
    } finally {
      setTotpLoading(false);
    }
  };

  const handleResetRequest = async () => {
    if (!resetEmail) return;
    setResetLoading(true);
    setResetMsg('');
    try {
      const res = await fetch('/api/user/password/reset-request', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email: resetEmail }),
      });
      const data = await res.json();
      if (data.code !== 0) throw new Error(data.msg || '请求失败');
      setResetMsg('验证码已发送到您的邮箱');
      setResetStep('code');
    } catch (err: any) {
      setResetMsg(err.message);
    } finally {
      setResetLoading(false);
    }
  };

  const handleResetConfirm = async () => {
    if (!resetCode || !resetPassword) return;
    setResetLoading(true);
    setResetMsg('');
    try {
      const res = await fetch('/api/user/password/reset-confirm', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email: resetEmail, code: resetCode, new_password: resetPassword }),
      });
      const data = await res.json();
      if (data.code !== 0) throw new Error(data.msg || '重置失败');
      setResetMsg('密码重置成功，请登录');
      setTimeout(() => { onResetOpenChange(); setResetStep('email'); setResetMsg(''); }, 1500);
    } catch (err: any) {
      setResetMsg(err.message);
    } finally {
      setResetLoading(false);
    }
  };

  // 2FA input screen
  if (requires2FA) {
    return (
      <div className="flex items-center justify-center min-h-screen p-4 relative overflow-hidden star-bg" style={{ background: 'var(--bg-base)' }}>
        <span className="auth-deco-1">✿</span>
        <span className="auth-deco-2">✦</span>
        <div className="animate-fade-in-up w-full max-w-md">
          <Card className="w-full" style={{
            background: 'var(--bg-surface)',
            backdropFilter: 'blur(16px)',
            border: '1px solid var(--border-color)',
            borderRadius: '24px',
            boxShadow: 'var(--shadow-card)',
          }}>
            <CardHeader className="flex flex-col gap-1 items-center pb-0 pt-8">
              <div className="text-3xl mb-2">🔐</div>
              <h1 className="text-2xl font-bold gradient-text">两步验证</h1>
              <p className="text-sm mt-1" style={{ color: 'var(--text-secondary)' }}>请输入验证器应用中的6位验证码</p>
            </CardHeader>
            <CardBody className="overflow-visible py-6 px-8">
              {error && <Alert color="danger" className="mb-4">{error}</Alert>}
              <div className="flex flex-col gap-4">
                <Input
                  label="验证码"
                  placeholder="000000"
                  value={totpCode}
                  onValueChange={setTotpCode}
                  maxLength={6}
                  description="也可输入备份码"
                />
                <Button
                  isLoading={totpLoading}
                  onPress={handle2FASubmit}
                  className="w-full font-bold h-11"
                  style={{ background: 'linear-gradient(135deg, var(--accent-primary), var(--accent-cosmic))', color: 'white', borderRadius: '12px' }}
                >
                  验证
                </Button>
                <Button variant="light" onPress={() => { setRequires2FA(false); setTotpCode(''); setError(''); }} style={{ color: 'var(--text-secondary)' }}>
                  返回登录
                </Button>
              </div>
            </CardBody>
          </Card>
        </div>
      </div>
    );
  }

  return (
    <div className="flex items-center justify-center min-h-screen p-4 relative overflow-hidden star-bg" style={{ background: 'var(--bg-base)' }}>
      <span className="auth-deco-1">✿</span>
      <span className="auth-deco-2">✦</span>
      <span className="auth-deco-3">❋</span>
      <span className="auth-deco-4">◈</span>

      <div className="animate-fade-in-up w-full max-w-md">
        <Card className="w-full" style={{
          background: 'var(--bg-surface)',
          backdropFilter: 'blur(16px)',
          border: '1px solid var(--border-color)',
          borderRadius: '24px',
          boxShadow: 'var(--shadow-card)',
        }}>
          <CardHeader className="flex flex-col gap-1 items-center pb-0 pt-8">
            {logoUrl
              ? <img src={logoUrl} alt="Logo" className="w-14 h-14 mb-2 rounded-full" style={{ boxShadow: '0 4px 16px var(--accent-glow)' }} />
              : <div className="text-4xl mb-2" style={{ color: 'var(--accent-primary)' }}>✿</div>
            }
            <h1 className="text-2xl font-bold gradient-text">{systemName}</h1>
            <p className="text-xs mt-1" style={{ color: 'var(--text-secondary)' }}>欢迎回来，请登录您的账号</p>
            {!initialized && (
              <Alert color="warning" className="mt-3 w-full">
                系统未初始化，请注册管理员账号
              </Alert>
            )}
          </CardHeader>

          <CardBody className="overflow-visible py-6 px-8">
            {error && <Alert color="danger" className="mb-4">{error}</Alert>}
            <Form className="flex flex-col gap-4" onSubmit={handleSubmit}>
              <Input isRequired label="用户名" placeholder="请输入用户名" value={username} onValueChange={setUsername} />
              <Input isRequired label="密码" type="password" placeholder="请输入密码" value={password} onValueChange={setPassword} />

              {turnstileEnabled && (
                <div id="turnstile-widget" className="cf-turnstile" data-sitekey={turnstileSiteKey} data-callback="onTurnstileSuccess" />
              )}

              <Button
                type="submit"
                isLoading={loading}
                className="w-full font-bold h-11 mt-1"
                style={{ background: 'linear-gradient(135deg, var(--accent-primary), var(--accent-cosmic))', color: 'white', borderRadius: '12px' }}
              >
                登录
              </Button>
            </Form>

            <div className="mt-2 text-right">
              <Button variant="light" size="sm" onPress={onResetOpen} style={{ color: 'var(--text-secondary)' }}>
                忘记密码?
              </Button>
            </div>

            {(githubEnabled || linuxDOEnabled || discordEnabled || oidcEnabled || telegramEnabled) && (
              <div className="relative my-4">
                <div className="absolute inset-0 flex items-center">
                  <div className="w-full" style={{ borderTop: '1px solid var(--border-color)' }} />
                </div>
                <div className="relative flex justify-center text-xs">
                  <span className="px-3" style={{ background: 'var(--bg-surface)', color: 'var(--text-secondary)' }}>或使用第三方登录</span>
                </div>
              </div>
            )}

            {githubEnabled && (
              <div className="mt-2">
                <Button variant="bordered" className="w-full" onPress={handleGithubLogin}
                  style={{ borderColor: 'var(--border-color)', color: 'var(--text-primary)', borderRadius: '12px' }}>
                  Sign in with GitHub
                </Button>
              </div>
            )}
            {linuxDOEnabled && (
              <div className="mt-2">
                <Button variant="bordered" className="w-full" onPress={handleLinuxDOLogin}
                  style={{ borderColor: 'var(--border-color)', color: 'var(--text-primary)', borderRadius: '12px' }}>
                  Sign in with LinuxDO
                </Button>
              </div>
            )}
            {discordEnabled && (
              <div className="mt-2">
                <Button variant="bordered" className="w-full" onPress={handleDiscordLogin}
                  style={{ borderColor: 'var(--border-color)', color: 'var(--text-primary)', borderRadius: '12px' }}>
                  Sign in with Discord
                </Button>
              </div>
            )}
            {oidcEnabled && (
              <div className="mt-2">
                <Button variant="bordered" className="w-full" onPress={handleOIDCLogin}
                  style={{ borderColor: 'var(--border-color)', color: 'var(--text-primary)', borderRadius: '12px' }}>
                  Sign in with SSO (OIDC)
                </Button>
              </div>
            )}
            {telegramEnabled && (
              <div className="mt-2">
                <script async src="https://telegram.org/js/telegram-widget.js?22"
                  data-telegram-login={telegramBotToken.split(':')[0]}
                  data-size="large"
                  data-auth-url={`${window.location.origin}/oauth/telegram/callback`}
                  data-request-access="write" />
                <Button variant="bordered" className="w-full" onPress={() => {
                  window.location.href = `https://oauth.telegram.org/auth?bot_id=${telegramBotToken.split(':')[0]}&origin=${window.location.origin}&return_to=${window.location.origin}/oauth/telegram/callback`;
                }} style={{ borderColor: 'var(--border-color)', color: 'var(--text-primary)', borderRadius: '12px' }}>
                  Sign in with Telegram
                </Button>
              </div>
            )}

            <div className="mt-5 text-center">
              <Button variant="light" onPress={() => navigate('/register')} style={{ color: 'var(--accent-primary)' }}>
                {initialized ? '没有账号？立即注册 ✦' : '初始化系统（注册管理员）'}
              </Button>
            </div>
          </CardBody>
        </Card>
      </div>

      {/* Password Reset Modal */}
      <Modal isOpen={isResetOpen} onOpenChange={onResetOpenChange}>
        <ModalContent>
          {(onClose) => (
            <>
              <ModalHeader>重置密码</ModalHeader>
              <ModalBody>
                {resetMsg && <Alert color={resetMsg.includes('成功') || resetMsg.includes('已发送') ? 'success' : 'danger'} className="mb-4">{resetMsg}</Alert>}
                {resetStep === 'email' ? (
                  <Input label="注册邮箱" placeholder="your@email.com" value={resetEmail} onValueChange={setResetEmail} />
                ) : (
                  <div className="flex flex-col gap-4">
                    <Input label="验证码" placeholder="6位验证码" value={resetCode} onValueChange={setResetCode} maxLength={6} />
                    <Input label="新密码" type="password" placeholder="至少8位" value={resetPassword} onValueChange={setResetPassword} />
                  </div>
                )}
              </ModalBody>
              <ModalFooter>
                <Button variant="light" onPress={onClose}>取消</Button>
                <Button
                  color="primary"
                  isLoading={resetLoading}
                  onPress={resetStep === 'email' ? handleResetRequest : handleResetConfirm}
                >
                  {resetStep === 'email' ? '发送验证码' : '重置密码'}
                </Button>
              </ModalFooter>
            </>
          )}
        </ModalContent>
      </Modal>
    </div>
  );
}
