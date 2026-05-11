<template>
  <el-container class="layout">
    <el-aside width="240px" class="aside">
      <div class="logo">ModLink Admin</div>
      <el-menu
        :default-active="section"
        class="menu"
        background-color="#0b1220"
        text-color="#94a3b8"
        active-text-color="#38bdf8"
        @select="onMenuSelect"
      >
        <el-menu-item index="users">用户与权限</el-menu-item>
        <el-menu-item index="channels">上游渠道</el-menu-item>
        <el-menu-item index="models">模型目录</el-menu-item>
      </el-menu>
    </el-aside>
    <el-container class="main-wrap">
      <el-header class="hdr">
        <div class="hdr-title">{{ panelTitle }}</div>
        <div class="hdr-sub">{{ panelDesc }}</div>
      </el-header>
      <el-main class="main">
        <el-skeleton v-if="loading" :rows="8" animated />
        <el-card v-else shadow="never" class="card">
          <div v-if="section === 'models'" class="toolbar">
            <el-button type="primary" @click="openModelDialog">添加模型</el-button>
          </div>
          <el-table
            v-if="section === 'users'"
            :data="users"
            size="small"
            stripe
            border
            empty-text="暂无数据"
          >
            <el-table-column prop="id" label="ID" width="80" />
            <el-table-column prop="email" label="邮箱" min-width="200" />
            <el-table-column prop="role" label="角色" width="100" />
            <el-table-column prop="status" label="状态" width="100" />
          </el-table>
          <el-table
            v-else-if="section === 'channels'"
            :data="channels"
            size="small"
            stripe
            border
            empty-text="暂无数据"
          >
            <el-table-column prop="id" label="ID" width="80" />
            <el-table-column prop="name" label="名称" min-width="140" />
            <el-table-column prop="base_url" label="Base URL" min-width="220" show-overflow-tooltip />
            <el-table-column prop="status" label="状态" width="100" />
          </el-table>
          <el-table
            v-else-if="section === 'models'"
            :data="models"
            size="small"
            stripe
            border
            empty-text="暂无数据"
          >
            <el-table-column prop="model_id" label="model_id" min-width="180" show-overflow-tooltip />
            <el-table-column prop="display_name" label="显示名" min-width="140" />
            <el-table-column prop="enabled" label="启用" width="90" />
          </el-table>
        </el-card>
      </el-main>
    </el-container>

    <el-dialog v-model="modelDialogVisible" title="添加模型" width="520px" destroy-on-close @closed="resetModelForm">
      <el-form :model="modelForm" label-width="140px" label-position="left">
        <el-form-item label="model_id" required>
          <el-input v-model="modelForm.model_id" placeholder="如 openai/gpt-4o（OpenRouter 模型 ID）" />
        </el-form-item>
        <el-form-item label="显示名">
          <el-input v-model="modelForm.display_name" placeholder="留空则与 model_id 相同" />
        </el-form-item>
        <el-form-item label="上游 model_id">
          <el-input v-model="modelForm.upstream_model_id" placeholder="留空则与 model_id 相同" />
        </el-form-item>
        <el-form-item label="渠道">
          <el-select v-model="modelForm.channel_id" placeholder="选择渠道" style="width: 100%">
            <el-option
              v-for="ch in channels"
              :key="String(ch.id)"
              :label="`${ch.id} · ${ch.name}`"
              :value="Number(ch.id)"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="输入 ¢/1K tokens">
          <el-input-number v-model="modelForm.input_per_1k_cents" :min="0" :step="1" controls-position="right" />
        </el-form-item>
        <el-form-item label="输出 ¢/1K tokens">
          <el-input-number v-model="modelForm.output_per_1k_cents" :min="0" :step="1" controls-position="right" />
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="modelForm.enabled" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="modelDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="modelSaving" @click="submitModel">保存</el-button>
      </template>
    </el-dialog>
  </el-container>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import api from '../api/client'
import { ElMessage } from 'element-plus'

type Section = 'users' | 'channels' | 'models'

const section = ref<Section>('users')
const loading = ref(true)
const users = ref<Record<string, unknown>[]>([])
const channels = ref<Record<string, unknown>[]>([])
const models = ref<Record<string, unknown>[]>([])

const modelDialogVisible = ref(false)
const modelSaving = ref(false)
const modelForm = reactive({
  model_id: '',
  display_name: '',
  upstream_model_id: '',
  channel_id: 1,
  input_per_1k_cents: 1,
  output_per_1k_cents: 3,
  enabled: true,
})

function onMenuSelect(index: string) {
  section.value = index as Section
}

const panelTitle = computed(() => {
  if (section.value === 'users') return '用户列表'
  if (section.value === 'channels') return '渠道列表'
  return '模型列表'
})

const panelDesc = computed(() => {
  if (section.value === 'users') return '平台注册用户与角色（需管理员 Token）'
  if (section.value === 'channels') return 'OpenRouter 等上游渠道配置'
  return '对外可选模型；新增会写入目录、路由与计价三表'
})

function resetModelForm() {
  modelForm.model_id = ''
  modelForm.display_name = ''
  modelForm.upstream_model_id = ''
  modelForm.channel_id = channels.value.length ? Number(channels.value[0].id) || 1 : 1
  modelForm.input_per_1k_cents = 1
  modelForm.output_per_1k_cents = 3
  modelForm.enabled = true
}

function openModelDialog() {
  resetModelForm()
  modelDialogVisible.value = true
}

async function submitModel() {
  const mid = modelForm.model_id.trim()
  if (!mid) {
    ElMessage.warning('请填写 model_id')
    return
  }
  modelSaving.value = true
  try {
    const { data } = await api.post<{
      code: number
      message?: string
    }>('/admin/models', {
      model_id: mid,
      display_name: modelForm.display_name.trim() || undefined,
      upstream_model_id: modelForm.upstream_model_id.trim() || undefined,
      channel_id: modelForm.channel_id || 1,
      input_per_1k_cents: modelForm.input_per_1k_cents,
      output_per_1k_cents: modelForm.output_per_1k_cents,
      enabled: modelForm.enabled,
    })
    if (data.code !== 0) {
      ElMessage.error(data.message || '保存失败')
      return
    }
    ElMessage.success('已添加')
    modelDialogVisible.value = false
    await load()
  } catch {
    ElMessage.error('请求失败')
  } finally {
    modelSaving.value = false
  }
}

async function load() {
  loading.value = true
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
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.layout {
  min-height: 100vh;
  background: #f1f5f9;
}
.aside {
  border-right: 1px solid #1e293b;
  background: #0b1220;
}
.logo {
  padding: 20px 16px;
  font-weight: 700;
  font-size: 1.05rem;
  color: #f8fafc;
  letter-spacing: 0.02em;
}
.menu {
  border-right: none;
}
.main-wrap {
  background: #f1f5f9;
}
.hdr {
  height: auto !important;
  padding: 20px 24px 12px;
  background: #fff;
  border-bottom: 1px solid #e2e8f0;
}
.hdr-title {
  font-size: 1.25rem;
  font-weight: 600;
  color: #0f172a;
}
.hdr-sub {
  margin-top: 6px;
  font-size: 13px;
  color: #64748b;
}
.main {
  padding: 20px 24px 32px;
}
.card {
  border-radius: 10px;
  border: 1px solid #e2e8f0;
}
.toolbar {
  margin-bottom: 12px;
}
</style>
