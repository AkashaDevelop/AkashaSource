export interface Channel {
  id: number;
  name: string;
  type: number;
  key: string;
  base_url: string;
  models: string;
  group: string;
  model_mapping: string;
  tags: string;
  priority: number;
  weight: number;
  status: number;
  response_time: number;
  balance: number;
  is_custom: number;
  custom_config_id: number;
}

export interface CustomChannelConfig {
  id: number;
  name: string;
  description: string;
}

export interface MultiKeyStatusItem {
  index: number;
  status: number;
  disabled_time: number;
  reason: string;
  key_preview: string;
}

export interface PromptDialogConfig {
  title: string;
  placeholder?: string;
  defaultValue?: string;
  description?: string;
  confirmText?: string;
  multiline?: boolean;
  readOnly?: boolean;
}
