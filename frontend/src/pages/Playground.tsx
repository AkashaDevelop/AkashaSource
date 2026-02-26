import { useState, useEffect, useRef } from 'react';
import {
  Card,
  CardBody,
  CardHeader,
  Button,
  Input,
  Select,
  SelectItem,
  Textarea,
  Switch,
  Divider,
} from '@heroui/react';
import { Play, Square, Trash2 } from 'lucide-react';
import { useAuthStore } from '../store/auth';

interface ModelItem {
  id: string;
  object: string;
}

export default function Playground() {
  const { token } = useAuthStore();
  const [models, setModels] = useState<ModelItem[]>([]);
  const [selectedModel, setSelectedModel] = useState('');
  const [systemPrompt, setSystemPrompt] = useState('');
  const [userMessage, setUserMessage] = useState('');
  const [response, setResponse] = useState('');
  const [loading, setLoading] = useState(false);
  const [stream, setStream] = useState(true);
  const [temperature, setTemperature] = useState('0.7');
  const [maxTokens, setMaxTokens] = useState('2048');
  const abortRef = useRef<AbortController | null>(null);

  useEffect(() => {
    fetchModels();
  }, []);

  const fetchModels = async () => {
    try {
      const res = await fetch('/v1/models', {
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (res.ok && data.data) {
        setModels(data.data);
        if (data.data.length > 0) setSelectedModel(data.data[0].id);
      }
    } catch (e) {
      console.error('Failed to fetch models:', e);
    }
  };

  const handleStop = () => {
    abortRef.current?.abort();
    setLoading(false);
  };

  const handleClear = () => {
    setResponse('');
    setUserMessage('');
  };

  const handleSend = async () => {
    if (!userMessage.trim() || !selectedModel) return;
    setLoading(true);
    setResponse('');

    const messages: any[] = [];
    if (systemPrompt.trim()) {
      messages.push({ role: 'system', content: systemPrompt });
    }
    messages.push({ role: 'user', content: userMessage });

    const body = {
      model: selectedModel,
      messages,
      stream,
      temperature: parseFloat(temperature) || 0.7,
      max_tokens: parseInt(maxTokens) || 2048,
    };

    const controller = new AbortController();
    abortRef.current = controller;

    try {
      const res = await fetch('/v1/chat/completions', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify(body),
        signal: controller.signal,
      });

      if (!res.ok) {
        const err = await res.json();
        setResponse(`Error: ${err.error?.message || res.statusText}`);
        setLoading(false);
        return;
      }

      if (stream) {
        await handleStreamResponse(res);
      } else {
        const data = await res.json();
        setResponse(data.choices?.[0]?.message?.content || 'No response');
      }
    } catch (e: any) {
      if (e.name !== 'AbortError') {
        setResponse(`Error: ${e.message}`);
      }
    } finally {
      setLoading(false);
    }
  };

  const handleStreamResponse = async (res: Response) => {
    const reader = res.body?.getReader();
    if (!reader) return;
    const decoder = new TextDecoder();
    let buffer = '';

    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      buffer += decoder.decode(value, { stream: true });

      const lines = buffer.split('\n');
      buffer = lines.pop() || '';

      for (const line of lines) {
        if (!line.startsWith('data: ')) continue;
        const data = line.slice(6);
        if (data === '[DONE]') return;
        try {
          const chunk = JSON.parse(data);
          const content = chunk.choices?.[0]?.delta?.content;
          if (content) {
            setResponse(prev => prev + content);
          }
        } catch { /* skip */ }
      }
    }
  };

  return (
    <div className="p-6 space-y-6 max-w-[1400px] mx-auto">
      <div>
        <h1 className="text-2xl font-bold">Playground</h1>
        <p className="text-default-500">在线测试 API 接口</p>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Left: Config */}
        <Card>
          <CardHeader><h3 className="font-semibold">请求配置</h3></CardHeader>
          <Divider />
          <CardBody className="gap-4">
            <Select
              label="模型"
              selectedKeys={selectedModel ? [selectedModel] : []}
              onChange={(e) => setSelectedModel(e.target.value)}
            >
              {models.map((m) => (
                <SelectItem key={m.id}>{m.id}</SelectItem>
              ))}
            </Select>
            <Input
              label="Temperature"
              type="number"
              value={temperature}
              onValueChange={setTemperature}
              step={0.1}
              min={0}
              max={2}
            />
            <Input
              label="Max Tokens"
              type="number"
              value={maxTokens}
              onValueChange={setMaxTokens}
            />
            <div className="flex items-center justify-between">
              <span className="text-sm">流式输出</span>
              <Switch isSelected={stream} onValueChange={setStream} size="sm" />
            </div>
            <Textarea
              label="System Prompt"
              placeholder="可选的系统提示词..."
              value={systemPrompt}
              onValueChange={setSystemPrompt}
              minRows={3}
            />
          </CardBody>
        </Card>

        {/* Right: Chat */}
        <div className="lg:col-span-2 space-y-4">
          <Card>
            <CardHeader><h3 className="font-semibold">输入</h3></CardHeader>
            <Divider />
            <CardBody>
              <Textarea
                placeholder="输入你的消息..."
                value={userMessage}
                onValueChange={setUserMessage}
                minRows={4}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) handleSend();
                }}
              />
              <div className="flex gap-2 mt-3">
                {loading ? (
                  <Button color="danger" startContent={<Square size={16} />} onPress={handleStop}>
                    停止
                  </Button>
                ) : (
                  <Button color="primary" startContent={<Play size={16} />} onPress={handleSend} isDisabled={!userMessage.trim()}>
                    发送 (Ctrl+Enter)
                  </Button>
                )}
                <Button variant="flat" startContent={<Trash2 size={16} />} onPress={handleClear}>
                  清空
                </Button>
              </div>
            </CardBody>
          </Card>

          <Card>
            <CardHeader><h3 className="font-semibold">响应</h3></CardHeader>
            <Divider />
            <CardBody>
              <div className="min-h-[200px] max-h-[500px] overflow-auto p-3 bg-default-100 rounded-lg font-mono text-sm whitespace-pre-wrap">
                {response || (loading ? '等待响应...' : '响应将显示在这里')}
              </div>
            </CardBody>
          </Card>
        </div>
      </div>
    </div>
  );
}
