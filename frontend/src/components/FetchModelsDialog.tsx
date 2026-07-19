import { useState, useMemo, useEffect } from 'react';
import { Search, Check, ChevronDown } from 'lucide-react';
import {
  Modal,
  ModalContent,
  ModalHeader,
  ModalBody,
  ModalFooter,
  Button,
  Input,
  Checkbox,
} from './ui';

// ✨🌸 从上游拉取模型の小窗～
// 拉取回来的模型不要一股脑塞进表单啦，先弹个窗让主人挑一挑嘛(๑˃̵ᴗ˂̵)
// 支持搜索、按"新模型 / 现有模型"分区、按供应商分组勾选，最后一键保存回表单～

export interface FetchModelsDialogProps {
  isOpen: boolean;
  onClose: () => void;
  channelName: string;           // 渠道名，显示在标题下方
  vendorLabel: string;           // 供应商分组名（如 OpenAI）
  fetchedModels: string[];       // 从上游拉回来的模型列表
  existingModels: string[];      // 当前表单已有的模型（用来区分新/旧）
  onSave: (selected: string[]) => void; // 保存时把最终勾选结果吐回去～
}

type TabKey = 'new' | 'existing';

export default function FetchModelsDialog({
  isOpen,
  onClose,
  channelName,
  vendorLabel,
  fetchedModels,
  existingModels,
  onSave,
}: FetchModelsDialogProps) {
  const [keyword, setKeyword] = useState('');
  const [tab, setTab] = useState<TabKey>('existing');
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [groupOpen, setGroupOpen] = useState(true);

  // 已有模型集合，用来判断某个模型是不是"新来的"～
  const existingSet = useMemo(
    () => new Set(existingModels.map((m) => m.trim()).filter(Boolean)),
    [existingModels]
  );

  // 把拉回来的模型拆成「新模型」和「现有模型」两拨～
  const { newModels, keepModels } = useMemo(() => {
    const uniq = Array.from(new Set(fetchedModels.map((m) => m.trim()).filter(Boolean)));
    const fresh: string[] = [];
    const kept: string[] = [];
    for (const m of uniq) {
      if (existingSet.has(m)) kept.push(m);
      else fresh.push(m);
    }
    return { newModels: fresh, keepModels: kept };
  }, [fetchedModels, existingSet]);

  // 每次打开时重置：默认全选拉回来的所有模型，Tab 落在有内容的一边～
  useEffect(() => {
    if (!isOpen) return;
    const all = Array.from(new Set(fetchedModels.map((m) => m.trim()).filter(Boolean)));
    setSelected(new Set(all));
    setKeyword('');
    setGroupOpen(true);
    setTab(newModels.length > 0 ? 'new' : 'existing');
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isOpen]);

  // 当前 Tab 下、经过搜索过滤后的模型列表～
  const currentList = tab === 'new' ? newModels : keepModels;
  const filtered = useMemo(() => {
    const kw = keyword.trim().toLowerCase();
    if (!kw) return currentList;
    return currentList.filter((m) => m.toLowerCase().includes(kw));
  }, [currentList, keyword]);

  const toggleOne = (name: string) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(name)) next.delete(name);
      else next.add(name);
      return next;
    });
  };

  // 当前分组全选/取消全选～
  const allInGroupSelected = filtered.length > 0 && filtered.every((m) => selected.has(m));
  const toggleGroupAll = () => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (allInGroupSelected) {
        filtered.forEach((m) => next.delete(m));
      } else {
        filtered.forEach((m) => next.add(m));
      }
      return next;
    });
  };

  const groupSelectedCount = filtered.filter((m) => selected.has(m)).length;

  const handleSave = () => {
    onSave(Array.from(selected));
    onClose();
  };

  return (
    <Modal isOpen={isOpen} onClose={onClose} size="lg" scrollBehavior="inside">
      <ModalContent>
        {(close) => (
          <>
            <ModalHeader onClose={close}>
              <div className="flex flex-col gap-0.5">
                <span>获取模型</span>
                <span className="text-xs font-normal text-[var(--text-muted)]">
                  渠道：<span className="font-semibold text-[var(--text-secondary)]">{channelName || '未命名渠道'}</span>
                </span>
              </div>
            </ModalHeader>

            <ModalBody className="space-y-3">
              {/* 🔍 搜索框 */}
              <Input
                size="sm"
                placeholder="搜索模型..."
                value={keyword}
                onValueChange={setKeyword}
                startContent={<Search size={15} />}
              />

              {/* 🗂️ 新模型 / 现有模型 两个 Tab */}
              <div className="flex items-center gap-1 p-1 rounded-xl bg-[var(--bg-elevated)] border border-[var(--border-color)]">
                <button
                  type="button"
                  onClick={() => setTab('new')}
                  className={`flex-1 text-sm py-1.5 rounded-lg transition-all duration-150 ${
                    tab === 'new'
                      ? 'bg-[var(--bg-surface)] font-semibold text-[var(--text-primary)] shadow-sm'
                      : 'text-[var(--text-muted)] hover:text-[var(--text-secondary)]'
                  }`}
                >
                  新模型 ({newModels.length})
                </button>
                <button
                  type="button"
                  onClick={() => setTab('existing')}
                  className={`flex-1 text-sm py-1.5 rounded-lg transition-all duration-150 ${
                    tab === 'existing'
                      ? 'bg-[var(--bg-surface)] font-semibold text-[var(--text-primary)] shadow-sm'
                      : 'text-[var(--text-muted)] hover:text-[var(--text-secondary)]'
                  }`}
                >
                  现有模型 ({keepModels.length})
                </button>
              </div>

              {/* 📦 分组卡片：供应商分组 + 全选 */}
              <div className="border border-[var(--border-color)] rounded-xl overflow-hidden">
                <button
                  type="button"
                  onClick={() => setGroupOpen((v) => !v)}
                  className="w-full flex items-center justify-between px-3 py-2.5 bg-[var(--bg-elevated)] hover:bg-[var(--nav-hover-bg)] transition-colors"
                >
                  <div className="flex items-center gap-1.5">
                    <ChevronDown
                      size={16}
                      className={`transition-transform duration-200 ${groupOpen ? '' : '-rotate-90'}`}
                    />
                    <span className="text-sm font-semibold">{vendorLabel} ({currentList.length})</span>
                  </div>
                  <div
                    className="flex items-center gap-1.5"
                    onClick={(e) => {
                      e.stopPropagation();
                      toggleGroupAll();
                    }}
                  >
                    <span className="text-xs text-[var(--text-muted)]">
                      {groupSelectedCount} / {currentList.length} selected
                    </span>
                    <span
                      className={`w-4 h-4 rounded-full flex items-center justify-center ${
                        allInGroupSelected
                          ? 'bg-gradient-to-br from-[var(--accent-primary)] to-[var(--accent-cosmic)]'
                          : 'border border-[var(--border-strong)]'
                      }`}
                    >
                      {allInGroupSelected && <Check size={10} className="text-white" strokeWidth={3} />}
                    </span>
                  </div>
                </button>

                {groupOpen && (
                  <div className="max-h-[280px] overflow-y-auto p-2">
                    {filtered.length === 0 ? (
                      <div className="flex items-center justify-center py-8 text-sm text-[var(--text-faint)]">
                        {keyword ? '没有匹配的模型～' : '这里空空如也 (´･ω･`)'}
                      </div>
                    ) : (
                      <div className="grid grid-cols-2 gap-x-4 gap-y-1">
                        {filtered.map((m) => (
                          <label
                            key={m}
                            className="flex items-center gap-2 px-2 py-1.5 rounded-lg hover:bg-[var(--nav-hover-bg)] cursor-pointer transition-colors"
                          >
                            <Checkbox
                              isSelected={selected.has(m)}
                              onValueChange={() => toggleOne(m)}
                            />
                            <span className="text-sm text-[var(--text-primary)] truncate" title={m}>
                              {m}
                            </span>
                          </label>
                        ))}
                      </div>
                    )}
                  </div>
                )}
              </div>

              {/* 🧮 已选统计条 */}
              <div className="px-3 py-2.5 rounded-xl bg-[var(--bg-elevated)] border border-[var(--border-color)] text-sm text-[var(--text-secondary)]">
                已选 <span className="font-semibold text-[var(--text-primary)]">{selected.size}</span> 个模型
              </div>
            </ModalBody>

            <ModalFooter>
              <Button variant="bordered" onPress={close}>
                取消
              </Button>
              <Button color="primary" onPress={handleSave} isDisabled={selected.size === 0}>
                保存模型
              </Button>
            </ModalFooter>
          </>
        )}
      </ModalContent>
    </Modal>
  );
}
