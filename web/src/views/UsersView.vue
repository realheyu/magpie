<template>
  <section class="page">
    <div class="page-header">
      <div>
        <h1>用户管理</h1>
        <p>管理员可以给用户分配应用级可见性。</p>
      </div>
      <el-button type="primary" :icon="Plus" @click="openCreate">新建用户</el-button>
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
      <el-table-column label="操作" width="180">
        <template #default="{ row }">
          <el-button text type="primary" @click="editUser(row)">编辑</el-button>
          <el-button text type="primary" @click="editPermissions(row)">权限</el-button>
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
          <el-select v-model="entry.appName" filterable placeholder="请选择应用">
            <el-option v-for="app in availableApps(entry.appName)" :key="app.appName" :label="app.appName" :value="app.appName" />
          </el-select>
          <el-select v-model="entry.permission">
            <el-option label="完整" value="full" />
            <el-option label="脱敏" value="masked" />
          </el-select>
          <el-button text type="danger" @click="permissionForm = permissionForm.filter((item) => item !== entry)">移除</el-button>
        </div>
      </div>
      <el-button :icon="Plus" @click="permissionForm.push({ appName: '', permission: 'masked' })">添加权限</el-button>
      <template #footer>
        <el-button @click="permissionDialog = false">取消</el-button>
        <el-button type="primary" @click="savePermissions">保存</el-button>
      </template>
    </el-dialog>
  </section>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import { api, type AppConfig, type User } from '@/api/client'

const users = ref<User[]>([])
const apps = ref<AppConfig[]>([])
const page = ref(1)
const pageSize = 20
const total = ref(0)
const loading = ref(false)
const userDialog = ref(false)
const permissionDialog = ref(false)
const selectedUser = ref<User | null>(null)
const userForm = reactive({ username: '', displayName: '', password: '', role: 'user', status: 'active' })
const permissionForm = ref<{ appName: string; permission: string }[]>([])

onMounted(load)

async function load() {
  loading.value = true
  try {
    const [userPage, appPage] = await Promise.all([api.listUsers('', page.value, pageSize), api.listApps('', 1, 100)])
    if (userPage.list.length === 0 && userPage.total > 0 && page.value > 1) {
      page.value = Math.ceil(userPage.total / pageSize)
      return load()
    }
    users.value = userPage.list
    total.value = userPage.total
    apps.value = appPage.list
  } finally {
    loading.value = false
  }
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

async function editPermissions(user: User) {
  selectedUser.value = user
  permissionForm.value = (await api.getPermissions(user.id)).permissions
  permissionDialog.value = true
}

async function savePermissions() {
  if (!selectedUser.value) return
  await api.setPermissions(selectedUser.value.id, permissionForm.value.filter((entry) => entry.appName))
  ElMessage.success('权限已保存')
  permissionDialog.value = false
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
</script>
