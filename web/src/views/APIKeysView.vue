<template>
  <section class="page">
    <div class="page-header">
      <div>
        <h1>API 密钥</h1>
        <p>密钥可以读取一个或多个应用配置。</p>
      </div>
      <el-button type="primary" :icon="Plus" @click="openCreate">新建密钥</el-button>
    </div>

    <el-table :data="keys" v-loading="loading" class="data-table">
      <el-table-column prop="name" label="名称" min-width="160" />
      <el-table-column prop="keyPreview" label="密钥预览" min-width="150" />
      <el-table-column label="状态" width="110">
        <template #default="{ row }">
          <el-tag :type="row.status === 'active' ? 'success' : 'info'">{{ statusText(row.status) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="可读应用" min-width="240">
        <template #default="{ row }">
          <div class="tag-list">
            <el-tag v-for="appName in row.appNames" :key="appName" size="small">{{ appName }}</el-tag>
          </div>
        </template>
      </el-table-column>
      <el-table-column label="过期时间" min-width="170" :formatter="(row: APIKeyItem) => formatDateTime(row.expiresAt)" />
      <el-table-column label="最后使用" min-width="170" :formatter="(row: APIKeyItem) => formatDateTime(row.lastUsedAt)" />
      <el-table-column label="创建时间" min-width="170" :formatter="(row: APIKeyItem) => formatDateTime(row.createdAt)" />
      <el-table-column label="更新时间" min-width="170" :formatter="(row: APIKeyItem) => formatDateTime(row.updatedAt)" />
      <el-table-column label="操作" width="240" fixed="right">
        <template #default="{ row }">
          <el-button text type="primary" @click="openAppEditor(row)">编辑应用</el-button>
          <el-button text type="primary" @click="toggleStatus(row)">{{ row.status === 'active' ? '禁用' : '启用' }}</el-button>
          <el-button text type="danger" @click="remove(row)">删除</el-button>
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

    <el-dialog v-model="createDialog" title="新建 API 密钥" width="560px">
      <el-form label-position="top">
        <el-form-item label="名称">
          <el-input v-model="form.name" placeholder="生产读取密钥" />
        </el-form-item>
        <el-form-item label="可读应用">
          <el-select v-model="form.appNames" multiple filterable placeholder="请选择应用">
            <el-option v-for="app in apps" :key="app.appName" :label="app.appName" :value="app.appName" />
          </el-select>
        </el-form-item>
        <el-form-item label="过期时间">
          <el-date-picker
            v-model="form.expiresAt"
            type="datetime"
            clearable
            placeholder="不设置则长期有效"
            value-format="YYYY-MM-DDTHH:mm:ssZ"
          />
        </el-form-item>
      </el-form>
      <el-alert v-if="plainKey" type="success" show-icon :closable="false" title="请立即复制该密钥，关闭后将不再展示。">
        <template #default>
          <code class="plain-key">{{ plainKey }}</code>
        </template>
      </el-alert>
      <template #footer>
        <el-button @click="createDialog = false">关闭</el-button>
        <el-button type="primary" :disabled="!!plainKey" @click="create">创建</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="appsDialog" :title="selectedKey ? `编辑 ${selectedKey.name} 的可读应用` : '编辑可读应用'" width="560px">
      <el-form label-position="top">
        <el-form-item label="可读应用">
          <el-select v-model="appForm.appNames" multiple filterable placeholder="请选择应用">
            <el-option v-for="app in apps" :key="app.appName" :label="app.appName" :value="app.appName" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="appsDialog = false">取消</el-button>
        <el-button type="primary" @click="saveKeyApps">保存</el-button>
      </template>
    </el-dialog>
  </section>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import { api, type APIKeyItem, type AppOption } from '@/api/client'
import { formatDateTime } from '@/utils/format'

const keys = ref<APIKeyItem[]>([])
const apps = ref<AppOption[]>([])
const page = ref(1)
const pageSize = 20
const total = ref(0)
const loading = ref(false)
const createDialog = ref(false)
const appsDialog = ref(false)
const plainKey = ref('')
const selectedKey = ref<APIKeyItem | null>(null)
const form = reactive<{ name: string; appNames: string[]; expiresAt?: string }>({ name: '', appNames: [], expiresAt: undefined })
const appForm = reactive({ appNames: [] as string[] })

onMounted(load)

async function load() {
  loading.value = true
  try {
    const [keyPage, appOptions] = await Promise.all([api.listAPIKeys(page.value, pageSize), api.listAppOptions()])
    if (keyPage.list.length === 0 && keyPage.total > 0 && page.value > 1) {
      page.value = Math.ceil(keyPage.total / pageSize)
      return load()
    }
    keys.value = keyPage.list
    total.value = keyPage.total
    apps.value = appOptions
  } finally {
    loading.value = false
  }
}

function openCreate() {
  Object.assign(form, { name: '', appNames: [], expiresAt: undefined })
  plainKey.value = ''
  createDialog.value = true
}

async function create() {
  if (!form.name.trim()) return ElMessage.warning('名称不能为空')
  if (form.appNames.length === 0) return ElMessage.warning('请选择至少一个应用')
  try {
    const resp = await api.createAPIKey({ name: form.name, appNames: form.appNames, expiresAt: form.expiresAt || undefined })
    plainKey.value = resp.apiKey
    await load()
  } catch (err) {
    ElMessage.error(err instanceof Error ? err.message : '创建失败')
  }
}

function openAppEditor(row: APIKeyItem) {
  selectedKey.value = row
  appForm.appNames = [...row.appNames]
  appsDialog.value = true
}

async function saveKeyApps() {
  if (!selectedKey.value) return
  if (appForm.appNames.length === 0) return ElMessage.warning('请选择至少一个应用')
  try {
    await api.updateAPIKeyApps(selectedKey.value.id, appForm.appNames)
    ElMessage.success('可读应用已保存')
    appsDialog.value = false
    await load()
  } catch (err) {
    ElMessage.error(err instanceof Error ? err.message : '保存可读应用失败')
  }
}

async function toggleStatus(row: APIKeyItem) {
  try {
    const nextStatus = row.status === 'active' ? 'disabled' : 'active'
    if (nextStatus === 'disabled') {
      await ElMessageBox.confirm(`确认禁用 ${row.name}？禁用后该密钥不能读取配置。`, '禁用确认', { type: 'warning' })
    }
    await api.updateAPIKeyStatus(row.id, nextStatus)
    ElMessage.success(nextStatus === 'active' ? '已启用' : '已禁用')
    await load()
  } catch (err) {
    if (!isDialogCancel(err)) ElMessage.error(err instanceof Error ? err.message : '更新状态失败')
  }
}

async function remove(row: APIKeyItem) {
  try {
    await ElMessageBox.confirm(`确认删除 ${row.name}？`, '删除确认', { type: 'warning' })
    await api.deleteAPIKey(row.id)
    ElMessage.success('已删除')
    await load()
  } catch (err) {
    if (!isDialogCancel(err)) ElMessage.error(err instanceof Error ? err.message : '删除失败')
  }
}

function statusText(status?: string) {
  return status === 'disabled' ? '禁用' : '启用'
}

function isDialogCancel(err: unknown) {
  return err === 'cancel' || err === 'close'
}
</script>
