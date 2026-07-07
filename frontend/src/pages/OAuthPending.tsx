import { useState, useEffect } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { Card, CardBody, CardHeader, Button, Input, Alert } from '../components/ui';
import { useAuthStore } from '../store/auth';

export default function OAuthPending() {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const { login } = useAuthStore();
  const sessionId = searchParams.get('oauth_pending') || '';
  const [invitationCode, setInvitationCode] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [valid, setValid] = useState(false);

  useEffect(() => {
    if (!sessionId) {
      navigate('/login');
      return;
    }
    fetch(`/api/oauth/pending?session_id=${sessionId}`)
      .then(res => res.json())
      .then(data => {
        if (data.code !== 0) {
          setError('OAuth 会话不存在或已过期，请重新登录');
        } else {
          setValid(true);
        }
      });
  }, [sessionId]);

  const handleSubmit = async () => {
    if (!invitationCode) return;
    setLoading(true);
    setError('');
    try {
      const res = await fetch('/api/oauth/complete', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ session_id: sessionId, invitation_code: invitationCode }),
      });
      const data = await res.json();
      if (data.code !== 0) throw new Error(data.msg || '注册失败');
      login(data.data.user, data.data.token);
      if (data.data.user.role >= 10) navigate('/admin');
      else navigate('/');
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
      <div className="animate-fade-in-up w-full max-w-md">
        <Card style={{ background: 'var(--bg-surface)', backdropFilter: 'blur(16px)', border: '1px solid var(--border-color)', borderRadius: '24px', boxShadow: 'var(--shadow-card)' }}>
          <CardHeader className="flex flex-col gap-1 items-center pb-0 pt-8">
            <div className="text-3xl mb-2">🎟️</div>
            <h1 className="text-2xl font-bold gradient-text">输入邀请码</h1>
            <p className="text-sm mt-1" style={{ color: 'var(--text-secondary)' }}>请输入邀请码完成注册</p>
          </CardHeader>
          <CardBody className="overflow-visible py-6 px-8">
            {error && <Alert color="danger" className="mb-4">{error}</Alert>}
            {valid ? (
              <>
                <Input
                  isRequired
                  label="邀请码"
                  placeholder="请输入邀请码"
                  value={invitationCode}
                  onValueChange={setInvitationCode}
                  onKeyDown={(e) => { if (e.key === 'Enter') handleSubmit(); }}
                />
                <Button
                  isLoading={loading}
                  onPress={handleSubmit}
                  className="w-full font-bold h-11 mt-4"
                  style={{ background: 'linear-gradient(135deg, var(--accent-primary), var(--accent-cosmic))', color: 'white', borderRadius: '12px' }}
                >
                  完成注册
                </Button>
              </>
            ) : (
              <Button variant="light" onPress={() => navigate('/login')} className="w-full" style={{ color: 'var(--text-secondary)' }}>
                返回登录
              </Button>
            )}
          </CardBody>
        </Card>
      </div>
    </div>
  );
}
