import { useState, useEffect, useRef, type ReactNode } from 'react';
import PageHeader from '../components/PageHeader';
import {
  Card, CardBody, CardHeader,
  Button, Input, Select, SelectItem, Textarea, Switch, Divider, Chip, Tooltip,
} from '../components/ui';
import {
  Play, Square, Trash2, Copy, Check, Settings2, MessageSquare, User, Bot,
  Plus, Clock, Coins, ChevronDown, ChevronUp,
} from 'lucide-react';
import { useAuthStore } from '../store/auth';
import { toast } from '../store/toast';

interface ModelItem { id: string; }
interface Usage { prompt_tokens: number; completion_tokens: number; total_tokens: number; }
interface ChatMessage { role: 'user' | 'assistant'; content: string; }

/** 消息气泡 */
function MessageBubble({ msg, streaming }: { msg: ChatMessage; streaming?: boolean }) {
  const isUser = msg.role === 'user';
  return (
    <div className={`flex gap-3 ${isUser ? 'flex-row-reverse' : ''}`}>
      <div className="flex-shrink-0 w-8 h-8 rounded-lg flex items-center justify-center"
        style={{
          background: isUser ? 'var(--color-info-bg)' : 'var(--nav-active-bg)',
          color: isUser ? 'var(--accent-cosmic)' : 'var(--accent-primary)',
        }}>
        {isUser ? <User size={16} /> : <Bot size={16} />}
      </div>
      <div className="min-w-0 max-w-[80%] px-4 py-3 rounded-2xl text-sm leading-relaxed whitespace-pre-wrap break-words"
        style={{
          background: isUser ? 'var(--color-info-bg)' : 'var(--bg-elevated)',
          border: '1px solid var(--border-color)',
          color: 'var(--text-primary)',
          borderTopRightRadius: isUser ? '4px' : undefined,
          borderTopLeftRadius: !isUser ? '4px' : undefined,
        }}>
        {msg.content}
        {streaming && <span className="animate-nebula-pulse" style={{ opacity: 0.7 }}>▋</span>}
      </div>
    </div>
  );
}

/** 配置项行 */
function ConfigRow({ label, description, children }: { label: string; description?: string; children: ReactNode }) {
  return (
    <div className="flex items-center justify-between gap-3">
      <div className="min-w-0">
        <p className="text-xs font-medium" style={{ color: 'var(--text-secondary)' }}>{label}</p>
        {description && <p className="text-[11px] mt-0.5" style={{ color: 'var(--text-faint)' }}>{description}</p>}
      </div>
      {children}
    </div>
  );
}

/** 统计芯片 */
function StatChip({ icon, label, value, color }: { icon: ReactNode; label: string; value: string | number; color?: string }) {
  return (
    <div className="flex items-center gap-1.5 px-2.5 py-1 rounded-lg"
      style={{ background: 'var(--bg-elevated)', border: '1px solid var(--border-color)' }}>
      <span style={{ color: color || 'var(--text-muted)', display: 'flex' }}>{icon}</span>
      <span className="text-[11px]" style={{ color: 'var(--text-muted)' }}>{label}</span>
      <span className="text-[11px] font-semibold" style={{ color: 'var(--text-primary)' }}>{value}</span>
    </div>
  );
}

