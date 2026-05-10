<template>
  <div class="wrap">
    <el-card class="card">
      <h2>模链云 · 管理后台</h2>
      <el-form @submit.prevent="onSubmit">
        <el-form-item label="邮箱">
          <el-input v-model="email" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input v-model="password" type="password" />
        </el-form-item>
        <el-button type="primary" native-type="submit" :loading="loading">登录</el-button>
      </el-form>
      <p class="hint">需账号 role=admin（注册时使用 bootstrap_admin_emails 中的邮箱）。</p>
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
    const d = data.data as { access_token?: string; role?: string }
    if (d.access_token) localStorage.setItem('mlk_admin_token', d.access_token)
    router.push('/admin')
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
  background: #0b1220;
}
.card {
  width: 420px;
}
.hint {
  font-size: 12px;
  color: #94a3b8;
  margin-top: 12px;
}
</style>
