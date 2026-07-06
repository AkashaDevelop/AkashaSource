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
  const [notice, setNotice] = useState('');
  const [githubEnabled, setGithubEnabled] = useState(false);
  const [linuxDOEnabled, setLinuxDOEnabled] = useState(false);
  const [discordEnabled, setDiscordEnabled] = useState(false);
  const [oidcEnabled, setOidcEnabled] = useState(false);
  const [telegramEnabled, setTelegramEnabled] = useState(false);
  const [telegramBotToken, setTelegramBotToken] = useState('');
  const [wechatEnabled, setWechatEnabled] = useState(false);
  const [turnstileEnabled, setTurnstileEnabled] = useState(false);
  const [turnstileSiteKey, setTurnstileSiteKey] = useState('');
  const [turnstileToken, setTurnstileToken] = useState('');
  const [captchaProvider, setCaptchaProvider] = useState('');
  const [geetestEnabled, setGeetestEnabled] = useState(false);
  const [geetestId, setGeetestId] = useState('');
  const [geetestResult, setGeetestResult] = useState<any>(null);
  const [hcaptchaEnabled, setHcaptchaEnabled] = useState(false);
  const [hcaptchaSiteKey, setHcaptchaSiteKey] = useState('');
  const [hcaptchaToken, setHcaptchaToken] = useState('');
  const [recaptchaEnabled, setRecaptchaEnabled] = useState(false);
  const [recaptchaSiteKey, setRecaptchaSiteKey] = useState('');
  const [recaptchaToken, setRecaptchaToken] = useState('');
  const [recaptchaVersion, setRecaptchaVersion] = useState('v2');
  const [registerEnabled, setRegisterEnabled] = useState(true);
  const [passwordLoginEnabled, setPasswordLoginEnabled] = useState(true);
  const [passkeyEnabled, setPasskeyEnabled] = useState(false);
  const [passkeyLoading, setPasskeyLoading] = useState(false);

  // 2FA state
  const [requires2FA, setRequires2FA] = useState(false);
  const [totpUserId, setTotpUserId] = useState<number | null>(null);
  const [preAuthTicket, setPreAuthTicket] = useState<string | null>(null);
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
          if (payload.options.notice) setNotice(payload.options.notice);
          if (payload.options.github_client_id) setGithubEnabled(true);
          if (payload.options.linuxdo_client_id) setLinuxDOEnabled(true);
          if (payload.options.discord_client_id) setDiscordEnabled(true);
          if (payload.options.oidc_client_id) setOidcEnabled(true);
          if (payload.options.telegram_bot_token) {
            setTelegramEnabled(true);
            setTelegramBotToken(payload.options.telegram_bot_token);
          }
          if (payload.options.wechat_app_id) setWechatEnabled(true);
          if (payload.options.captcha_provider) setCaptchaProvider(payload.options.captcha_provider);
          if (payload.options.turnstile_check_enabled === 'true') {
            setTurnstileEnabled(true);
            setTurnstileSiteKey(payload.options.turnstile_site_key || '');
          }
          if (payload.options.geetest_enabled === 'true') {
            setGeetestEnabled(true);
            setGeetestId(payload.options.geetest_id || '');
          }
          if (payload.options.hcaptcha_enabled === 'true') {
            setHcaptchaEnabled(true);
            setHcaptchaSiteKey(payload.options.hcaptcha_site_key || '');
          }
          if (payload.options.recaptcha_enabled === 'true') {
            setRecaptchaEnabled(true);
            setRecaptchaSiteKey(payload.options.recaptcha_site_key || '');
          }
          if (payload.options.recaptcha_version) setRecaptchaVersion(payload.options.recaptcha_version);
          if (payload.options.register_enabled !== undefined) setRegisterEnabled(payload.options.register_enabled !== 'false');
          if (payload.options.password_login_enabled !== undefined) setPasswordLoginEnabled(payload.options.password_login_enabled !== 'false');
          if (payload.options.passkey_enabled === 'true') setPasskeyEnabled(true);
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

  useEffect(() => {
    if (hcaptchaEnabled && hcaptchaSiteKey && (window as any).hcaptcha) {
      try {
        (window as any).hcaptcha.render('#hcaptcha-widget', {
          sitekey: hcaptchaSiteKey,
          callback: (token: string) => setHcaptchaToken(token),
        });
      } catch (e) { /* already rendered */ }
    }
  }, [hcaptchaEnabled, hcaptchaSiteKey]);

  useEffect(() => {
    if (!recaptchaEnabled || !recaptchaSiteKey) return;
    const existing = document.querySelector('#recaptcha-api-script');
    if (existing) return;
    const script = document.createElement('script');
    script.id = 'recaptcha-api-script';
    if (recaptchaVersion === 'v3') {
      script.src = `https://www.google.com/recaptcha/api.js?render=${recaptchaSiteKey}`;
    } else {
      script.src = 'https://www.google.com/recaptcha/api.js?render=explicit';
    }
    script.async = true;
    script.defer = true;
    document.head.appendChild(script);
  }, [recaptchaEnabled, recaptchaSiteKey, recaptchaVersion]);

  useEffect(() => {
    if (recaptchaEnabled && recaptchaSiteKey && recaptchaVersion === 'v2' && (window as any).grecaptcha && (window as any).grecaptcha.render) {
      try {
        (window as any).grecaptcha.render('#recaptcha-widget', {
          sitekey: recaptchaSiteKey,
          callback: (token: string) => setRecaptchaToken(token),
        });
      } catch (e) { /* already rendered */ }
    }
  }, [recaptchaEnabled, recaptchaSiteKey, recaptchaVersion]);

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

  const handlePasskeyLogin = async () => {
    setPasskeyLoading(true);
    setError('');
    try {
      const beginRes = await fetch('/api/user/passkey/login/begin', { method: 'POST' });
      const beginData = await beginRes.json();
      if (beginData.code !== 0) throw new Error(beginData.msg || 'Passkey 启动失败');
      const { session_id, options } = beginData.data;
      const pk = options.publicKey;
      pk.challenge = b64urlToBuf(pk.challenge);
      pk.allowCredentials = [];
      const cred = await navigator.credentials.get({ publicKey: pk });
      if (!cred) throw new Error('Passkey 验证已取消');
      const credential = cred as any;
      const finishBody = {
        session_id,
        id: credential.id,
        rawId: bufToB64url(credential.rawId),
        response: {
          clientDataJSON: bufToB64url(credential.response.clientDataJSON),
          authenticatorData: bufToB64url(credential.response.authenticatorData),
          signature: bufToB64url(credential.response.signature),
          userHandle: credential.response.userHandle ? bufToB64url(credential.response.userHandle) : null,
        },
        type: credential.type,
      };
      const finishRes = await fetch('/api/user/passkey/login/finish', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(finishBody),
      });
      const finishData = await finishRes.json();
      if (finishData.code !== 0) throw new Error(finishData.msg || 'Passkey 登录失败');
      login(finishData.data.user, finishData.data.token);
      if (finishData.data.user.role >= 10) navigate('/admin');
      else navigate('/');
    } catch (err: any) {
      setError(err.message || 'Passkey 登录失败');
    } finally {
      setPasskeyLoading(false);
    }
  };

  const b64urlToBuf = (b64: string) => Uint8Array.from(atob(b64.replace(/-/g, '+').replace(/_/g, '/')), c => c.charCodeAt(0));
  const bufToB64url = (buf: ArrayBuffer) => btoa(String.fromCharCode(...new Uint8Array(buf))).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const useGeetest = captchaProvider === 'geetest' && geetestEnabled;
    const useTurnstile = captchaProvider === 'turnstile' ? turnstileEnabled : (!captchaProvider && turnstileEnabled);
    const useHCaptcha = captchaProvider === 'hcaptcha' && hcaptchaEnabled;
    const useReCaptcha = captchaProvider === 'recaptcha' && recaptchaEnabled;

    if (useTurnstile && !turnstileToken) {
      setError("请完成人机验证");
      return;
    }
    if (useHCaptcha && !hcaptchaToken) {
      setError("请完成人机验证");
      return;
    }
    let recaptchaTokenToSend = recaptchaToken;
    if (useReCaptcha) {
      if (recaptchaVersion === 'v3') {
        try {
          recaptchaTokenToSend = await (window as any).grecaptcha.execute(recaptchaSiteKey, { action: 'login' });
        } catch {
          setError('人机验证失败，请重试');
          return;
        }
      } else {
        if (!recaptchaTokenToSend) {
          setError("请完成人机验证");
          return;
        }
      }
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
      if (useHCaptcha) body.hcaptcha = hcaptchaToken;
      if (useReCaptcha) body.recaptcha = recaptchaTokenToSend;

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
        if (useHCaptcha && (window as any).hcaptcha) {
          (window as any).hcaptcha.reset();
          setHcaptchaToken('');
        }
        if (useReCaptcha && recaptchaVersion === 'v2' && (window as any).grecaptcha) {
          (window as any).grecaptcha.reset();
          setRecaptchaToken('');
        }
        setGeetestResult(null);
        throw new Error(data.msg || 'Login failed');
      }

      // Check if 2FA is required
      if (data.data?.requires_2fa) {
        setRequires2FA(true);
        setTotpUserId(data.data.user_id);
        setPreAuthTicket(data.data.pre_auth_ticket);
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
        body: JSON.stringify({ user_id: totpUserId, code: totpCode, pre_auth_ticket: preAuthTicket }),
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
                <Button variant="light" onPress={() => { setRequires2FA(false); setTotpCode(''); setPreAuthTicket(null); setError(''); }} style={{ color: 'var(--text-secondary)' }}>
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
            {notice && (
              <Alert color="primary" className="mt-3 w-full text-left whitespace-pre-wrap">
                {notice}
              </Alert>
            )}
          </CardHeader>

          <CardBody className="overflow-visible py-6 px-8">
            {error && <Alert color="danger" className="mb-4">{error}</Alert>}
            {passwordLoginEnabled && (
              <Form className="flex flex-col gap-4" onSubmit={handleSubmit}>
                <Input isRequired label="用户名" placeholder="请输入用户名" value={username} onValueChange={setUsername} />
                <Input isRequired label="密码" type="password" placeholder="请输入密码" value={password} onValueChange={setPassword} />

                {turnstileEnabled && (
                  <div id="turnstile-widget" className="cf-turnstile" data-sitekey={turnstileSiteKey} data-callback="onTurnstileSuccess" />
                )}

                {hcaptchaEnabled && captchaProvider === 'hcaptcha' && (
                  <div id="hcaptcha-widget" />
                )}

                {recaptchaEnabled && recaptchaVersion === 'v2' && captchaProvider === 'recaptcha' && (
                  <div id="recaptcha-widget" className="g-recaptcha" />
                )}
                {recaptchaEnabled && recaptchaVersion === 'v3' && captchaProvider === 'recaptcha' && (
                  <p className="text-xs" style={{ color: 'var(--text-muted)' }}>受 reCAPTCHA v3 保护</p>
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
            )}

            {passwordLoginEnabled && (
              <div className="mt-2 text-right">
                <Button variant="light" size="sm" onPress={onResetOpen} style={{ color: 'var(--text-secondary)' }}>
                  忘记密码?
                </Button>
              </div>
            )}

            {(passkeyEnabled || githubEnabled || linuxDOEnabled || discordEnabled || oidcEnabled || telegramEnabled || wechatEnabled) && (
              <div className="relative my-4">
                <div className="absolute inset-0 flex items-center">
                  <div className="w-full" style={{ borderTop: '1px solid var(--border-color)' }} />
                </div>
                <div className="relative flex justify-center text-xs">
                  <span className="px-3" style={{ background: 'var(--bg-surface)', color: 'var(--text-secondary)' }}>{passwordLoginEnabled ? '或使用其他方式' : '登录方式'}</span>
                </div>
              </div>
            )}

            {passkeyEnabled && (
              <div className="mt-2">
                <Button
                  variant="bordered"
                  isLoading={passkeyLoading}
                  className="w-full"
                  onPress={handlePasskeyLogin}
                  style={{ borderColor: 'var(--accent-primary)', color: 'var(--accent-primary)', borderRadius: '12px' }}
                >
                  🔑 Passkey 登录
                </Button>
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
            {wechatEnabled && (
              <div className="mt-2">
                <Button variant="bordered" className="w-full" onPress={() => { window.location.href = '/oauth/wechat'; }}
                  style={{ borderColor: 'var(--border-color)', color: 'var(--text-primary)', borderRadius: '12px' }}>
                  微信扫码登录
                </Button>
              </div>
            )}

            {registerEnabled && (
              <div className="mt-5 text-center">
                <Button variant="light" onPress={() => navigate(initialized ? '/register' : '/setup')} style={{ color: 'var(--accent-primary)' }}>
                  {initialized ? '没有账号？立即注册 ✦' : '初始化系统（配置数据库 + 创建管理员）'}
                </Button>
              </div>
            )}
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
