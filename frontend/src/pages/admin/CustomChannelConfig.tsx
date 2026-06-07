import { useState, useEffect } from 'react';
import {
  Button,
  Input,
  Textarea,
  Select,
  SelectItem,
  Modal,
  ModalContent,
  ModalHeader,
  ModalBody,
  ModalFooter,
  useDisclosure,
  Switch,
  Chip,
} from '../../components/ui';
import { Plus, Edit, Trash2, Copy, Settings } from 'lucide-react';
import { useAuthStore } from '../../store/auth';
import { toast } from '../../store/toast';
import { confirm } from '../../store/confirm';
import EmptyState from '../../components/EmptyState';
import LoadingRows from '../../components/LoadingRows';

interface CustomChannelConfig {
  id: number;
  name: string;
  description: string;
  adapter_type: string;
  request_method: string;
  request_endpoint: string;
  request_content_type: string;
  auth_type: string;
  auth_header_name: string;
  auth_header_template: string;
  field_model: string;
  field_messages: string;
  field_temperature: string;
  field_max_tokens: string;
  field_stream: string;
  field_top_p: string;
  field_stop: string;
  response_content_path: string;
  response_usage_path: string;
  response_prompt_tokens_path: string;
  response_completion_tokens_path: string;
  response_total_tokens_path: string;
  response_error_path: string;
  stream_enabled: number;
  stream_data_prefix: string;
  stream_end_marker: string;
  stream_content_path: string;
  timeout: number;
  is_public: number;
  creator_id: number;
  created_at: number;
}

