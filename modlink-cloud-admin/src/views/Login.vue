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
import { ElMessage } from 'element-plus'
import api, { type ApiEnvelope } from '../api/client'

const router = useRouter()
const email = ref('')
const password = ref('')
const loading = ref(false)

async function onSubmit() {
  loading.value = true
  try {
    const { data } = await api.post<ApiEnvelope<Record<string, unknown>>>('/auth/login', {
      email: email.value.trim().toLowerCase(),
      password: password.value,
    })
    if (!data || typeof data !== 'object' || typeof data.code !== 'number') {
      ElMessage.error(
        '服务返回异常：请确认 Vite 代理可达后端。默认联调 http://106.13.108.88:8100；本机只跑 Platform 时在 .env.development 写 VITE_PROXY_TARGET=http://127.0.0.1:8081',
      )
      return
    }
    if (data.code !== 0 || !data.data) {
      ElMessage.error(data.message || '登录失败')
      return
    }
    const d = data.data as { access_token?: string }
    if (!d.access_token) {
      ElMessage.error('未返回 access_token')
      return
    }
    localStorage.setItem('mlk_admin_token', d.access_token)
    const me = await api.get<ApiEnvelope<{ role?: string }>>('/auth/me')
    if (!me.data || typeof me.data.code !== 'number' || me.data.code !== 0 || !me.data.data) {
      localStorage.removeItem('mlk_admin_token')
      ElMessage.error(me.data?.message || '无法验证身份')
      return
    }
    const role = String(me.data.data.role || '').toLowerCase()
    if (role !== 'admin') {
      localStorage.removeItem('mlk_admin_token')
      ElMessage.error('该账号不是管理员：请使用配置中 bootstrap_admin_emails 内的邮箱注册')
      return
    }
    router.push('/admin')
  } catch (e: unknown) {
    const ax = e as { response?: { data?: { message?: string } }; message?: string }
    ElMessage.error(ax.response?.data?.message || ax.message || '请求失败（检查网络与代理）')
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
