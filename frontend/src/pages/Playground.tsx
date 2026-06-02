import { useState, useEffect, useRef } from 'react';
import PageHeader from '../components/PageHeader';
import {
  Card, CardBody, Button, Input, Select, SelectItem, Textarea, Switch, Divider, Chip, Tooltip,
} from '../components/ui';
import { Play, Square, Trash2, Copy, Check, Settings2 } from 'lucide-react';
import { useAuthStore } from '../store/auth';
import { toast } from '../store/toast';

interface ModelItem { id: string; }
interface Usage { prompt_tokens: number; completion_tokens: number; total_tokens: number; }

export default function Playground() {
  const { token } = useAuthStore();
  const [models, setModels] = useState<ModelItem[]>([]);
  const [selectedModel, setSelectedModel] = useState('');
  const [systemPrompt, setSystemPrompt] = useState('');
  const [userMessage, setUserMessage] = useState('');
  const [response, setResponse] = useState('');
  const [usage, setUsage] = useState<Usage | null>(null);
  const [loading, setLoading] = useState(false);
  const [stream, setStream] = useState(true);
  const [temperature, setTemperature] = useState('0.7');
  const [maxTokens, setMaxTokens] = useState('2048');
  const [copied, setCopied] = useState(false);
  const abortRef = useRef<AbortController | null>(null);
  const responseRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    fetch('/v1/models', { headers: { Authorization: `Bearer ${token}` } })
      .then(r => r.json())
      .then(d => {
        if (d.data?.length) {
          setModels(d.data);
          setSelectedModel(d.data[0].id);
        }
      })
      .catch(console.error);
  }, []);

  // Auto-scroll response
  useEffect(() => {
    if (responseRef.current) {
      responseRef.current.scrollTop = responseRef.current.scrollHeight;
    }
  }, [response]);

  const handleStop = () => { abortRef.current?.abort(); setLoading(false); };

  const handleClear = () => { setResponse(''); setUserMessage(''); setUsage(null); };

  const handleCopy = async () => {
    if (!response) return;
    await navigator.clipboard.writeText(response);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const handleSend = async () => {
    if (!userMessage.trim() || !selectedModel) return;
    setLoading(true);
    setResponse('');
    setUsage(null);

    const messages: any[] = [];
    if (systemPrompt.trim()) messages.push({ role: 'system', content: systemPrompt });
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
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify(body),
        signal: controller.signal,
      });

      if (!res.ok) {
        const err = await res.json();
        setResponse(`错误：${err.error?.message || res.statusText}`);
        setLoading(false);
        return;
      }

      if (stream) {
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
            if (data === '[DONE]') continue;
            try {
              const chunk = JSON.parse(data);
              const content = chunk.choices?.[0]?.delta?.content;
              if (content) setResponse(prev => prev + content);
              if (chunk.usage) setUsage(chunk.usage);
            } catch { /* skip */ }
          }
        }
      } else {
        const data = await res.json();
        setResponse(data.choices?.[0]?.message?.content || '（无响应）');
        if (data.usage) setUsage(data.usage);
      }
    } catch (e: any) {
      if (e.name !== 'AbortError') {
        setResponse(`错误：${e.message}`);
        toast.error('请求失败');
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="space-y-5">
      <PageHeader title="API 调试台" description="在线测试模型接口，支持流式输出" />

      <div className="grid grid-cols-1 lg:grid-cols-4 gap-5">

        {/* 左侧配置 */}
        <Card className="lg:col-span-1 h-fit">
          <CardBody className="gap-4 p-5">
            <div className="flex items-center gap-2 mb-1">
              <Settings2 size={16} style={{ color: 'var(--accent-primary)' }} />
              <span style={{ fontSize: '14px', fontWeight: 600, color: 'var(--text-primary)' }}>请求配置</span>
            </div>
            <Divider />

            <Select
              label="模型"
              size="sm"
              selectedKeys={selectedModel ? [selectedModel] : []}
              onSelectionChange={keys => setSelectedModel([...keys][0] as string || '')}
            >
              {models.map(m => <SelectItem key={m.id}>{m.id}</SelectItem>)}
            </Select>

            <Input
              label="Temperature"
              size="sm"
              type="number"
              value={temperature}
              onValueChange={setTemperature}
              description="0 = 确定性，2 = 最随机"
            />

            <Input
              label="最大 Token 数"
              size="sm"
              type="number"
              value={maxTokens}
              onValueChange={setMaxTokens}
            />

            <div style={{
              display: 'flex', alignItems: 'center', justifyContent: 'space-between',
              padding: '10px 12px', borderRadius: '10px',
              background: 'var(--bg-elevated)', border: '1px solid var(--border-color)',
            }}>
              <span style={{ fontSize: '13px', color: 'var(--text-secondary)' }}>流式输出</span>
              <Switch isSelected={stream} onValueChange={setStream} size="sm" />
            </div>

            <Textarea
              label="系统提示词"
              placeholder="可选，设置模型角色..."
              value={systemPrompt}
              onValueChange={setSystemPrompt}
              minRows={3}
              maxRows={6}
            />
          </CardBody>
        </Card>

        {/* 右侧对话区 */}
        <div className="lg:col-span-3 space-y-4">

          {/* 输入区 */}
          <Card>
            <CardBody className="p-5">
              <Textarea
                placeholder="输入消息，按 Ctrl+Enter 发送..."
                value={userMessage}
                onValueChange={setUserMessage}
                minRows={4}
                maxRows={10}
                onKeyDown={e => { if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) handleSend(); }}
              />
              <div className="flex gap-2 mt-3 justify-between items-center">
                <div className="flex gap-2">
                  {loading ? (
                    <Button color="danger" size="sm" startContent={<Square size={14} />} onPress={handleStop}>
                      停止生成
                    </Button>
                  ) : (
                    <Button
                      color="primary"
                      size="sm"
                      startContent={<Play size={14} />}
                      onPress={handleSend}
                      isDisabled={!userMessage.trim() || !selectedModel}
                    >
                      发送
                    </Button>
                  )}
                  <Button size="sm" variant="flat" startContent={<Trash2 size={14} />} onPress={handleClear}>
                    清空
                  </Button>
                </div>
                <span style={{ fontSize: '12px', color: 'var(--text-faint)' }}>Ctrl+Enter 发送</span>
              </div>
            </CardBody>
          </Card>

          {/* 响应区 */}
          <Card>
            <CardBody className="p-5">
              <div className="flex items-center justify-between mb-3">
                <span style={{ fontSize: '14px', fontWeight: 600, color: 'var(--text-primary)' }}>响应</span>
                <div className="flex items-center gap-3">
                  {usage && (
                    <div className="flex gap-2">
                      <Chip size="sm" variant="flat" className="text-[11px]">
                        提示 {usage.prompt_tokens}
                      </Chip>
                      <Chip size="sm" variant="flat" color="success" className="text-[11px]">
                        补全 {usage.completion_tokens}
                      </Chip>
                      <Chip size="sm" variant="flat" color="primary" className="text-[11px]">
                        共 {usage.total_tokens}
                      </Chip>
                    </div>
                  )}
                  {response && (
                    <Tooltip content={copied ? '已复制' : '复制响应'}>
                      <button onClick={handleCopy} style={{
                        background: 'none', border: 'none', cursor: 'pointer',
                        color: copied ? 'var(--color-success-fg)' : 'var(--text-muted)', padding: '4px',
                        transition: 'color 0.2s',
                      }}>
                        {copied ? <Check size={16} /> : <Copy size={16} />}
                      </button>
                    </Tooltip>
                  )}
                </div>
              </div>

              <div
                ref={responseRef}
                style={{
                  minHeight: '200px',
                  maxHeight: '520px',
                  overflowY: 'auto',
                  padding: '14px',
                  borderRadius: 'var(--radius-lg)',
                  background: 'var(--bg-elevated)',
                  border: '1px solid var(--border-color)',
                  fontFamily: 'monospace',
                  fontSize: '13px',
                  lineHeight: 1.7,
                  whiteSpace: 'pre-wrap',
                  wordBreak: 'break-word',
                  color: response ? 'var(--text-primary)' : 'var(--text-faint)',
                }}
              >
                {response
                  ? (<>{response}{loading && <span className="animate-nebula-pulse" style={{ opacity: 0.7 }}>▋</span>}</>)
                  : loading
                    ? <span className="animate-nebula-pulse" style={{ color: 'var(--accent-primary)' }}>▋</span>
                    : '响应将显示在这里...'
                }
              </div>
            </CardBody>
          </Card>

        </div>
      </div>
    </div>
  );
}
