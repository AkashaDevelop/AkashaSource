import { useState, useEffect } from 'react';
import {
  Card,
  CardHeader,
  CardBody,
  Button,
  Table,
  TableHeader,
  TableColumn,
  TableBody,
  TableRow,
  TableCell,
  Chip,
  Snippet,
  Alert,
} from '@heroui/react';
import { Plus } from 'lucide-react';
import { useAuthStore } from '../../store/auth';

interface Invitation {
  id: number;
  code: string;
  status: number; // 1: Unused, 2: Used
  cost: number;
  created_at: number;
  used_at: number;
  invitee_id: number;
}

export default function UserInvitation() {
  const [invitations, setInvitations] = useState<Invitation[]>([]);
  const [loading, setLoading] = useState(false);
  const [generating, setGenerating] = useState(false);
  const [error, setError] = useState('');
  const { token } = useAuthStore();

  const fetchInvitations = async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/user/invitation', {
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (res.ok) {
        setInvitations(data.data || []);
      } else {
        setError(data.error || 'Failed to fetch invitations');
      }
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchInvitations();
  }, []);

  const handleGenerate = async () => {
    setGenerating(true);
    setError('');
    try {
      const res = await fetch('/api/user/invitation', {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (res.ok) {
        fetchInvitations();
      } else {
        setError(data.error || 'Failed to generate invitation code');
      }
    } catch (err: any) {
      setError(err.message);
    } finally {
      setGenerating(false);
    }
  };

  return (
    <div className="p-6">
      <Card>
        <CardHeader className="flex justify-between items-center">
          <div className="flex flex-col">
            <h1 className="text-xl font-bold">Invitation Management</h1>
            <p className="text-small text-default-500">Manage your invitation codes</p>
          </div>
          <Button
            color="primary"
            endContent={<Plus size={16} />}
            isLoading={generating}
            onPress={handleGenerate}
          >
            Generate Code
          </Button>
        </CardHeader>
        <CardBody>
          {error && (
            <Alert color="danger" className="mb-4" onClose={() => setError('')}>
              {error}
            </Alert>
          )}

          <Table aria-label="Invitation codes table">
            <TableHeader>
              <TableColumn>CODE</TableColumn>
              <TableColumn>STATUS</TableColumn>
              <TableColumn>COST</TableColumn>
              <TableColumn>CREATED AT</TableColumn>
              <TableColumn>USED AT</TableColumn>
            </TableHeader>
            <TableBody emptyContent="No invitation codes found" isLoading={loading}>
              {invitations.map((item) => (
                <TableRow key={item.id}>
                  <TableCell>
                    <Snippet symbol="" variant="flat" size="sm">
                      {item.code}
                    </Snippet>
                  </TableCell>
                  <TableCell>
                    <Chip
                      color={item.status === 1 ? 'success' : 'default'}
                      size="sm"
                      variant="flat"
                    >
                      {item.status === 1 ? 'Unused' : 'Used'}
                    </Chip>
                  </TableCell>
                  <TableCell>{item.cost}</TableCell>
                  <TableCell>
                    {new Date(item.created_at * 1000).toLocaleString()}
                  </TableCell>
                  <TableCell>
                    {item.used_at
                      ? new Date(item.used_at * 1000).toLocaleString()
                      : '-'}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardBody>
      </Card>
    </div>
  );
}
