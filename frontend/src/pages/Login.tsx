import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Card, CardBody, CardHeader, Button, Input, Form, Alert } from '@heroui/react';
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
  const [turnstileEnabled, setTurnstileEnabled] = useState(false);
  const [turnstileSiteKey, setTurnstileSiteKey] = useState('');
  const [turnstileToken, setTurnstileToken] = useState('');

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
          if (data.options.turnstile_check_enabled === 'true') {
            setTurnstileEnabled(true);
            setTurnstileSiteKey(data.options.turnstile_site_key || '');
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

  const handleGithubLogin = () => {
    window.location.href = '/oauth/github';
  };

  const handleLinuxDOLogin = () => {
    window.location.href = '/oauth/linuxdo';
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (turnstileEnabled && !turnstileToken) {
      setError("Please complete the security check");
      return;
    }

    setLoading(true);
    setError('');

    try {
      const res = await fetch('/api/user/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ 
          username, 
          password,
          turnstile: turnstileToken 
        }),
      });

      const data = await res.json();
      if (!res.ok) {
        if (turnstileEnabled) {
          (window as any).turnstile.reset();
          setTurnstileToken('');
        }
        throw new Error(data.error || 'Login failed');
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

          {githubEnabled && (
            <div className="mt-4">
              <Button 
                variant="bordered" 
                className="w-full"
                onPress={handleGithubLogin}
              >
                Sign in with GitHub
              </Button>
            </div>
          )}

          {linuxDOEnabled && (
            <div className="mt-4">
              <Button 
                variant="bordered" 
                className="w-full"
                onPress={handleLinuxDOLogin}
              >
                Sign in with LinuxDO
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
    </div>
  );
}
