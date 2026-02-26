import { useState, useEffect } from 'react';
import {
  Table, TableHeader, TableColumn, TableBody, TableRow, TableCell,
  Card, CardBody, Input, Chip,
} from '@heroui/react';
import { Search } from 'lucide-react';

interface ModelPrice {
  model: string;
  input_ratio: number;
  output_ratio: number;
}

export default function Pricing() {
  const [models, setModels] = useState<ModelPrice[]>([]);
  const [loading, setLoading] = useState(false);
  const [search, setSearch] = useState('');

  useEffect(() => {
    setLoading(true);
    fetch('/api/pricing')
      .then(res => res.json())
      .then(data => {
        if (data.models) setModels(data.models);
      })
      .catch(console.error)
      .finally(() => setLoading(false));
  }, []);

  const filtered = models.filter(m =>
    m.model.toLowerCase().includes(search.toLowerCase())
  );

  return (
    <div className="max-w-4xl mx-auto p-6 space-y-6">
      <div className="text-center space-y-2">
        <h1 className="text-3xl font-bold">模型定价</h1>
        <p className="text-default-500">所有可用模型的倍率一览</p>
      </div>

      <Card>
        <CardBody>
          <Input
            placeholder="搜索模型..."
            value={search}
            onValueChange={setSearch}
            startContent={<Search size={18} className="text-default-400" />}
            className="mb-4"
          />
          <Table aria-label="Pricing table" removeWrapper>
            <TableHeader>
              <TableColumn>模型</TableColumn>
              <TableColumn>输入倍率</TableColumn>
              <TableColumn>输出倍率</TableColumn>
            </TableHeader>
            <TableBody emptyContent="暂无定价数据" isLoading={loading}>
              {filtered.map((m) => (
                <TableRow key={m.model}>
                  <TableCell>
                    <Chip size="sm" variant="flat">{m.model}</Chip>
                  </TableCell>
                  <TableCell>{m.input_ratio}</TableCell>
                  <TableCell>{m.output_ratio}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardBody>
      </Card>
    </div>
  );
}
