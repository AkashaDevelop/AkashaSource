import { useState, useEffect } from 'react';
import {
  Button,
  Card,
  CardBody,
  CardHeader,
  Input,
  Link,
  Form,
  Alert,
} from '@heroui/react';
import { useNavigate } from 'react-router-dom';

export default function Register() {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [email, setEmail] = useState('');
  const [invitationCode, setInvitationCode] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();

  const [captchaProvider, setCaptchaProvider] = useState('');
  const [turnstileEnabled, setTurnstileEnabled] = useState(false);
  const [turnstileSiteKey, setTurnstileSiteKey] = useState('');
  const [turnstileToken, setTurnstileToken] = useState('');
  const [geetestEnabled, setGeetestEnabled] = useState(false);
  const [geetestId, setGeetestId] = useState('');
  const [geetestResult, setGeetestResult] = useState<any>(null);

  useEffect(() => {
    fetch('/api/system/status')
      .then(res => res.json())
      .then(data => {
        if (data.options) {
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
      const body: any = { username, password, email, invitation_code: invitationCode };
      if (useTurnstile) body.turnstile = turnstileToken;
      if (useGeetest && geetestData) body.geetest = geetestData;

      const res = await fetch('/api/user/register', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      });

      const data = await res.json();

      if (!res.ok) {
        if (useTurnstile && (window as any).turnstile) {
          (window as any).turnstile.reset();
          setTurnstileToken('');
        }
        setGeetestResult(null);
        throw new Error(data.error || 'Registration failed');
      }

      navigate('/login');
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex items-center justify-center min-h-screen bg-gray-50 dark:bg-gray-900 p-4">
      <Card className="w-full max-w-md">
        <CardHeader className="flex flex-col gap-1 items-center pb-0">
          <h1 className="text-2xl font-bold">Create Account</h1>
          <p className="text-small text-default-500">Sign up for a new account</p>
        </CardHeader>
        <CardBody className="overflow-visible py-4">
          {error && (
            <Alert color="danger" className="mb-4">
              {error}
            </Alert>
          )}
          <Form className="flex flex-col gap-4" onSubmit={handleSubmit}>
            <Input
              isRequired
              label="Username"
              placeholder="Choose a username"
              value={username}
              onValueChange={setUsername}
            />
            <Input
              isRequired
              label="Email"
              placeholder="Enter your email"
              type="email"
              value={email}
              onValueChange={setEmail}
            />
            <Input
              isRequired
              label="Password"
              placeholder="Choose a password"
              type="password"
              value={password}
              onValueChange={setPassword}
            />
            <Input
              label="Invitation Code"
              placeholder="Enter invitation code (optional)"
              value={invitationCode}
              onValueChange={setInvitationCode}
            />

            {turnstileEnabled && captchaProvider !== 'geetest' && (
              <div id="register-turnstile-widget" className="cf-turnstile" data-sitekey={turnstileSiteKey}></div>
            )}

            <Button
              color="primary"
              type="submit"
              isLoading={loading}
              className="w-full font-bold"
            >
              Sign Up
            </Button>
          </Form>
          <div className="mt-4 text-center text-small">
            Already have an account?{' '}
            <Link href="/login" size="sm">
              Log in
            </Link>
          </div>
        </CardBody>
      </Card>
    </div>
  );
}
