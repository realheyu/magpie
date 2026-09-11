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
      <el-table-column prop="keyPreview" label="密钥预览" min-width="160" />
      <el-table-column label="可读应用" min-width="220">
        <template #default="{ row }">{{ row.appNames.join(', ') }}</template>
      </el-table-column>
      <el-table-column prop="lastUsedAt" label="最后使用" min-width="170" />
      <el-table-column label="操作" width="110">
        <template #default="{ row }">
          <el-button text type="danger" @click="remove(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="createDialog" title="新建 API 密钥" width="560px">
      <el-form label-position="top">
        <el-form-item label="名称">
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item label="可读应用">
          <el-select v-model="form.appNames" multiple filterable allow-create default-first-option placeholder="app2-prod">
            <el-option v-for="app in apps" :key="app.appName" :label="app.appName" :value="app.appName" />
          </el-select>
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
  </section>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import { api, type APIKeyItem, type AppConfig } from '@/api/client'

const keys = ref<APIKeyItem[]>([])
const apps = ref<AppConfig[]>([])
const loading = ref(false)
const createDialog = ref(false)
const plainKey = ref('')
const form = reactive({ name: '', appNames: [] as string[] })

onMounted(load)

async function load() {
  loading.value = true
  try {
    const [keyPage, appPage] = await Promise.all([api.listAPIKeys(), api.listApps('', 1, 100)])
    keys.value = keyPage.list
    apps.value = appPage.list
  } finally {
    loading.value = false
  }
}

function openCreate() {
  Object.assign(form, { name: '', appNames: [] })
  plainKey.value = ''
  createDialog.value = true
}

async function create() {
  try {
    const resp = await api.createAPIKey(form)
    plainKey.value = resp.apiKey
    await load()
  } catch (err) {
    ElMessage.error(err instanceof Error ? err.message : '创建失败')
  }
}

async function remove(row: APIKeyItem) {
  await ElMessageBox.confirm(`确认删除 ${row.name}？`, '删除确认', { type: 'warning' })
  await api.deleteAPIKey(row.id)
  ElMessage.success('已删除')
  await load()
}
</script>
