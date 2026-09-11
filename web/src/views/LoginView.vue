<template>
  <div class="login-page">
    <el-form class="login-panel" :model="form" @submit.prevent="submit">
      <h1>Magpie</h1>
      <p>登录后管理应用配置。</p>
      <el-form-item>
        <el-input v-model="form.username" size="large" placeholder="用户名" autocomplete="username" />
      </el-form-item>
      <el-form-item>
        <el-input v-model="form.password" size="large" placeholder="密码" type="password" autocomplete="current-password" show-password />
      </el-form-item>
      <el-button type="primary" size="large" native-type="submit" :loading="loading" class="full-button">登录</el-button>
    </el-form>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const auth = useAuthStore()
const loading = ref(false)
const form = reactive({ username: 'admin', password: '' })

async function submit() {
  loading.value = true
  try {
    await auth.login(form.username, form.password)
    router.push('/apps')
  } catch (err) {
    ElMessage.error(err instanceof Error ? err.message : '登录失败')
  } finally {
    loading.value = false
  }
}
</script>
