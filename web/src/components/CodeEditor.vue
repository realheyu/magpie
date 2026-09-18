<template>
  <div class="code-editor">
    <Codemirror
      :model-value="modelValue || ''"
      :extensions="extensions"
      :disabled="disabled"
      @update:model-value="onUpdate"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Codemirror } from 'vue-codemirror'
import { json } from '@codemirror/lang-json'
import { yaml } from '@codemirror/lang-yaml'
import { StreamLanguage } from '@codemirror/language'
import { toml } from '@codemirror/legacy-modes/mode/toml'
import { properties } from '@codemirror/legacy-modes/mode/properties'
import { placeholder } from '@codemirror/view'

const props = defineProps<{
  modelValue?: string
  language?: string
  disabled?: boolean
}>()
const emit = defineEmits<{ (e: 'update:modelValue', value: string): void }>()

const placeholderExt = placeholder('在这里粘贴完整配置内容')

// 语言跟随“格式”下拉：toml/yaml/json 有专用解析器，properties 走 legacy 模式
const extensions = computed(() => {
  switch (props.language) {
    case 'json':
      return [json(), placeholderExt]
    case 'yaml':
      return [yaml(), placeholderExt]
    case 'toml':
      return [StreamLanguage.define(toml), placeholderExt]
    case 'properties':
      return [StreamLanguage.define(properties), placeholderExt]
    default:
      return [placeholderExt]
  }
})

function onUpdate(value: string) {
  emit('update:modelValue', value)
}
</script>

<style scoped>
.code-editor {
  width: 100%;
}

.code-editor :deep(.cm-editor) {
  height: 460px;
  width: 100%;
  border: 1px solid #dcdfe6;
  border-radius: 6px;
  font-size: 13px;
}

.code-editor :deep(.cm-editor.cm-focused) {
  border-color: #2f7cf6;
}

.code-editor :deep(.cm-scroller) {
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, 'Liberation Mono', monospace;
  line-height: 1.6;
}

.code-editor :deep(.cm-gutters) {
  background: #f7f8fa;
  border-right: 1px solid #e6eaf0;
  color: #9aa7bd;
}
</style>
