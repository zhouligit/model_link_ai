<template>
  <div class="wrap">
    <el-card class="card">
      <h2>模链云 · 登录</h2>
      <el-form @submit.prevent="onSubmit">
        <el-form-item label="邮箱">
          <el-input v-model="email" autocomplete="username" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input v-model="password" type="password" autocomplete="current-password" />
        </el-form-item>
        <el-button type="primary" native-type="submit" :loading="loading">登录</el-button>
      </el-form>
      <p class="hint">测试账号：先在后端注册；bootstrap admin 邮箱见 configs config.example.yaml。</p>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import axios from 'axios'
import { ElMessage } from 'element-plus'
import type { ApiEnvelope } from '../api/client'

const router = useRouter()
const email = ref('')
const password = ref('')
const loading = ref(false)

async function onSubmit() {
  loading.value = true
  try {
    const { data } = await axios.post<ApiEnvelope<Record<string, unknown>>>('/mlk/platform/v1/auth/login', {
      email: email.value,
      password: password.value,
    })
    if (data.code !== 0 || !data.data) {
      ElMessage.error(data.message || '登录失败')
      return
    }
    const d = data.data as { access_token?: string; refresh_token?: string }
    if (d.access_token) localStorage.setItem('mlk_access_token', d.access_token)
    if (d.refresh_token) localStorage.setItem('mlk_refresh_token', d.refresh_token)
    router.push('/app')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.wrap {
  min-height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(160deg, #0f172a, #1e293b);
}
.card {
  width: 400px;
}
.hint {
  font-size: 12px;
  color: #64748b;
  margin-top: 12px;
}
</style>
