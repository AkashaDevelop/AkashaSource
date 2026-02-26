import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Card, CardBody, CardHeader, Button, Input, Form, Alert, Modal, ModalContent, ModalHeader, ModalBody, ModalFooter, useDisclosure } from '@heroui/react';
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
        if (data.initialized === false) {
          setInitialized(false);
        }
        if (data.options) {
          if (data.options.system_name) setSystemName(data.options.system_name);
          if (data.options.logo_url) setLogoUrl(data.options.logo_url);
          if (data.options.github_client_id) setGithubEnabled(true);
          if (data.options.linuxdo_client_id) setLinuxDOEnabled(true);
          if (data.options.discord_client_id) setDiscordEnabled(true);
          if (data.options.oidc_client_id) setOidcEnabled(true);
          if (data.options.telegram_bot_token) {
            setTelegramEnabled(true);
            setTelegramBotToken(data.options.telegram_bot_token);
          }
          if (data.options.captcha_provider) setCaptchaProvider(data.options.captcha_provider);
          if (data.options.turnstile_check_enabled === 'true') {
            setTurnstileEnabled(true);
            setTurnstileSiteKey(data.options.turnstile_site_key || '');
          }
          if (data.options.geetest_enabled === 'true') {
            setGeetestEnabled(true);
            setGeetestId(data.options.geetest_id || '');
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
      if (!res.ok) {
        if (turnstileEnabled && (window as any).turnstile) {
          (window as any).turnstile.reset();
          setTurnstileToken('');
        }
        setGeetestResult(null);
        throw new Error(data.error || 'Login failed');
      }

      // Check if 2FA is required
      if (data.requires_2fa) {
        setRequires2FA(true);
        setTotpUserId(data.user_id);
        return;
      }

      login(data.user, data.token);

      if (data.user.role >= 10) {
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
      if (!res.ok) throw new Error(data.error || '验证失败');
      login(data.user, data.token);
      if (data.user.role >= 10) navigate('/admin');
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
      if (!res.ok) throw new Error(data.error || '请求失败');
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
      if (!res.ok) throw new Error(data.error || '重置失败');
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
      <div className="flex items-center justify-center min-h-screen bg-gray-50 dark:bg-gray-900 p-4">
        <Card className="w-full max-w-md">
          <CardHeader className="flex flex-col gap-1 items-center pb-0">
            <h1 className="text-2xl font-bold">两步验证</h1>
            <p className="text-default-500 text-sm">请输入验证器应用中的6位验证码</p>
          </CardHeader>
          <CardBody className="overflow-visible py-4">
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
              <Button color="primary" isLoading={totpLoading} onPress={handle2FASubmit} className="w-full font-bold">
                验证
              </Button>
              <Button variant="light" onPress={() => { setRequires2FA(false); setTotpCode(''); setError(''); }}>
                返回登录
              </Button>
            </div>
          </CardBody>
        </Card>
      </div>
    );
  }

  return (
    <div className="flex items-center justify-center min-h-screen bg-gray-50 dark:bg-gray-900 p-4">
      <Card className="w-full max-w-md">
        <CardHeader className="flex flex-col gap-1 items-center pb-0">
          {logoUrl && <img src={logoUrl} alt="Logo" className="w-12 h-12 mb-2" />}
          <h1 className="text-2xl font-bold">{systemName}</h1>
          {!initialized && (
            <Alert color="warning" className="mt-2 text-center w-full">
              System not initialized. Please register an admin account.
            </Alert>
          )}
        </CardHeader>
        <CardBody className="overflow-visible py-4">
          {error && <Alert color="danger" className="mb-4">{error}</Alert>}
          <Form className="flex flex-col gap-4" onSubmit={handleSubmit}>
            <Input isRequired label="Username" value={username} onValueChange={setUsername} />
            <Input isRequired label="Password" type="password" value={password} onValueChange={setPassword} />

            {turnstileEnabled && (
                <div id="turnstile-widget" className="cf-turnstile" data-sitekey={turnstileSiteKey} data-callback="onTurnstileSuccess"></div>
            )}

            <Button color="primary" type="submit" isLoading={loading} className="w-full font-bold">
              Log In
            </Button>
          </Form>

          <div className="mt-2 text-right">
            <Button variant="light" size="sm" onPress={onResetOpen} className="text-default-500">
              忘记密码?
            </Button>
          </div>

          {githubEnabled && (
            <div className="mt-4">
              <Button variant="bordered" className="w-full" onPress={handleGithubLogin}>
                Sign in with GitHub
              </Button>
            </div>
          )}

          {linuxDOEnabled && (
            <div className="mt-4">
              <Button variant="bordered" className="w-full" onPress={handleLinuxDOLogin}>
                Sign in with LinuxDO
              </Button>
            </div>
          )}

          {discordEnabled && (
            <div className="mt-4">
              <Button variant="bordered" className="w-full" onPress={handleDiscordLogin}>
                Sign in with Discord
              </Button>
            </div>
          )}

          {oidcEnabled && (
            <div className="mt-4">
              <Button variant="bordered" className="w-full" onPress={handleOIDCLogin}>
                Sign in with SSO (OIDC)
              </Button>
            </div>
          )}

          {telegramEnabled && (
            <div className="mt-4 flex justify-center">
              <script async src="https://telegram.org/js/telegram-widget.js?22"
                data-telegram-login={telegramBotToken.split(':')[0]}
                data-size="large"
                data-auth-url={`${window.location.origin}/oauth/telegram/callback`}
                data-request-access="write" />
              <Button variant="bordered" className="w-full" onPress={() => {
                window.location.href = `https://oauth.telegram.org/auth?bot_id=${telegramBotToken.split(':')[0]}&origin=${window.location.origin}&return_to=${window.location.origin}/oauth/telegram/callback`;
              }}>
                Sign in with Telegram
              </Button>
            </div>
          )}

          <div className="mt-4 text-center text-small">
             <Button variant="light" onPress={() => navigate('/register')} className="text-primary">
               {initialized ? "No account? Sign up" : "Initialize System (Register Admin)"}
             </Button>
          </div>
        </CardBody>
      </Card>

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
