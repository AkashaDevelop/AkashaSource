import { useState, useEffect } from 'react';
import {
  Modal,
  ModalContent,
  ModalHeader,
  ModalBody,
  ModalFooter,
  Button,
  Select,
  SelectItem,
  Switch,
  Chip,
  Pagination,
  Input,
  Tooltip,
} from './ui';
import { RefreshCw, CheckCircle, XCircle, Loader2 } from 'lucide-react';
import { toast } from '../store/toast';
import type { Channel } from '../pages/admin/channelTypes';

// ✨ 超萌的渠道测试弹窗组件～
// 支持批量测试多个模型、端点类型选择、实时显示测试结果！

interface ModelTestResult {
  model: string;
  status: '未测试' | '测试中' | '成功' | '失败';
  response_time: number;
  error: string;
}

interface ChannelTestDialogProps {
  isOpen: boolean;
  onOpenChange: () => void;
  channel: Channel | null;
  token: string;
  onTestComplete?: () => void;
}

export default function ChannelTestDialog({
  isOpen,
  onOpenChange,
  channel,
  token,
  onTestComplete,
}: ChannelTestDialogProps) {
  const [endpointType, setEndpointType] = useState('auto');
  const [testMode, setTestMode] = useState(false);
  const [modelResults, setModelResults] = useState<ModelTestResult[]>([]);
  const [testing, setTesting] = useState(false);
  const [currentPage, setCurrentPage] = useState(1);
  const [customPrompt, setCustomPrompt] = useState('');
  const pageSize = 30;

  useEffect(() => {
    if (isOpen && channel) {
      // 初始化模型列表
      const models = channel.models
        ? channel.models.split(',').map(m => m.trim()).filter(Boolean)
        : [];
      setModelResults(
        models.map(model => ({
          model,
          status: '未测试',
          response_time: 0,
          error: '',
        }))
      );
      setCurrentPage(1);
    }
  }, [isOpen, channel]);

  const paginatedResults = modelResults.slice(
    (currentPage - 1) * pageSize,
    currentPage * pageSize
  );
  const totalPages = Math.ceil(modelResults.length / pageSize);

  const handleTestAll = async () => {
    if (!channel) return;

    setTesting(true);
    const models = modelResults.map(r => r.model);

    // 设置所有模型为"测试中"状态
    setModelResults(prev =>
      prev.map(r => ({ ...r, status: '测试中' }))
    );

    try {
      const res = await fetch(`/api/channel/${channel.id}/test-batch`, {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          models,
          prompt: customPrompt.trim() || undefined
        }),
      });

      const data = await res.json();
      if (data.code === 0 && data.data?.results) {
        setModelResults(data.data.results);
        toast.success(
          `测试完成：${data.data.success_count}/${data.data.total} 成功`
        );
        onTestComplete?.();
      } else {
        toast.error(data.msg || '批量测试失败');
        setModelResults(prev =>
          prev.map(r => ({ ...r, status: '失败', error: data.msg || '测试失败' }))
        );
      }
    } catch (error) {
      console.error('Batch test error:', error);
      toast.error('批量测试请求失败');
      setModelResults(prev =>
        prev.map(r => ({ ...r, status: '失败', error: '请求失败' }))
      );
    } finally {
      setTesting(false);
    }
  };

  const handleTestSingle = async (model: string) => {
    if (!channel) return;

    // 设置单个模型为"测试中"状态
    setModelResults(prev =>
      prev.map(r =>
        r.model === model ? { ...r, status: '测试中' } : r
      )
    );

    try {
      const res = await fetch(`/api/channel/test-model/${channel.id}`, {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          model,
          prompt: customPrompt.trim() || undefined
        }),
      });

      const data = await res.json();
      if (data.code === 0 && data.data?.success) {
        setModelResults(prev =>
          prev.map(r =>
            r.model === model
              ? {
                  ...r,
                  status: '成功',
                  response_time: data.data.time,
                  error: '',
                }
              : r
          )
        );
        toast.success(`${model} 测试通过 (${data.data.time}ms)`);
      } else {
        const errorMsg = data.data?.msg || data.msg || '测试失败';
        setModelResults(prev =>
          prev.map(r =>
            r.model === model
              ? {
                  ...r,
                  status: '失败',
                  error: errorMsg,
                }
              : r
          )
        );
        toast.error(`${model} 测试失败: ${errorMsg}`);
      }
    } catch (error) {
      console.error('Single model test error:', error);
      const errorMsg = error instanceof Error ? error.message : '请求失败';
      setModelResults(prev =>
        prev.map(r =>
          r.model === model
            ? { ...r, status: '失败', error: errorMsg }
            : r
        )
      );
      toast.error(`测试失败: ${errorMsg}`);
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case '成功':
        return <CheckCircle size={16} className="text-success" />;
      case '失败':
        return <XCircle size={16} className="text-danger" />;
      case '测试中':
        return <Loader2 size={16} className="text-primary animate-spin" />;
      default:
        return <span className="text-default-400">-</span>;
    }
  };

  return (
    <Modal
      isOpen={isOpen}
      onOpenChange={onOpenChange}
      size="xl"
      scrollBehavior="inside"
    >
      <ModalContent>
        {onClose => (
          <>
            <ModalHeader>
              <div className="flex flex-col gap-1">
                <span>测试渠道连接: {channel?.name}</span>
                <span className="text-xs text-default-500 font-normal">
                  批量测试模型可用性
                </span>
              </div>
            </ModalHeader>
            <ModalBody className="space-y-3">
              {/* 自定义测试问题 */}
              <div className="space-y-1">
                <label className="text-xs font-medium text-default-700">测试问题（可选）</label>
                <Input
                  size="sm"
                  placeholder="留空使用默认问题（如：你好）"
                  value={customPrompt}
                  onValueChange={setCustomPrompt}
                  description="自定义测试时发送的问题"
                />
              </div>

              {/* 端点类型和测试模式 - 横向布局 */}
              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1">
                  <label className="text-xs font-medium">端点类型</label>
                  <Select
                    selectedKeys={[endpointType]}
                    onSelectionChange={keys =>
                      setEndpointType([...keys][0] as string)
                    }
                    placeholder="选择端点类型"
                    size="sm"
                  >
                    <SelectItem key="auto">自动检测</SelectItem>
                    <SelectItem key="openai">OpenAI</SelectItem>
                    <SelectItem key="azure">Azure</SelectItem>
                    <SelectItem key="claude">Claude</SelectItem>
                    <SelectItem key="gemini">Gemini</SelectItem>
                  </Select>
                </div>

                <div className="space-y-1">
                  <label className="text-xs font-medium">测试模式</label>
                  <div className="h-10 flex items-center">
                    <Switch
                      isSelected={testMode}
                      onValueChange={setTestMode}
                      size="sm"
                    >
                      <span className="text-sm">{testMode ? '已启用' : '已禁用'}</span>
                    </Switch>
                  </div>
                </div>
              </div>

              {/* 渠道模型列表 */}
              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <span className="text-sm font-medium text-default-700">
                    渠道模型 ({modelResults.length})
                  </span>
                  <Button
                    size="sm"
                    color="primary"
                    variant="solid"
                    startContent={<RefreshCw size={14} />}
                    onPress={handleTestAll}
                    isLoading={testing}
                    isDisabled={modelResults.length === 0}
                  >
                    测试全部
                  </Button>
                </div>

                {/* 模型测试结果表格 - 紧凑版 */}
                <div className="border border-default-300 rounded-lg overflow-hidden max-h-[400px] overflow-y-auto bg-white dark:bg-default-50">
                  <table className="w-full text-xs">
                    <thead className="bg-default-200 dark:bg-default-100 sticky top-0">
                      <tr>
                        <th className="px-3 py-2 text-left font-semibold text-default-700">模型</th>
                        <th className="px-3 py-2 text-center font-semibold text-default-700 w-16">
                          状态
                        </th>
                        <th className="px-3 py-2 text-center font-semibold text-default-700 w-24">
                          结果
                        </th>
                        <th className="px-3 py-2 text-center font-semibold text-default-700 w-16">
                          操作
                        </th>
                      </tr>
                    </thead>
                    <tbody>
                      {modelResults.length === 0 ? (
                        <tr>
                          <td
                            colSpan={4}
                            className="px-3 py-6 text-center text-default-500 text-xs"
                          >
                            该渠道未配置模型
                          </td>
                        </tr>
                      ) : (
                        paginatedResults.map((result, idx) => (
                          <tr
                            key={result.model}
                            className={`border-t border-default-200 hover:bg-default-100 dark:hover:bg-default-100 ${
                              idx % 2 === 0 ? 'bg-white dark:bg-default-50' : 'bg-default-50 dark:bg-white'
                            }`}
                          >
                            <td className="px-3 py-2 font-mono text-xs truncate max-w-[200px] text-default-700" title={result.model}>
                              {result.model}
                            </td>
                            <td className="px-3 py-2 text-center">
                              {getStatusIcon(result.status)}
                            </td>
                            <td className="px-3 py-2 text-center">
                              {result.status === '成功' ? (
                                <Chip
                                  size="sm"
                                  color="success"
                                  variant="flat"
                                  className="text-xs h-6"
                                >
                                  {result.response_time}ms
                                </Chip>
                              ) : result.status === '失败' ? (
                                <Tooltip content={result.error || '测试失败'}>
                                  <Chip
                                    size="sm"
                                    color="danger"
                                    variant="flat"
                                    className="text-xs h-6 max-w-[100px] truncate cursor-help"
                                  >
                                    {result.error || '失败'}
                                  </Chip>
                                </Tooltip>
                              ) : (
                                <span className="text-default-400">-</span>
                              )}
                            </td>
                            <td className="px-3 py-2 text-center">
                              <Button
                                size="sm"
                                color="primary"
                                variant="flat"
                                isIconOnly
                                onPress={() => handleTestSingle(result.model)}
                                isDisabled={
                                  testing || result.status === '测试中'
                                }
                                className="h-7 w-7 min-w-7"
                              >
                                <RefreshCw
                                  size={14}
                                  className={
                                    result.status === '测试中'
                                      ? 'animate-spin'
                                      : ''
                                  }
                                />
                              </Button>
                            </td>
                          </tr>
                        ))
                      )}
                    </tbody>
                  </table>
                </div>

                {/* 分页 - 紧凑版 */}
                {totalPages > 1 && (
                  <div className="flex items-center justify-between pt-1">
                    <span className="text-xs text-default-500">
                      共 {modelResults.length} 个，每页 {pageSize} 个
                    </span>
                    <Pagination
                      total={totalPages}
                      page={currentPage}
                      onChange={setCurrentPage}
                    />
                  </div>
                )}
              </div>
            </ModalBody>
            <ModalFooter>
              <Button color="danger" variant="light" onPress={onClose} size="sm">
                关闭
              </Button>
            </ModalFooter>
          </>
        )}
      </ModalContent>
    </Modal>
  );
}