export default function Playground() {
  const { token } = useAuthStore();
  const [models, setModels] = useState<ModelItem[]>([]);
  const [selectedModel, setSelectedModel] = useState('');
  const [systemPrompt, setSystemPrompt] = useState('');
  const [userMessage, setUserMessage] = useState('');
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [streamingContent, setStreamingContent] = useState('');
  const [usage, setUsage] = useState<Usage | null>(null);
  const [elapsed, setElapsed] = useState(0);
  const [loading, setLoading] = useState(false);
  const [stream, setStream] = useState(true);
  const [temperature, setTemperature] = useState('0.7');
  const [maxTokens, setMaxTokens] = useState('2048');
  const [copied, setCopied] = useState(false);
  const [showRaw, setShowRaw] = useState(false);
  const [lastRequest, setLastRequest] = useState('');
  const abortRef = useRef<AbortController | null>(null);
  const messagesRef = useRef<HTMLDivElement>(null);
  const startTimeRef = useRef<number>(0);

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

  useEffect(() => {
    if (messagesRef.current) {
      messagesRef.current.scrollTop = messagesRef.current.scrollHeight;
    }
  }, [messages, streamingContent]);

  const handleStop = () => {
    abortRef.current?.abort();
    if (streamingContent) {
      setMessages(prev => [...prev, { role: 'assistant', content: streamingContent }]);
      setStreamingContent('');
    }
    setLoading(false);
  };

  const handleNewChat = () => {
    setMessages([]);
    setStreamingContent('');
    setUsage(null);
    setElapsed(0);
    setUserMessage('');
    setLastRequest('');
  };

  const handleCopy = async () => {
    const lastAssistant = [...messages].reverse().find(m => m.role === 'assistant');
    const text = lastAssistant?.content || streamingContent;
    if (!text) return;
    await navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const handleSend = async () => {
    if (!userMessage.trim() || !selectedModel || loading) return;
    const currentMessage = userMessage.trim();
    setLoading(true);
    setStreamingContent('');

    // 构建消息列表
    const apiMessages: any[] = [];
    if (systemPrompt.trim()) apiMessages.push({ role: 'system', content: systemPrompt });
    for (const m of messages) apiMessages.push({ role: m.role, content: m.content });
    apiMessages.push({ role: 'user', content: currentMessage });

    const body = {
      model: selectedModel,
      messages: apiMessages,
      stream,
      temperature: parseFloat(temperature) || 0.7,
      max_tokens: parseInt(maxTokens) || 2048,
    };
    setLastRequest(JSON.stringify(body, null, 2));

    // 添加用户消息到界面
    setMessages(prev => [...prev, { role: 'user', content: currentMessage }]);
    setUserMessage('');

    const controller = new AbortController();
    abortRef.current = controller;
    startTimeRef.current = Date.now();

    try {
      const res = await fetch('/v1/chat/completions', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify(body),
        signal: controller.signal,
      });

      if (!res.ok) {
        const err = await res.json().catch(() => ({}));
        const errMsg = err.error?.message || res.statusText;
        setMessages(prev => [...prev, { role: 'assistant', content: `⚠️ 错误：${errMsg}` }]);
        setLoading(false);
        return;
      }

      let assistantContent = '';

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
              if (content) {
                assistantContent += content;
                setStreamingContent(assistantContent);
              }
              if (chunk.usage) setUsage(chunk.usage);
            } catch { /* skip */ }
          }
        }
      } else {
        const data = await res.json();
        assistantContent = data.choices?.[0]?.message?.content || '（无响应）';
        if (data.usage) setUsage(data.usage);
      }

      // 添加助手回复
      if (assistantContent) {
        setMessages(prev => [...prev, { role: 'assistant', content: assistantContent }]);
      }
      setStreamingContent('');
      setElapsed(Date.now() - startTimeRef.current);
    } catch (e: any) {
      if (e.name !== 'AbortError') {
        setMessages(prev => [...prev, { role: 'assistant', content: `⚠️ 请求失败：${e.message}` }]);
        toast.error('请求失败');
      }
    } finally {
      setLoading(false);
      setElapsed(Date.now() - startTimeRef.current);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) {
      e.preventDefault();
      handleSend();
    }
  };

  return (
    <div className="space-y-5">
      <PageHeader
        title="API 调试台"
        description="在线测试模型接口，支持多轮对话与流式输出"
        actions={
          <Button size="sm" variant="flat" startContent={<Plus size={15} />} onPress={handleNewChat}>
            新对话
          </Button>
        }
      />

      <div className="grid grid-cols-1 lg:grid-cols-4 gap-5">

        {/* ════════ 左侧配置面板 ════════ */}
        <div className="lg:col-span-1 space-y-4">
          <Card>
            <CardHeader>
              <div className="flex items-center gap-2">
                <Settings2 size={16} style={{ color: 'var(--accent-primary)' }} />
                <span className="text-sm font-semibold" style={{ color: 'var(--text-primary)' }}>请求配置</span>
              </div>
            </CardHeader>
            <CardBody className="gap-4">
              <Select
                label="模型"
                size="sm"
                selectedKeys={selectedModel ? [selectedModel] : []}
                onSelectionChange={keys => setSelectedModel([...keys][0] as string || '')}
              >
                {models.map(m => <SelectItem key={m.id}>{m.id}</SelectItem>)}
              </Select>

              <div className="flex flex-col gap-1.5">
                <div className="flex items-center justify-between">
                  <span className="text-xs font-medium" style={{ color: 'var(--text-secondary)' }}>Temperature</span>
                  <Chip size="sm" variant="flat" className="text-[11px]">{temperature}</Chip>
                </div>
                <input
                  type="range"
                  min="0"
                  max="2"
                  step="0.1"
                  value={temperature}
                  onChange={e => setTemperature(e.target.value)}
                  className="w-full h-1.5 rounded-full appearance-none cursor-pointer"
                  style={{ background: 'var(--bg-elevated)', accentColor: 'var(--accent-primary)' }}
                />
                <div className="flex justify-between text-[10px]" style={{ color: 'var(--text-faint)' }}>
                  <span>精确</span>
                  <span>随机</span>
                </div>
              </div>

              <Input
                label="最大 Token 数"
                size="sm"
                type="number"
                value={maxTokens}
                onValueChange={setMaxTokens}
              />

              <Divider />

              <ConfigRow label="流式输出" description="逐字返回响应">
                <Switch isSelected={stream} onValueChange={setStream} size="sm" />
              </ConfigRow>

              <Divider />

              <Textarea
                label="系统提示词"
                placeholder="设定模型角色、行为约束..."
                value={systemPrompt}
                onValueChange={setSystemPrompt}
                minRows={3}
                maxRows={8}
              />
            </CardBody>
          </Card>

          {/* 原始请求 JSON */}
          {lastRequest && (
            <Card>
              <CardHeader>
                <button
                  type="button"
                  onClick={() => setShowRaw(v => !v)}
                  className="flex items-center gap-2 w-full"
                >
                  <span className="text-sm font-semibold" style={{ color: 'var(--text-primary)' }}>请求详情</span>
                  {showRaw
                    ? <ChevronUp size={14} style={{ color: 'var(--text-muted)' }} />
                    : <ChevronDown size={14} style={{ color: 'var(--text-muted)' }} />}
                </button>
              </CardHeader>
              {showRaw && (
                <CardBody>
                  <pre
                    className="text-[11px] font-mono overflow-auto rounded-lg p-3"
                    style={{
                      background: 'var(--bg-elevated)',
                      border: '1px solid var(--border-color)',
                      color: 'var(--text-secondary)',
                      maxHeight: '300px',
                    }}
                  >
                    {lastRequest}
                  </pre>
                </CardBody>
              )}
            </Card>
          )}
        </div>

        {/* ════════ 右侧对话区 ════════ */}
        <div className="lg:col-span-3">
          <Card className="flex flex-col" style={{ minHeight: 'calc(100vh - 200px)' }}>
            {/* 对话工具栏 */}
            <div className="flex items-center justify-between px-5 py-3 border-b" style={{ borderColor: 'var(--border-color)' }}>
              <div className="flex items-center gap-2">
                <MessageSquare size={16} style={{ color: 'var(--accent-primary)' }} />
                <span className="text-sm font-semibold" style={{ color: 'var(--text-primary)' }}>对话</span>
                {selectedModel && (
                  <Chip size="sm" variant="flat" color="primary">{selectedModel}</Chip>
                )}
              </div>
              <div className="flex items-center gap-2">
                {messages.length > 0 && (
                  <Tooltip content={copied ? '已复制' : '复制最后回复'}>
                    <button
                      onClick={handleCopy}
                      className="p-1.5 rounded-lg transition-colors"
                      style={{
                        color: copied ? 'var(--color-success-fg)' : 'var(--text-muted)',
                        background: 'var(--bg-elevated)',
                        border: '1px solid var(--border-color)',
                      }}
                    >
                      {copied ? <Check size={14} /> : <Copy size={14} />}
                    </button>
                  </Tooltip>
                )}
                <Tooltip content="清空对话">
                  <button
                    onClick={handleNewChat}
                    className="p-1.5 rounded-lg transition-colors"
                    style={{
                      color: 'var(--text-muted)',
                      background: 'var(--bg-elevated)',
                      border: '1px solid var(--border-color)',
                    }}
                  >
                    <Trash2 size={14} />
                  </button>
                </Tooltip>
              </div>
            </div>

            {/* 消息列表 */}
            <div
              ref={messagesRef}
              className="flex-1 overflow-y-auto p-5 space-y-4"
              style={{ minHeight: '300px' }}
            >
              {messages.length === 0 && !streamingContent && !loading ? (
                <div className="flex flex-col items-center justify-center h-full gap-3 py-20">
                  <div className="w-16 h-16 rounded-2xl flex items-center justify-center"
                    style={{ background: 'var(--nav-active-bg)' }}>
                    <Bot size={32} style={{ color: 'var(--accent-primary)' }} />
                  </div>
                  <div className="text-center">
                    <p className="text-sm font-medium" style={{ color: 'var(--text-secondary)' }}>开始对话</p>
                    <p className="text-xs mt-1" style={{ color: 'var(--text-faint)' }}>
                      输入消息，按 Ctrl+Enter 发送
                    </p>
                  </div>
                </div>
              ) : (
                <>
                  {messages.map((msg, i) => (
                    <MessageBubble key={i} msg={msg} />
                  ))}
                  {streamingContent && (
                    <MessageBubble msg={{ role: 'assistant', content: streamingContent }} streaming />
                  )}
                  {loading && !streamingContent && (
                    <div className="flex gap-3">
                      <div className="flex-shrink-0 w-8 h-8 rounded-lg flex items-center justify-center"
                        style={{ background: 'var(--nav-active-bg)', color: 'var(--accent-primary)' }}>
                        <Bot size={16} />
                      </div>
                      <div className="px-4 py-3 rounded-2xl" style={{ background: 'var(--bg-elevated)', border: '1px solid var(--border-color)' }}>
                        <span className="animate-nebula-pulse" style={{ color: 'var(--accent-primary)' }}>▋</span>
                      </div>
                    </div>
                  )}
                </>
              )}
            </div>

            {/* 统计栏 */}
            {(usage || elapsed > 0) && (
              <div className="flex items-center gap-2 px-5 py-2 border-t flex-wrap" style={{ borderColor: 'var(--border-color)' }}>
                {usage && (
                  <>
                    <StatChip icon={<Coins size={12} />} label="提示" value={usage.prompt_tokens} />
                    <StatChip icon={<Coins size={12} />} label="补全" value={usage.completion_tokens} color="var(--color-success-fg)" />
                    <StatChip icon={<Coins size={12} />} label="总计" value={usage.total_tokens} color="var(--accent-primary)" />
                  </>
                )}
                {elapsed > 0 && (
                  <StatChip icon={<Clock size={12} />} label="耗时" value={`${(elapsed / 1000).toFixed(2)}s`} color="var(--accent-cosmic)" />
                )}
              </div>
            )}

            {/* 输入区 */}
            <div className="p-4 border-t" style={{ borderColor: 'var(--border-color)' }}>
              <div className="relative">
                <Textarea
                  placeholder="输入消息，Ctrl+Enter 发送..."
                  value={userMessage}
                  onValueChange={setUserMessage}
                  minRows={2}
                  maxRows={6}
                  onKeyDown={handleKeyDown}
                  isDisabled={loading}
                />
              </div>
              <div className="flex items-center justify-between mt-3">
                <span className="text-[11px]" style={{ color: 'var(--text-faint)' }}>
                  {loading ? '生成中...' : 'Ctrl+Enter 发送'}
                </span>
                <div className="flex gap-2">
                  {loading ? (
                    <Button color="danger" size="sm" variant="flat" startContent={<Square size={14} />} onPress={handleStop}>
                      停止
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
                </div>
              </div>
            </div>
          </Card>
        </div>
      </div>
    </div>
  );
}
