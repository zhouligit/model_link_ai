<template>
  <el-container class="layout">
    <el-aside width="220px" class="aside">
      <div class="logo">ModLink Admin</div>
      <el-menu default-active="users">
        <el-menu-item index="users">用户</el-menu-item>
        <el-menu-item index="channels">渠道</el-menu-item>
        <el-menu-item index="models">模型</el-menu-item>
      </el-menu>
    </el-aside>
    <el-container>
      <el-header class="hdr">运营控制台</el-header>
      <el-main>
        <el-tabs v-model="tab">
          <el-tab-pane label="用户列表" name="users">
            <el-table :data="users" size="small">
              <el-table-column prop="id" label="ID" width="80" />
              <el-table-column prop="email" label="邮箱" />
              <el-table-column prop="role" label="角色" width="100" />
              <el-table-column prop="status" label="状态" width="100" />
            </el-table>
          </el-tab-pane>
          <el-tab-pane label="渠道" name="ch">
            <el-table :data="channels" size="small">
              <el-table-column prop="id" label="ID" />
              <el-table-column prop="name" label="名称" />
              <el-table-column prop="base_url" label="Base URL" />
              <el-table-column prop="status" label="状态" />
            </el-table>
          </el-tab-pane>
          <el-tab-pane label="模型" name="mdl">
            <el-table :data="models" size="small">
              <el-table-column prop="model_id" label="model_id" />
              <el-table-column prop="display_name" label="显示名" />
              <el-table-column prop="enabled" label="启用" />
            </el-table>
          </el-tab-pane>
        </el-tabs>
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import api from '../api/client'
import { ElMessage } from 'element-plus'

const tab = ref('users')
const users = ref<Record<string, unknown>[]>([])
const channels = ref<Record<string, unknown>[]>([])
const models = ref<Record<string, unknown>[]>([])

async function load() {
  try {
    const [u, c, m] = await Promise.all([
      api.get<{ code: number; data?: { items: Record<string, unknown>[] } }>('/admin/users'),
      api.get<{ code: number; data?: { items: Record<string, unknown>[] } }>('/admin/channels'),
      api.get<{ code: number; data?: { items: Record<string, unknown>[] } }>('/admin/models'),
    ])
    if (u.data.code === 0 && u.data.data?.items) users.value = u.data.data.items
    if (c.data.code === 0 && c.data.data?.items) channels.value = c.data.data.items
    if (m.data.code === 0 && m.data.data?.items) models.value = m.data.data.items
  } catch {
    ElMessage.error('加载失败（需 admin 账号）')
  }
}

onMounted(load)
</script>

<style scoped>
.layout {
  min-height: 100vh;
}
.aside {
  border-right: 1px solid #e5e7eb;
}
.logo {
  padding: 16px;
  font-weight: 700;
}
.hdr {
  border-bottom: 1px solid #e5e7eb;
  line-height: 60px;
}
</style>