export default function CustomChannelConfig() {
  const [configs, setConfigs] = useState<CustomChannelConfig[]>([]);
  const [loading, setLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const { token } = useAuthStore();
  const { isOpen, onOpen, onOpenChange } = useDisclosure();
  const [editingConfig, setEditingConfig] = useState<CustomChannelConfig | null>(null);

  const [formData, setFormData] = useState({
    name: '',
    description: '',
    adapter_type: 'openai_compatible',
    request_method: 'POST',
    request_endpoint: '/v1/chat/completions',
    request_content_type: 'application/json',
    auth_type: 'bearer',
    auth_header_name: 'Authorization',
    auth_header_template: 'Bearer {key}',
    field_model: 'model',
    field_messages: 'messages',
    field_temperature: 'temperature',
    field_max_tokens: 'max_tokens',
    field_stream: 'stream',
    field_top_p: 'top_p',
    field_stop: 'stop',
    response_content_path: 'choices.0.message.content',
    response_usage_path: 'usage',
    response_prompt_tokens_path: 'usage.prompt_tokens',
    response_completion_tokens_path: 'usage.completion_tokens',
    response_total_tokens_path: 'usage.total_tokens',
    response_error_path: 'error.message',
    stream_enabled: 1,
    stream_data_prefix: 'data: ',
    stream_end_marker: '[DONE]',
    stream_content_path: 'choices.0.delta.content',
    timeout: 120,
    is_public: 0,
  });

  useEffect(() => {
    fetchConfigs();
  }, []);

  const fetchConfigs = async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/custom-channel-config', {
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.code === 0) {
        setConfigs(data.data || []);
      }
    } catch (error) {
      console.error('Failed to fetch configs:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleAdd = () => {
    setEditingConfig(null);
    setFormData({
      name: '',
      description: '',
      adapter_type: 'openai_compatible',
      request_method: 'POST',
      request_endpoint: '/v1/chat/completions',
      request_content_type: 'application/json',
      auth_type: 'bearer',
      auth_header_name: 'Authorization',
      auth_header_template: 'Bearer {key}',
      field_model: 'model',
      field_messages: 'messages',
      field_temperature: 'temperature',
      field_max_tokens: 'max_tokens',
      field_stream: 'stream',
      field_top_p: 'top_p',
      field_stop: 'stop',
      response_content_path: 'choices.0.message.content',
      response_usage_path: 'usage',
      response_prompt_tokens_path: 'usage.prompt_tokens',
      response_completion_tokens_path: 'usage.completion_tokens',
      response_total_tokens_path: 'usage.total_tokens',
      response_error_path: 'error.message',
      stream_enabled: 1,
      stream_data_prefix: 'data: ',
      stream_end_marker: '[DONE]',
      stream_content_path: 'choices.0.delta.content',
      timeout: 120,
      is_public: 0,
    });
    onOpen();
  };

  const handleEdit = (config: CustomChannelConfig) => {
    setEditingConfig(config);
    setFormData({
      name: config.name,
      description: config.description,
      adapter_type: config.adapter_type,
      request_method: config.request_method,
      request_endpoint: config.request_endpoint,
      request_content_type: config.request_content_type,
      auth_type: config.auth_type,
      auth_header_name: config.auth_header_name,
      auth_header_template: config.auth_header_template,
      field_model: config.field_model,
      field_messages: config.field_messages,
      field_temperature: config.field_temperature,
      field_max_tokens: config.field_max_tokens,
      field_stream: config.field_stream,
      field_top_p: config.field_top_p,
      field_stop: config.field_stop,
      response_content_path: config.response_content_path,
      response_usage_path: config.response_usage_path,
      response_prompt_tokens_path: config.response_prompt_tokens_path,
      response_completion_tokens_path: config.response_completion_tokens_path,
      response_total_tokens_path: config.response_total_tokens_path,
      response_error_path: config.response_error_path,
      stream_enabled: config.stream_enabled,
      stream_data_prefix: config.stream_data_prefix,
      stream_end_marker: config.stream_end_marker,
      stream_content_path: config.stream_content_path,
      timeout: config.timeout,
      is_public: config.is_public,
    });
    onOpen();
  };

  const handleSubmit = async (onClose: () => void) => {
    if (!formData.name.trim()) {
      toast.error('请输入配置名称');
      return;
    }

    const url = '/api/custom-channel-config';
    const method = editingConfig ? 'PUT' : 'POST';

    const body = editingConfig
      ? { ...formData, id: editingConfig.id }
      : formData;

    setSubmitting(true);
    try {
      const res = await fetch(url, {
        method,
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify(body),
      });

      const data = await res.json();
      if (data.code !== 0) {
        toast.error(data.msg || '操作失败');
        return;
      }

      toast.success(editingConfig ? '配置更新成功' : '配置创建成功');
      fetchConfigs();
      onClose();
    } catch (error) {
      console.error('Operation error:', error);
      toast.error('请求失败');
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async (id: number) => {
    if (!await confirm({
      title: '删除配置',
      message: '确定要删除这个配置吗？如果有渠道正在使用此配置，将无法删除。',
      danger: true
    })) return;

    try {
      const res = await fetch(`/api/custom-channel-config/${id}`, {
        method: 'DELETE',
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.code === 0) {
        toast.success('删除成功');
        fetchConfigs();
      } else {
        toast.error(data.msg || '删除失败');
      }
    } catch (error) {
      console.error('Delete error:', error);
      toast.error('请求失败');
    }
  };

  const handleCopy = (config: CustomChannelConfig) => {
    setEditingConfig(null);
    setFormData({
      ...formData,
      name: config.name + ' (副本)',
      description: config.description,
      adapter_type: config.adapter_type,
      request_method: config.request_method,
      request_endpoint: config.request_endpoint,
      request_content_type: config.request_content_type,
      auth_type: config.auth_type,
      auth_header_name: config.auth_header_name,
      auth_header_template: config.auth_header_template,
      field_model: config.field_model,
      field_messages: config.field_messages,
      field_temperature: config.field_temperature,
      field_max_tokens: config.field_max_tokens,
      field_stream: config.field_stream,
      field_top_p: config.field_top_p,
      field_stop: config.field_stop,
      response_content_path: config.response_content_path,
      response_usage_path: config.response_usage_path,
      response_prompt_tokens_path: config.response_prompt_tokens_path,
      response_completion_tokens_path: config.response_completion_tokens_path,
      response_total_tokens_path: config.response_total_tokens_path,
      response_error_path: config.response_error_path,
      stream_enabled: config.stream_enabled,
      stream_data_prefix: config.stream_data_prefix,
      stream_end_marker: config.stream_end_marker,
      stream_content_path: config.stream_content_path,
      timeout: config.timeout,
      is_public: 0,
    });
    onOpen();
  };

  return (
    <div className="p-6">
      <div className="flex justify-between items-center mb-6">
        <div>
          <h1 className="text-2xl font-bold">自定义渠道配置</h1>
          <p className="text-sm text-default-500 mt-1">
            管理自定义 API 渠道的配置模板
          </p>
        </div>
        <Button color="primary" startContent={<Plus size={16} />} onPress={handleAdd}>
          新建配置
        </Button>
      </div>

      {loading ? (
        <LoadingRows count={3} />
      ) : configs.length === 0 ? (
        <EmptyState
          title="暂无自定义配置"
          description="点击右上角按钮创建第一个配置模板"
        />
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {configs.map((config) => (
            <div
              key={config.id}
              className="border border-default-200 rounded-lg p-4 hover:shadow-lg transition-shadow"
            >
              <div className="flex justify-between items-start mb-3">
                <div className="flex-1">
                  <h3 className="font-semibold text-lg">{config.name}</h3>
                  {config.description && (
                    <p className="text-sm text-default-500 mt-1">{config.description}</p>
                  )}
                </div>
                {config.is_public === 1 && (
                  <Chip size="sm" color="success" variant="flat">
                    公开
                  </Chip>
                )}
              </div>

              <div className="space-y-2 text-sm mb-4">
                <div className="flex justify-between">
                  <span className="text-default-500">请求端点:</span>
                  <code className="text-xs bg-default-100 px-1 rounded">
                    {config.request_endpoint}
                  </code>
                </div>
                <div className="flex justify-between">
                  <span className="text-default-500">认证方式:</span>
                  <span>{config.auth_type}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-default-500">流式支持:</span>
                  <span>{config.stream_enabled ? '是' : '否'}</span>
                </div>
              </div>

              <div className="flex gap-2">
                <Button
                  size="sm"
                  variant="flat"
                  startContent={<Edit size={14} />}
                  onPress={() => handleEdit(config)}
                  className="flex-1"
                >
                  编辑
                </Button>
                <Button
                  size="sm"
                  variant="flat"
                  color="secondary"
                  startContent={<Copy size={14} />}
                  onPress={() => handleCopy(config)}
                  className="flex-1"
                >
                  复制
                </Button>
                <Button
                  size="sm"
                  variant="flat"
                  color="danger"
                  isIconOnly
                  onPress={() => handleDelete(config.id)}
                >
                  <Trash2 size={14} />
                </Button>
              </div>
            </div>
          ))}
        </div>
      )}

      <Modal isOpen={isOpen} onOpenChange={onOpenChange} size="4xl" scrollBehavior="inside">
        <ModalContent>
          {(onClose) => (
            <>
              <ModalHeader>
                {editingConfig ? '编辑配置' : '新建配置'}
              </ModalHeader>
              <ModalBody>
                <div className="space-y-6">
                  {/* 基础信息 */}
                  <div>
                    <h3 className="text-lg font-semibold mb-3 flex items-center gap-2">
                      <Settings size={18} />
                      基础信息
                    </h3>
                    <div className="grid grid-cols-2 gap-4">
                      <Input
                        label="配置名称"
                        placeholder="例如: DeepSeek Compatible"
                        value={formData.name}
                        onValueChange={(v) => setFormData({ ...formData, name: v })}
                        isRequired
                        className="col-span-2"
                      />
                      <Textarea
                        label="配置描述"
                        placeholder="简要说明此配置的用途"
                        value={formData.description}
                        onValueChange={(v) => setFormData({ ...formData, description: v })}
                        className="col-span-2"
                        minRows={2}
                      />
                      <Select
                        label="适配器类型"
                        selectedKeys={[formData.adapter_type]}
                        onSelectionChange={(keys) =>
                          setFormData({ ...formData, adapter_type: [...keys][0] as string })
                        }
                      >
                        <SelectItem key="openai_compatible">OpenAI 兼容</SelectItem>
                        <SelectItem key="anthropic_compatible">Anthropic 兼容</SelectItem>
                        <SelectItem key="custom">自定义</SelectItem>
                      </Select>
                      <div className="flex items-end">
                        <Switch
                          isSelected={formData.is_public === 1}
                          onValueChange={(v) => setFormData({ ...formData, is_public: v ? 1 : 0 })}
                        >
                          公开配置（允许其他管理员使用）
                        </Switch>
                      </div>
                    </div>
                  </div>

                  {/* 请求配置 */}
                  <div>
                    <h3 className="text-lg font-semibold mb-3">请求配置</h3>
                    <div className="grid grid-cols-2 gap-4">
                      <Select
                        label="请求方法"
                        selectedKeys={[formData.request_method]}
                        onSelectionChange={(keys) =>
                          setFormData({ ...formData, request_method: [...keys][0] as string })
                        }
                      >
                        <SelectItem key="POST">POST</SelectItem>
                        <SelectItem key="GET">GET</SelectItem>
                        <SelectItem key="PUT">PUT</SelectItem>
                      </Select>
                      <Input
                        label="请求端点"
                        placeholder="/v1/chat/completions"
                        value={formData.request_endpoint}
                        onValueChange={(v) => setFormData({ ...formData, request_endpoint: v })}
                      />
                      <Input
                        label="Content-Type"
                        value={formData.request_content_type}
                        onValueChange={(v) => setFormData({ ...formData, request_content_type: v })}
                      />
                      <Select
                        label="认证类型"
                        selectedKeys={[formData.auth_type]}
                        onSelectionChange={(keys) =>
                          setFormData({ ...formData, auth_type: [...keys][0] as string })
                        }
                      >
                        <SelectItem key="bearer">Bearer Token</SelectItem>
                        <SelectItem key="api_key">API Key</SelectItem>
                        <SelectItem key="custom">自定义</SelectItem>
                      </Select>
                      <Input
                        label="认证头名称"
                        value={formData.auth_header_name}
                        onValueChange={(v) => setFormData({ ...formData, auth_header_name: v })}
                      />
                      <Input
                        label="认证头模板"
                        placeholder="Bearer {key}"
                        value={formData.auth_header_template}
                        onValueChange={(v) => setFormData({ ...formData, auth_header_template: v })}
                        description="使用 {key} 作为占位符"
                      />
                    </div>
                  </div>

                  {/* 字段映射 */}
                  <div>
                    <h3 className="text-lg font-semibold mb-3">字段映射（OpenAI → 目标）</h3>
                    <div className="grid grid-cols-2 gap-4">
                      <Input
                        label="model 字段"
                        value={formData.field_model}
                        onValueChange={(v) => setFormData({ ...formData, field_model: v })}
                        description="目标 API 的模型字段名"
                      />
                      <Input
                        label="messages 字段"
                        value={formData.field_messages}
                        onValueChange={(v) => setFormData({ ...formData, field_messages: v })}
                      />
                      <Input
                        label="temperature 字段"
                        value={formData.field_temperature}
                        onValueChange={(v) => setFormData({ ...formData, field_temperature: v })}
                      />
                      <Input
                        label="max_tokens 字段"
                        value={formData.field_max_tokens}
                        onValueChange={(v) => setFormData({ ...formData, field_max_tokens: v })}
                      />
                      <Input
                        label="stream 字段"
                        value={formData.field_stream}
                        onValueChange={(v) => setFormData({ ...formData, field_stream: v })}
                      />
                      <Input
                        label="top_p 字段"
                        value={formData.field_top_p}
                        onValueChange={(v) => setFormData({ ...formData, field_top_p: v })}
                      />
                    </div>
                    <p className="text-xs text-default-500 mt-2">
                      提示：如果目标 API 不支持某个字段，填写 "-" 或留空
                    </p>
                  </div>

                  {/* 响应路径 */}
                  <div>
                    <h3 className="text-lg font-semibold mb-3">响应字段路径（点路径语法）</h3>
                    <div className="grid grid-cols-2 gap-4">
                      <Input
                        label="内容路径"
                        placeholder="choices.0.message.content"
                        value={formData.response_content_path}
                        onValueChange={(v) => setFormData({ ...formData, response_content_path: v })}
                        className="col-span-2"
                      />
                      <Input
                        label="Prompt Tokens 路径"
                        placeholder="usage.prompt_tokens"
                        value={formData.response_prompt_tokens_path}
                        onValueChange={(v) => setFormData({ ...formData, response_prompt_tokens_path: v })}
                      />
                      <Input
                        label="Completion Tokens 路径"
                        placeholder="usage.completion_tokens"
                        value={formData.response_completion_tokens_path}
                        onValueChange={(v) => setFormData({ ...formData, response_completion_tokens_path: v })}
                      />
                      <Input
                        label="Total Tokens 路径"
                        placeholder="usage.total_tokens"
                        value={formData.response_total_tokens_path}
                        onValueChange={(v) => setFormData({ ...formData, response_total_tokens_path: v })}
                      />
                      <Input
                        label="错误信息路径"
                        placeholder="error.message"
                        value={formData.response_error_path}
                        onValueChange={(v) => setFormData({ ...formData, response_error_path: v })}
                      />
                    </div>
                  </div>

                  {/* 流式配置 */}
                  <div>
                    <h3 className="text-lg font-semibold mb-3">流式响应配置</h3>
                    <div className="grid grid-cols-2 gap-4">
                      <div className="col-span-2">
                        <Switch
                          isSelected={formData.stream_enabled === 1}
                          onValueChange={(v) => setFormData({ ...formData, stream_enabled: v ? 1 : 0 })}
                        >
                          支持流式响应（SSE）
                        </Switch>
                      </div>
                      {formData.stream_enabled === 1 && (
                        <>
                          <Input
                            label="数据前缀"
                            placeholder="data: "
                            value={formData.stream_data_prefix}
                            onValueChange={(v) => setFormData({ ...formData, stream_data_prefix: v })}
                          />
                          <Input
                            label="结束标记"
                            placeholder="[DONE]"
                            value={formData.stream_end_marker}
                            onValueChange={(v) => setFormData({ ...formData, stream_end_marker: v })}
                          />
                          <Input
                            label="流内容路径"
                            placeholder="choices.0.delta.content"
                            value={formData.stream_content_path}
                            onValueChange={(v) => setFormData({ ...formData, stream_content_path: v })}
                            className="col-span-2"
                          />
                        </>
                      )}
                    </div>
                  </div>

                  {/* 其他配置 */}
                  <div>
                    <h3 className="text-lg font-semibold mb-3">其他配置</h3>
                    <div className="grid grid-cols-2 gap-4">
                      <Input
                        label="超时时间（秒）"
                        type="number"
                        value={formData.timeout.toString()}
                        onValueChange={(v) => setFormData({ ...formData, timeout: parseInt(v) || 120 })}
                      />
                    </div>
                  </div>
                </div>
              </ModalBody>
              <ModalFooter>
                <Button variant="light" onPress={onClose}>
                  取消
                </Button>
                <Button color="primary" onPress={() => handleSubmit(onClose)} isLoading={submitting}>
                  {editingConfig ? '更新' : '创建'}
                </Button>
              </ModalFooter>
            </>
          )}
        </ModalContent>
      </Modal>
    </div>
  );
}
