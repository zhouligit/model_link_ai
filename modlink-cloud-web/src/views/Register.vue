<template>
  <div class="wrap">
    <el-card class="card">
      <h2>模链云 · 注册</h2>
      <el-form @submit.prevent="onSubmit">
        <el-form-item label="邮箱">
          <el-input v-model="email" autocomplete="username" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input v-model="password" type="password" autocomplete="new-password" />
        </el-form-item>
        <el-form-item label="显示名">
          <el-input v-model="displayName" autocomplete="nickname" placeholder="可选" />
        </el-form-item>
        <el-button type="primary" native-type="submit" :loading="loading">注册并登录</el-button>
      </el-form>
      <p class="hint">密码至少 8 位。若邮箱在平台「管理员 bootstrap 列表」中，注册后即为管理员。</p>
      <p class="link">
        <router-link to="/login">已有账号？去登录</router-link>
      </p>
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
const displayName = ref('')
const loading = ref(false)

async function onSubmit() {
  if (password.value.length < 8) {
    ElMessage.warning('密码至少 8 位')
    return
  }
  loading.value = true
  try {
    const { data } = await axios.post<ApiEnvelope<Record<string, unknown>>>('/mlk/platform/v1/auth/register', {
      email: email.value.trim(),
      password: password.value,
      display_name: displayName.value.trim() || undefined,
    })
    if (data.code !== 0 || !data.data) {
      ElMessage.error(data.message || '注册失败')
      return
    }
    const d = data.data as { access_token?: string; refresh_token?: string }
    if (d.access_token) localStorage.setItem('mlk_access_token', d.access_token)
    if (d.refresh_token) localStorage.setItem('mlk_refresh_token', d.refresh_token)
    ElMessage.success('注册成功')
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
.link {
  margin-top: 16px;
  text-align: center;
  font-size: 14px;
}
.link a {
  color: #38bdf8;
  text-decoration: none;
}
.link a:hover {
  text-decoration: underline;
}
</style>
