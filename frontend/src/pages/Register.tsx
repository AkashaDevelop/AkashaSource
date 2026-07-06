import { useState, useEffect } from 'react';
import { Button, Card, CardBody, CardHeader, Input, Link, Form, Alert } from '../components/ui';
import { useNavigate } from 'react-router-dom';

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export default function Register() {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [email, setEmail] = useState('');
  const [invitationCode, setInvitationCode] = useState('');
  const [error, setError] = useState('');
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();

  const validateField = (name: string, value: string) => {
    let msg = '';
    if (name === 'username' && !value.trim()) msg = '用户名不能为空';
    if (name === 'email' && value && !EMAIL_RE.test(value)) msg = '邮箱格式不正确';
    if (name === 'password' && value.length > 0 && value.length < 6) msg = '密码至少 6 位';
    setFieldErrors(prev => ({ ...prev, [name]: msg }));
    return msg;
  };

  const validateAll = () => {
    const errs: Record<string, string> = {};
    if (!username.trim()) errs.username = '用户名不能为空';
    if (email && !EMAIL_RE.test(email)) errs.email = '邮箱格式不正确';
    if (password.length < 6) errs.password = '密码至少 6 位';
    setFieldErrors(errs);
    return Object.keys(errs).length === 0;
  };

  const [captchaProvider, setCaptchaProvider] = useState('');
  const [turnstileEnabled, setTurnstileEnabled] = useState(false);
  const [turnstileSiteKey, setTurnstileSiteKey] = useState('');
  const [turnstileToken, setTurnstileToken] = useState('');
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
  const [emailVerifyEnabled, setEmailVerifyEnabled] = useState(false);
  const [emailCode, setEmailCode] = useState('');
  const [sendingCode, setSendingCode] = useState(false);
  const [codeSent, setCodeSent] = useState(false);
  const [countdown, setCountdown] = useState(0);
  const [registerDisabled, setRegisterDisabled] = useState(false);

  useEffect(() => {
    fetch('/api/system/status')
      .then(res => res.json())
      .then(data => {
        const payload = data.code === 0 ? data.data : data;
        if (payload.options) {
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
          if (payload.options.email_verification_enabled === 'true') {
            setEmailVerifyEnabled(true);
          }
          if (payload.options.register_enabled === 'false') {
            setRegisterDisabled(true);
          }
        }
      });
  }, []);

  useEffect(() => {
    if (turnstileEnabled && turnstileSiteKey && (window as any).turnstile) {
      (window as any).turnstile.render('#register-turnstile-widget', {
        sitekey: turnstileSiteKey,
        callback: (token: string) => setTurnstileToken(token),
      });
    }
  }, [turnstileEnabled, turnstileSiteKey]);

  useEffect(() => {
    if (geetestEnabled && geetestId && (window as any).initGeetest4) {
      (window as any).initGeetest4({
        captchaId: geetestId,
        product: 'bind',
      }, (captchaObj: any) => {
        (window as any)._geetestRegCaptcha = captchaObj;
        captchaObj.onSuccess(() => {
          setGeetestResult(captchaObj.getValidate());
        });
      });
    }
  }, [geetestEnabled, geetestId]);

  useEffect(() => {
    if (hcaptchaEnabled && hcaptchaSiteKey && (window as any).hcaptcha) {
      try {
        (window as any).hcaptcha.render('#register-hcaptcha-widget', {
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
        (window as any).grecaptcha.render('#register-recaptcha-widget', {
          sitekey: recaptchaSiteKey,
          callback: (token: string) => setRecaptchaToken(token),
        });
      } catch (e) { /* already rendered */ }
    }
  }, [recaptchaEnabled, recaptchaSiteKey, recaptchaVersion]);

  const sendCode = async () => {
    if (!email) { setError('请先填写邮箱'); return; }
    setSendingCode(true);
    try {
      const res = await fetch('/api/user/email/verify-code', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email }),
      });
      const data = await res.json();
      if (data.code === 0) {
        setCodeSent(true);
        setCountdown(60);
        const timer = setInterval(() => setCountdown(v => { if (v <= 1) { clearInterval(timer); return 0; } return v - 1; }), 1000);
      } else {
        setError(data.msg || '发送失败');
      }
    } catch { setError('发送失败'); }
    finally { setSendingCode(false); }
  };

  const triggerGeetest = (): Promise<any> => {
    return new Promise((resolve) => {
      const captchaObj = (window as any)._geetestRegCaptcha;
      if (!captchaObj) { resolve(null); return; }
      captchaObj.onSuccess(() => {
        resolve(captchaObj.getValidate());
      });
      captchaObj.showCaptcha();
    });
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!validateAll()) return;
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
          recaptchaTokenToSend = await (window as any).grecaptcha.execute(recaptchaSiteKey, { action: 'register' });
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
      const body: any = { username, password, email, invitation_code: invitationCode };
      if (emailVerifyEnabled) body.email_code = emailCode;
      if (useTurnstile) body.turnstile = turnstileToken;
      if (useGeetest && geetestData) body.geetest = geetestData;
      if (useHCaptcha) body.hcaptcha = hcaptchaToken;
      if (useReCaptcha) body.recaptcha = recaptchaTokenToSend;

      const res = await fetch('/api/user/register', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      });

      const data = await res.json();

      if (data.code !== 0) {
        if (useTurnstile && (window as any).turnstile) {
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
        throw new Error(data.msg || 'Registration failed');
      }

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

      <div className="animate-fade-in-up w-full max-w-md">
        <Card className="w-full" style={{
          background: 'var(--bg-surface)',
          backdropFilter: 'blur(16px)',
          border: '1px solid var(--border-color)',
          borderRadius: '24px',
          boxShadow: 'var(--shadow-card)',
        }}>
          <CardHeader className="flex flex-col gap-1 items-center pb-0 pt-8">
            <div className="text-4xl mb-2">🌸</div>
            <h1 className="text-2xl font-bold gradient-text">创建账号</h1>
            <p className="text-xs mt-1" style={{ color: 'var(--text-secondary)' }}>注册新账号，开始使用</p>
          </CardHeader>
          <CardBody className="overflow-visible py-6 px-8">
            {error && <Alert color="danger" className="mb-4">{error}</Alert>}
            {registerDisabled ? (
              <div className="text-center py-8">
                <p className="text-base" style={{ color: 'var(--text-secondary)' }}>注册已关闭，请联系管理员</p>
              </div>
            ) : (
            <Form className="flex flex-col gap-4" onSubmit={handleSubmit}>
              <div>
                <Input
                  isRequired
                  label="用户名"
                  placeholder="请输入用户名"
                  value={username}
                  onValueChange={v => { setUsername(v); if (fieldErrors.username) validateField('username', v); }}
                  onBlur={() => validateField('username', username)}
                  isInvalid={!!fieldErrors.username}
                />
                {fieldErrors.username && <span className="field-error">{fieldErrors.username}</span>}
              </div>
              <div>
                <Input
                  isRequired
                  label="邮箱"
                  placeholder="请输入邮箱"
                  type="email"
                  value={email}
                  onValueChange={v => { setEmail(v); if (fieldErrors.email) validateField('email', v); }}
                  onBlur={() => validateField('email', email)}
                  isInvalid={!!fieldErrors.email}
                />
                {fieldErrors.email && <span className="field-error">{fieldErrors.email}</span>}
              </div>
              {emailVerifyEnabled && (
                <div style={{ display: 'flex', gap: '8px' }}>
                  <Input label="邮箱验证码" placeholder="6位验证码" value={emailCode} onValueChange={setEmailCode} style={{ flex: 1 }} />
                  <Button size="sm" variant="flat" isLoading={sendingCode} isDisabled={countdown > 0}
                    onPress={sendCode} style={{ alignSelf: 'flex-end', height: '40px', minWidth: '90px', borderRadius: '10px' }}>
                    {countdown > 0 ? `${countdown}s` : codeSent ? '重新发送' : '发送验证码'}
                  </Button>
                </div>
              )}
              <Input isRequired label="密码" placeholder="请设置密码（至少 6 位）" type="password" value={password}
                onValueChange={v => { setPassword(v); if (fieldErrors.password) validateField('password', v); }}
                onBlur={() => validateField('password', password)}
                isInvalid={!!fieldErrors.password}
              />
              {fieldErrors.password && <span className="field-error">{fieldErrors.password}</span>}
              <Input label="邀请码" placeholder="邀请码（选填）" value={invitationCode} onValueChange={setInvitationCode} />

              {turnstileEnabled && captchaProvider !== 'geetest' && (
                <div id="register-turnstile-widget" className="cf-turnstile" data-sitekey={turnstileSiteKey} />
              )}

              {hcaptchaEnabled && captchaProvider === 'hcaptcha' && (
                <div id="register-hcaptcha-widget" />
              )}

              {recaptchaEnabled && recaptchaVersion === 'v2' && captchaProvider === 'recaptcha' && (
                <div id="register-recaptcha-widget" className="g-recaptcha" />
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
                注册
              </Button>
            </Form>
            )}
            <div className="mt-5 text-center">
              <span className="text-sm" style={{ color: 'var(--text-secondary)' }}>已有账号？</span>
              <Link href="/login" style={{ color: 'var(--accent-primary)' }}> 立即登录 ✦</Link>
            </div>
          </CardBody>
        </Card>
      </div>
    </div>
  );
}
