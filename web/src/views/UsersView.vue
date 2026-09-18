<template>
  <section class="page">
    <div class="page-header">
      <div>
        <h1>用户管理</h1>
        <p>管理员可以给用户分配应用级可见性。</p>
      </div>
      <div class="toolbar">
        <el-input v-model="query" placeholder="搜索用户名或显示名" clearable @change="search" />
        <el-button type="primary" :icon="Plus" @click="openCreate">新建用户</el-button>
      </div>
    </div>
    <el-table :data="users" v-loading="loading" class="data-table">
      <el-table-column prop="username" label="用户名" min-width="160" />
      <el-table-column prop="displayName" label="显示名" min-width="160" />
      <el-table-column label="角色" width="110">
        <template #default="{ row }">{{ roleText(row.role) }}</template>
      </el-table-column>
      <el-table-column label="状态" width="120">
        <template #default="{ row }">{{ statusText(row.status) }}</template>
      </el-table-column>
      <el-table-column label="最后登录" min-width="170" :formatter="(row: User) => formatDateTime(row.lastLoginAt)" />
      <el-table-column label="创建时间" min-width="170" :formatter="(row: User) => formatDateTime(row.createdAt)" />
      <el-table-column label="更新时间" min-width="170" :formatter="(row: User) => formatDateTime(row.updatedAt)" />
      <el-table-column label="操作" width="220">
        <template #default="{ row }">
          <el-button text type="primary" @click="editUser(row)">编辑</el-button>
          <el-button text type="primary" @click="editPermissions(row)">权限</el-button>
          <el-button text type="danger" :disabled="row.id === auth.user?.id" @click="deleteUser(row)">删除</el-button>
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

    <el-dialog v-model="userDialog" :title="selectedUser ? '编辑用户' : '新建用户'" width="520px">
      <el-form :model="userForm" label-position="top">
        <el-form-item label="用户名">
          <el-input v-model="userForm.username" :disabled="!!selectedUser" />
        </el-form-item>
        <el-form-item label="显示名">
          <el-input v-model="userForm.displayName" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input v-model="userForm.password" type="password" show-password :placeholder="selectedUser ? '留空则不修改当前密码' : ''" />
        </el-form-item>
        <div class="form-grid">
          <el-form-item label="角色">
            <el-select v-model="userForm.role">
              <el-option label="管理员" value="admin" />
              <el-option label="普通用户" value="user" />
            </el-select>
          </el-form-item>
          <el-form-item label="状态">
            <el-select v-model="userForm.status">
              <el-option label="启用" value="active" />
              <el-option label="禁用" value="disabled" />
            </el-select>
          </el-form-item>
        </div>
      </el-form>
      <template #footer>
        <el-button @click="userDialog = false">取消</el-button>
        <el-button type="primary" @click="saveUser">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="permissionDialog" title="应用权限" width="620px">
      <div class="permission-list">
        <div v-for="(entry, index) in permissionForm" :key="index" class="permission-row">
          <template v-if="editingIndex === index">
            <el-select v-model="entry.appName" filterable placeholder="请选择应用">
              <el-option v-for="app in availableApps(entry.appName)" :key="app.appName" :label="app.appName" :value="app.appName" />
            </el-select>
            <el-select v-model="entry.permission">
              <el-option label="完整" value="full" />
              <el-option label="脱敏" value="masked" />
            </el-select>
            <div class="row-actions">
              <el-button text type="primary" @click="confirmEntry(index)">确定</el-button>
              <el-button text @click="cancelEntry(index)">取消</el-button>
            </div>
          </template>
          <template v-else>
            <span class="permission-app">{{ entry.appName }}</span>
            <el-tag :type="entry.permission === 'full' ? 'success' : 'info'" size="small">{{ entry.permission === 'full' ? '完整' : '脱敏' }}</el-tag>
            <div class="row-actions">
              <el-button text type="primary" :disabled="editingIndex >= 0" @click="startEdit(index)">编辑</el-button>
              <el-button text type="danger" :disabled="editingIndex >= 0" @click="removeEntry(index)">移除</el-button>
            </div>
          </template>
        </div>
      </div>
      <el-button :icon="Plus" :disabled="editingIndex >= 0" @click="addEntry">添加权限</el-button>
      <template #footer>
        <el-button @click="permissionDialog = false">取消</el-button>
        <el-button type="primary" :disabled="editingIndex >= 0" @click="savePermissions">保存</el-button>
      </template>
    </el-dialog>
  </section>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import { api, type AppOption, type User } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import { formatDateTime } from '@/utils/format'

