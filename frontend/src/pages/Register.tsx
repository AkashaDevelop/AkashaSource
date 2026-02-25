import { useState } from 'react';
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

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError('');

    try {
      const res = await fetch('/api/user/register', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ username, password, email, invitation_code: invitationCode }),
      });

      const data = await res.json();

      if (!res.ok) {
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
