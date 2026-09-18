<template>
  <router-view v-if="isLoginRoute" />
  <el-container v-else class="shell">
    <el-aside width="240px" class="sidebar">
      <div class="brand">
        <div class="brand-mark">M</div>
        <div>
          <strong>Magpie</strong>
          <span>配置中心</span>
        </div>
      </div>
      <el-menu router :default-active="route.path" class="nav">
        <el-menu-item index="/apps">
          <el-icon><Document /></el-icon>
          <span>应用配置</span>
        </el-menu-item>
        <el-menu-item index="/users" v-if="auth.user?.role === 'admin'">
          <el-icon><User /></el-icon>
          <span>用户管理</span>
        </el-menu-item>
        <el-menu-item index="/api-keys" v-if="auth.user?.role === 'admin'">
          <el-icon><Key /></el-icon>
          <span>API 密钥</span>
        </el-menu-item>
        <el-menu-item index="/audit-logs" v-if="auth.user?.role === 'admin'">
          <el-icon><Tickets /></el-icon>
          <span>审计日志</span>
        </el-menu-item>
      </el-menu>
    </el-aside>
    <el-container>
      <el-header class="topbar">
        <span>{{ auth.user?.displayName || auth.user?.username }}</span>
        <el-button :icon="SwitchButton" text @click="logout">退出登录</el-button>
      </el-header>
      <el-main class="content">
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Document, Key, SwitchButton, Tickets, User } from '@element-plus/icons-vue'
import { useAuthStore } from './stores/auth'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const isLoginRoute = computed(() => route.path === '/login')

onMounted(() => {
  if (auth.token && !auth.user) auth.loadMe().catch(() => auth.clear())
})

async function logout() {
  await auth.logout()
  router.push('/login')
}
</script>
