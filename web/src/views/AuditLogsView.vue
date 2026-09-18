<template>
  <section class="page">
    <div class="page-header">
      <div>
        <h1>审计日志</h1>
        <p>记录配置、权限和 API 密钥的管理操作。</p>
      </div>
      <div class="toolbar">
        <el-input v-model="query" placeholder="搜索动作、资源或备注" clearable @change="search" />
      </div>
    </div>

    <el-table :data="logs" v-loading="loading" class="data-table">
      <el-table-column prop="action" label="动作" min-width="170" />
      <el-table-column label="资源类型" width="120">
        <template #default="{ row }">{{ resourceTypeText(row.resourceType) }}</template>
      </el-table-column>
      <el-table-column prop="resourceId" label="资源" min-width="160" />
      <el-table-column label="操作者" width="150">
        <template #default="{ row }">{{ actorText(row) }}</template>
      </el-table-column>
      <el-table-column prop="metadata" label="备注" min-width="220" show-overflow-tooltip />
      <el-table-column label="时间" min-width="170" :formatter="(row: AuditLog) => formatDateTime(row.createdAt)" />
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
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api, type AuditLog } from '@/api/client'
import { formatDateTime } from '@/utils/format'

const logs = ref<AuditLog[]>([])
const query = ref('')
const page = ref(1)
const pageSize = 20
const total = ref(0)
const loading = ref(false)

onMounted(load)

async function load() {
  loading.value = true
  try {
    const result = await api.listAuditLogs(query.value, page.value, pageSize)
    if (result.list.length === 0 && result.total > 0 && page.value > 1) {
      page.value = Math.ceil(result.total / pageSize)
      return load()
    }
    logs.value = result.list
    total.value = result.total
  } finally {
    loading.value = false
  }
}

function search() {
  page.value = 1
  load()
}

function actorText(row: AuditLog) {
  if (!row.actorId) return row.actorType
  return `${row.actorType} #${row.actorId}`
}

function resourceTypeText(type: string) {
  if (type === 'app') return '应用'
  if (type === 'user') return '用户'
  if (type === 'api_key') return 'API 密钥'
  return type
}
</script>
