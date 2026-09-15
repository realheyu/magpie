<template>
  <section class="page">
    <div class="page-header">
      <div>
        <h1>应用配置</h1>
        <p>每个应用维护一份完整配置内容。</p>
      </div>
      <div class="toolbar">
        <el-input v-model="query" placeholder="搜索应用" clearable @change="search" />
        <el-button type="primary" :icon="Plus" v-if="isAdmin" @click="openCreate">新建应用</el-button>
      </div>
    </div>

    <el-table :data="apps" v-loading="loading" class="data-table" @row-click="selectApp">
      <el-table-column prop="appName" label="应用" min-width="180" />
      <el-table-column prop="format" label="格式" width="110" />
      <el-table-column prop="version" label="版本" width="110" />
      <el-table-column label="权限" width="130">
        <template #default="{ row }">{{ permissionText(row.permission) }}</template>
      </el-table-column>
      <el-table-column label="敏感配置" width="120">
        <template #default="{ row }">
          <el-tag :type="row.sensitive ? 'warning' : 'info'">{{ row.sensitive ? '是' : '否' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="updatedAt" label="更新时间" min-width="180" />
    </el-table>

    <div class="pagination-bar">
      <el-pagination
        v-model:current-page="page"
        :page-size="pageSize"
        :total="total"
        layout="total, prev, pager, next"
        @current-change="load"
      />
    </div>

    <el-drawer v-model="editorOpen" size="58%" :title="editing?.appName || '新建应用'">
      <el-form v-if="draft" label-position="top" class="editor-form">
        <el-form-item label="应用名">
          <el-input v-model="draft.appName" :disabled="!!editing" placeholder="app2-prod" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="draft.description" placeholder="可选描述" />
        </el-form-item>
        <div class="form-grid">
          <el-form-item label="格式">
            <el-select v-model="draft.format">
              <el-option label="TOML" value="toml" />
              <el-option label="YAML" value="yaml" />
              <el-option label="JSON" value="json" />
              <el-option label="Properties" value="properties" />
              <el-option label="纯文本" value="text" />
            </el-select>
          </el-form-item>
          <el-form-item label="敏感配置">
            <el-switch v-model="draft.sensitive" />
          </el-form-item>
        </div>
        <el-form-item label="配置内容">
          <el-input v-model="draft.content" type="textarea" :rows="18" resize="vertical" placeholder="在这里粘贴完整配置内容" />
        </el-form-item>
        <el-form-item label="变更说明">
          <el-input v-model="changeSummary" placeholder="描述本次变更" />
        </el-form-item>
        <div class="drawer-actions">
          <el-button @click="loadRevisions" :disabled="!editing">版本历史</el-button>
          <el-button type="danger" plain v-if="isAdmin && editing" @click="removeApp">删除</el-button>
          <el-button type="primary" :loading="saving" :disabled="draft.permission === 'masked'" @click="save">保存</el-button>
        </div>
      </el-form>
    </el-drawer>

    <el-drawer v-model="revisionOpen" size="45%" title="版本历史">
      <el-table :data="revisions">
        <el-table-column prop="version" label="版本" width="100" />
        <el-table-column prop="changeSummary" label="变更说明" min-width="180" />
        <el-table-column prop="createdAt" label="创建时间" min-width="170" />
        <el-table-column label="操作" width="110">
          <template #default="{ row }">
            <el-button text type="primary" @click="rollback(row.version)">回滚</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-drawer>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import { api, type AppConfig, type Revision } from '@/api/client'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const apps = ref<AppConfig[]>([])
const revisions = ref<Revision[]>([])
const query = ref('')
const page = ref(1)
const pageSize = 20
const total = ref(0)
const loading = ref(false)
const saving = ref(false)
const editorOpen = ref(false)
const revisionOpen = ref(false)
const editing = ref<AppConfig | null>(null)
const changeSummary = ref('')
const draft = ref<Partial<AppConfig> | null>(null)
const isAdmin = computed(() => auth.user?.role === 'admin')

onMounted(load)

async function load() {
  loading.value = true
  try {
    const result = await api.listApps(query.value, page.value, pageSize)
    if (result.list.length === 0 && result.total > 0 && page.value > 1) {
      page.value = Math.ceil(result.total / pageSize)
      return load()
    }
    apps.value = result.list
    total.value = result.total
  } finally {
    loading.value = false
  }
}

function search() {
  page.value = 1
  load()
}

function openCreate() {
  editing.value = null
  changeSummary.value = '初始化配置'
  draft.value = reactive({ appName: '', description: '', format: 'toml', content: '', sensitive: false, status: 'active' })
  editorOpen.value = true
}

function selectApp(row: AppConfig) {
  editing.value = row
  changeSummary.value = ''
  draft.value = reactive({ ...row })
  editorOpen.value = true
}

async function save() {
  if (!draft.value?.appName) return ElMessage.warning('应用名不能为空')
  saving.value = true
  try {
    if (editing.value) {
      await api.updateApp(editing.value.appName, { ...draft.value, changeSummary: changeSummary.value })
    } else {
      await api.createApp({ ...draft.value, appName: draft.value.appName, changeSummary: changeSummary.value })
    }
    ElMessage.success('已保存')
    editorOpen.value = false
    await load()
  } catch (err) {
    ElMessage.error(err instanceof Error ? err.message : '保存失败')
  } finally {
    saving.value = false
  }
}

async function removeApp() {
  if (!editing.value) return
  await ElMessageBox.confirm(`确认删除 ${editing.value.appName}？`, '删除确认', { type: 'warning' })
  await api.deleteApp(editing.value.appName)
  ElMessage.success('已删除')
  editorOpen.value = false
  await load()
}

async function loadRevisions() {
  if (!editing.value) return
  const page = await api.revisions(editing.value.appName)
  revisions.value = page.list
  revisionOpen.value = true
}

async function rollback(version: number) {
  if (!editing.value) return
  await api.rollback(editing.value.appName, version)
  ElMessage.success('已回滚')
  revisionOpen.value = false
  editorOpen.value = false
  await load()
}

function permissionText(permission?: string) {
  if (permission === 'full') return '完整'
  if (permission === 'masked') return '脱敏'
  return '无权限'
}
</script>