const auth = useAuthStore()
const users = ref<User[]>([])
const apps = ref<AppOption[]>([])
const query = ref('')
const page = ref(1)
const pageSize = 20
const total = ref(0)
const loading = ref(false)
const userDialog = ref(false)
const permissionDialog = ref(false)
const selectedUser = ref<User | null>(null)
const userForm = reactive({ username: '', displayName: '', password: '', role: 'user', status: 'active' })
const permissionForm = ref<{ appName: string; permission: string }[]>([])
// 正在编辑的权限行下标，-1 表示没有；editBackup 记录编辑前的值用于取消还原
const editingIndex = ref(-1)
const editBackup = ref<{ appName: string; permission: string } | null>(null)

onMounted(load)

async function load() {
  loading.value = true
  try {
    const [userPage, appOptions] = await Promise.all([api.listUsers(query.value, page.value, pageSize), api.listAppOptions()])
    if (userPage.list.length === 0 && userPage.total > 0 && page.value > 1) {
      page.value = Math.ceil(userPage.total / pageSize)
      return load()
    }
    users.value = userPage.list
    total.value = userPage.total
    apps.value = appOptions
  } finally {
    loading.value = false
  }
}

function search() {
  page.value = 1
  load()
}

function openCreate() {
  selectedUser.value = null
  Object.assign(userForm, { username: '', displayName: '', password: '', role: 'user', status: 'active' })
  userDialog.value = true
}

function editUser(user: User) {
  selectedUser.value = user
  Object.assign(userForm, { username: user.username, displayName: user.displayName, password: '', role: user.role, status: user.status })
  userDialog.value = true
}

async function saveUser() {
  try {
    if (selectedUser.value) {
      await api.updateUser(selectedUser.value.id, userForm)
    } else {
      await api.createUser(userForm)
    }
    ElMessage.success('已保存')
    userDialog.value = false
    await load()
  } catch (err) {
    ElMessage.error(err instanceof Error ? err.message : '保存失败')
  }
}

async function deleteUser(user: User) {
  try {
    await ElMessageBox.confirm(`确认删除用户 ${user.username}？`, '删除确认', { type: 'warning' })
    await api.deleteUser(user.id)
    ElMessage.success('已删除')
    await load()
  } catch (err) {
    if (!isDialogCancel(err)) ElMessage.error(err instanceof Error ? err.message : '删除失败')
  }
}

async function editPermissions(user: User) {
  try {
    selectedUser.value = user
    permissionForm.value = (await api.getPermissions(user.id)).permissions
    editingIndex.value = -1
    editBackup.value = null
    permissionDialog.value = true
  } catch (err) {
    ElMessage.error(err instanceof Error ? err.message : '加载权限失败')
  }
}

function addEntry() {
  if (editingIndex.value >= 0) return
  permissionForm.value.push({ appName: '', permission: 'masked' })
  editingIndex.value = permissionForm.value.length - 1
}

function startEdit(index: number) {
  if (editingIndex.value >= 0) return
  editingIndex.value = index
  editBackup.value = { ...permissionForm.value[index] }
}

function cancelEntry(index: number) {
  if (editBackup.value) {
    permissionForm.value[index] = { ...editBackup.value }
  } else {
    permissionForm.value.splice(index, 1)
  }
  editingIndex.value = -1
  editBackup.value = null
}

function confirmEntry(index: number) {
  const entry = permissionForm.value[index]
  if (!entry.appName) {
    ElMessage.warning('请选择应用')
    return
  }
  if (permissionForm.value.some((item, i) => i !== index && item.appName === entry.appName)) {
    ElMessage.warning('该应用已在权限列表中')
    return
  }
  editingIndex.value = -1
  editBackup.value = null
}

function removeEntry(index: number) {
  permissionForm.value.splice(index, 1)
  editingIndex.value = -1
  editBackup.value = null
}

async function savePermissions() {
  if (!selectedUser.value) return
  if (editingIndex.value >= 0) {
    ElMessage.warning('请先完成当前行的编辑')
    return
  }
  try {
    await api.setPermissions(selectedUser.value.id, permissionForm.value)
    ElMessage.success('权限已保存')
    permissionDialog.value = false
  } catch (err) {
    ElMessage.error(err instanceof Error ? err.message : '保存权限失败')
  }
}

function roleText(role: string) {
  return role === 'admin' ? '管理员' : '普通用户'
}

function statusText(status: string) {
  return status === 'disabled' ? '禁用' : '启用'
}

function availableApps(currentAppName: string) {
  const selected = new Set(permissionForm.value.map((entry) => entry.appName).filter(Boolean))
  return apps.value.filter((app) => app.appName === currentAppName || !selected.has(app.appName))
}

function isDialogCancel(err: unknown) {
  return err === 'cancel' || err === 'close'
}
</script>
