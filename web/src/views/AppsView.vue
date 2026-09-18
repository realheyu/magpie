<template>
  <section class="page">
    <div class="page-header">
      <div>
        <h1>应用配置</h1>
      </div>
      <div class="toolbar">
        <el-input v-model="query" placeholder="搜索应用" clearable @change="search" />
        <el-button :icon="RefreshLeft" v-if="isAdmin" @click="openRestore">恢复应用</el-button>
        <el-button type="primary" :icon="Plus" v-if="isAdmin" @click="openCreate">新建应用</el-button>
      </div>
    </div>

    <el-table :data="apps" v-loading="loading" class="data-table">
      <el-table-column prop="appName" label="应用" min-width="180" />
      <el-table-column prop="description" label="描述" min-width="180" show-overflow-tooltip />
      <el-table-column prop="format" label="格式" width="110" />
      <el-table-column prop="version" label="版本" width="110" />
      <el-table-column label="状态" width="110">
        <template #default="{ row }">
          <el-tag :type="row.status === 'active' ? 'success' : 'info'">{{ statusText(row.status) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="权限" width="130">
        <template #default="{ row }">{{ permissionText(row.permission) }}</template>
      </el-table-column>
      <el-table-column label="敏感配置" width="120">
        <template #default="{ row }">
          <el-tag :type="row.sensitive ? 'warning' : 'info'">{{ row.sensitive ? '是' : '否' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="创建时间" min-width="170" :formatter="(row: AppConfig) => formatDateTime(row.createdAt)" />
      <el-table-column label="更新时间" min-width="170" :formatter="(row: AppConfig) => formatDateTime(row.updatedAt)" />
      <el-table-column label="操作" width="170" fixed="right">
        <template #default="{ row }">
          <el-button text type="primary" @click="openDetail(row)">详情</el-button>
          <el-button text type="primary" :disabled="row.permission === 'masked' && !isAdmin" @click="openEdit(row)">编辑</el-button>
        </template>
      </el-table-column>
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

    <el-drawer v-model="editorOpen" size="58%" :title="(editing?.appName || '新建应用') + (readOnly ? '（只读）' : '')">
      <el-form v-if="draft" label-position="top" class="editor-form">
        <el-form-item label="应用名">
          <el-input v-model="draft.appName" :disabled="!!editing" placeholder="app2-prod" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="draft.description" :disabled="readOnly" placeholder="可选描述" />
        </el-form-item>
        <div class="form-grid">
          <el-form-item label="格式">
            <el-select v-model="draft.format" :disabled="readOnly">
              <el-option label="TOML" value="toml" />
              <el-option label="YAML" value="yaml" />
              <el-option label="JSON" value="json" />
              <el-option label="Properties" value="properties" />
              <el-option label="纯文本" value="text" />
            </el-select>
          </el-form-item>
          <el-form-item label="敏感配置">
            <el-switch v-model="draft.sensitive" :disabled="readOnly" />
          </el-form-item>
          <el-form-item label="状态">
            <el-select v-model="draft.status" :disabled="readOnly || !isAdmin">
              <el-option label="启用" value="active" />
              <el-option label="禁用" value="disabled" />
            </el-select>
          </el-form-item>
        </div>
        <el-form-item label="配置内容">
          <CodeEditor v-model="draft.content" :language="draft.format" :disabled="readOnly" />
        </el-form-item>
        <el-descriptions v-if="editing" :column="3" border class="revision-meta">
          <el-descriptions-item label="版本">{{ editing.version }}</el-descriptions-item>
          <el-descriptions-item label="创建时间">{{ formatDateTime(editing.createdAt) }}</el-descriptions-item>
          <el-descriptions-item label="更新时间">{{ formatDateTime(editing.updatedAt) }}</el-descriptions-item>
        </el-descriptions>
        <el-form-item label="变更说明">
          <el-input v-model="changeSummary" :disabled="readOnly" placeholder="描述本次变更" />
        </el-form-item>
        <div class="drawer-actions">
          <el-button @click="loadRevisions" :disabled="!editing">版本历史</el-button>
          <el-button type="danger" plain v-if="isAdmin && editing && !readOnly" @click="removeApp">删除</el-button>
          <el-button v-if="!readOnly" type="primary" :loading="saving" :disabled="draft.permission === 'masked'" @click="save">保存</el-button>
        </div>
      </el-form>
    </el-drawer>

    <el-drawer v-model="revisionOpen" size="45%" title="版本历史">
      <el-table :data="revisions">
        <el-table-column prop="version" label="版本" width="100" />
        <el-table-column prop="format" label="格式" width="100" />
        <el-table-column prop="changeSummary" label="变更说明" min-width="180" />
        <el-table-column prop="createdByUserId" label="创建人ID" width="110" />
        <el-table-column label="创建时间" min-width="170" :formatter="(row: Revision) => formatDateTime(row.createdAt)" />
        <el-table-column label="操作" width="150">
          <template #default="{ row }">
            <el-button text type="primary" @click="viewRevision(row)">查看</el-button>
            <el-button text type="primary" :disabled="readOnly" @click="rollback(row.version)">回滚</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-drawer>

    <el-dialog v-model="revisionDetailOpen" :title="selectedRevision ? `版本 ${selectedRevision.version}` : '版本内容'" width="760px">
      <el-descriptions v-if="selectedRevision" :column="2" border class="revision-meta">
        <el-descriptions-item label="应用">{{ selectedRevision.appName }}</el-descriptions-item>
        <el-descriptions-item label="格式">{{ selectedRevision.format }}</el-descriptions-item>
        <el-descriptions-item label="敏感配置">{{ selectedRevision.sensitive ? '是' : '否' }}</el-descriptions-item>
        <el-descriptions-item label="创建时间">{{ formatDateTime(selectedRevision.createdAt) }}</el-descriptions-item>
        <el-descriptions-item label="变更说明" :span="2">{{ selectedRevision.changeSummary || '-' }}</el-descriptions-item>
      </el-descriptions>
      <CodeEditor v-if="selectedRevision" v-model="selectedRevision.content" :language="selectedRevision.format" disabled />
      <template #footer>
        <el-button @click="revisionDetailOpen = false">关闭</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="restoreDialog" title="从历史版本恢复应用" width="760px">
      <el-form label-position="top">
        <el-form-item label="应用名">
          <div class="inline-form-row">
            <el-input v-model="restoreForm.appName" placeholder="例如 app2-prod" />
            <el-button :loading="restoreLoading" @click="loadRestoreRevisions">加载版本</el-button>
          </div>
        </el-form-item>
      </el-form>
      <el-table :data="restoreRevisions" v-loading="restoreLoading" class="data-table">
        <el-table-column prop="version" label="版本" width="100" />
        <el-table-column prop="format" label="格式" width="100" />
        <el-table-column prop="changeSummary" label="变更说明" min-width="180" />
        <el-table-column label="创建时间" min-width="170" :formatter="(row: Revision) => formatDateTime(row.createdAt)" />
        <el-table-column label="操作" width="110">
          <template #default="{ row }">
            <el-button text type="primary" @click="restoreFromRevision(row.version)">恢复</el-button>
          </template>
        </el-table-column>
      </el-table>
      <template #footer>
        <el-button @click="restoreDialog = false">关闭</el-button>
      </template>
    </el-dialog>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, RefreshLeft } from '@element-plus/icons-vue'
import { api, type AppConfig, type Revision } from '@/api/client'
import CodeEditor from '@/components/CodeEditor.vue'
import { formatDateTime } from '@/utils/format'
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
const revisionDetailOpen = ref(false)
const restoreDialog = ref(false)
const restoreLoading = ref(false)
const readOnly = ref(false)
const editing = ref<AppConfig | null>(null)
const selectedRevision = ref<Revision | null>(null)
const restoreRevisions = ref<Revision[]>([])
const changeSummary = ref('')
const draft = ref<Partial<AppConfig> | null>(null)
const restoreForm = reactive({ appName: '' })
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
  readOnly.value = false
  changeSummary.value = '初始化配置'
  draft.value = reactive({ appName: '', description: '', format: 'toml', content: '', sensitive: false, status: 'active' })
  editorOpen.value = true
}

function openRestore() {
  restoreForm.appName = ''
  restoreRevisions.value = []
  restoreDialog.value = true
}

async function openDetail(row: AppConfig) {
  const app = await loadFullApp(row)
  if (!app) return
  editing.value = app
  readOnly.value = true
  changeSummary.value = ''
  draft.value = reactive({ ...app })
  editorOpen.value = true
}

async function openEdit(row: AppConfig) {
  const app = await loadFullApp(row)
  if (!app) return
  editing.value = app
  readOnly.value = false
  changeSummary.value = ''
  draft.value = reactive({ ...app })
  editorOpen.value = true
}

async function loadFullApp(row: AppConfig) {
  try {
    return await api.getApp(row.appName)
  } catch (err) {
    ElMessage.error(err instanceof Error ? err.message : '加载应用详情失败')
    return null
  }
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
  try {
    await ElMessageBox.confirm(`确认删除 ${editing.value.appName}？删除后可从历史版本恢复。`, '删除确认', { type: 'warning' })
    await api.deleteApp(editing.value.appName)
    ElMessage.success('已删除')
    editorOpen.value = false
    await load()
  } catch (err) {
    if (!isDialogCancel(err)) ElMessage.error(err instanceof Error ? err.message : '删除失败')
  }
}

async function loadRevisions() {
  if (!editing.value) return
  const page = await api.revisions(editing.value.appName)
  revisions.value = page.list
  revisionOpen.value = true
}

async function loadRestoreRevisions() {
  const appName = restoreForm.appName.trim()
  if (!appName) return ElMessage.warning('应用名不能为空')
  restoreLoading.value = true
  try {
    const page = await api.revisions(appName)
    restoreRevisions.value = page.list
    if (page.list.length === 0) ElMessage.info('没有可恢复的历史版本')
  } catch (err) {
    ElMessage.error(err instanceof Error ? err.message : '加载历史版本失败')
  } finally {
    restoreLoading.value = false
  }
}

async function restoreFromRevision(version: number) {
  const appName = restoreForm.appName.trim()
  if (!appName) return
  try {
    await ElMessageBox.confirm(`确认从 ${appName} 的版本 ${version} 恢复应用？`, '恢复确认', { type: 'warning' })
    await api.restoreApp(appName, version)
    ElMessage.success('已恢复')
    restoreDialog.value = false
    await load()
  } catch (err) {
    if (!isDialogCancel(err)) ElMessage.error(err instanceof Error ? err.message : '恢复失败')
  }
}

async function rollback(version: number) {
  if (!editing.value) return
  try {
    await ElMessageBox.confirm(`确认回滚到版本 ${version}？`, '回滚确认', { type: 'warning' })
    await api.rollback(editing.value.appName, version)
    ElMessage.success('已回滚')
    revisionOpen.value = false
    editorOpen.value = false
    await load()
  } catch (err) {
    if (!isDialogCancel(err)) ElMessage.error(err instanceof Error ? err.message : '回滚失败')
  }
}

function viewRevision(revision: Revision) {
  selectedRevision.value = { ...revision }
  revisionDetailOpen.value = true
}

function permissionText(permission?: string) {
  if (permission === 'full') return '完整'
  if (permission === 'masked') return '脱敏'
  return '无权限'
}

function statusText(status?: string) {
  return status === 'disabled' ? '禁用' : '启用'
}

function isDialogCancel(err: unknown) {
  return err === 'cancel' || err === 'close'
}
</script>
