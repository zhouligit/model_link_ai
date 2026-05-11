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
        <div class="hdr-text">
          <div class="hdr-title">{{ panelTitle }}</div>
          <div class="hdr-sub">{{ panelDesc }}</div>
        </div>
        <el-button type="danger" link @click="doLogout">退出登录</el-button>
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
            <el-table-column prop="id" label="ID" width="72" />
            <el-table-column prop="email" label="邮箱" min-width="160" show-overflow-tooltip />
            <el-table-column prop="display_name" label="显示名" min-width="120" show-overflow-tooltip />
            <el-table-column prop="role" label="角色" width="88" />
            <el-table-column prop="status" label="状态" width="88" />
            <el-table-column prop="created_at" label="注册时间" min-width="168" show-overflow-tooltip />
            <el-table-column label="操作" width="88" fixed="right">
              <template #default="{ row }">
                <el-button link type="primary" @click="openUserDrawer(row)">详情</el-button>
              </template>
            </el-table-column>
          </el-table>
          <template v-else-if="section === 'channels'">
            <p class="channel-hint">
              在此填写 OpenRouter 的 API Key（写入数据库）。若服务器上仍配置了环境变量
              <code>MODLINK_OPENROUTER_API_KEY</code> 或配置文件里的
              <code>openrouter_api_key</code>，会<strong>优先于</strong>此处生效。
            </p>
            <el-table :data="channels" size="small" stripe border empty-text="暂无数据">
              <el-table-column prop="id" label="ID" width="80" />
              <el-table-column prop="name" label="名称" min-width="140" />
              <el-table-column prop="base_url" label="Base URL" min-width="200" show-overflow-tooltip />
              <el-table-column label="上游密钥" width="120">
                <template #default="{ row }">
                  <el-tag v-if="row.api_key_set" type="success" size="small">已配置</el-tag>
                  <el-tag v-else type="info" size="small">未配置</el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="status" label="状态" width="100" />
              <el-table-column label="操作" width="120" fixed="right">
                <template #default="{ row }">
                  <el-button link type="primary" @click="openChannelKeyDialog(row)">配置密钥</el-button>
                </template>
              </el-table-column>
            </el-table>
          </template>
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

    <el-drawer v-model="userDrawerVisible" title="用户详情" size="560px" destroy-on-close>
      <el-skeleton v-if="userDetailLoading" :rows="10" animated />
      <template v-else>
        <h4 class="drawer-h">账户信息</h4>
        <el-descriptions v-if="userDetailUser" :column="1" border size="small" class="drawer-block">
          <el-descriptions-item label="ID">{{ userDetailUser.id }}</el-descriptions-item>
          <el-descriptions-item label="邮箱">{{ fmtCell(userDetailUser.email) }}</el-descriptions-item>
          <el-descriptions-item label="手机">{{ fmtCell(userDetailUser.phone) }}</el-descriptions-item>
          <el-descriptions-item label="显示名">{{ fmtCell(userDetailUser.display_name) }}</el-descriptions-item>
          <el-descriptions-item label="角色">{{ fmtCell(userDetailUser.role) }}</el-descriptions-item>
          <el-descriptions-item label="状态">{{ fmtCell(userDetailUser.status) }}</el-descriptions-item>
          <el-descriptions-item label="最近登录">{{ fmtCell(userDetailUser.last_login_at) }}</el-descriptions-item>
          <el-descriptions-item label="注册时间">{{ fmtCell(userDetailUser.created_at) }}</el-descriptions-item>
        </el-descriptions>

        <h4 class="drawer-h">个人钱包</h4>
        <el-descriptions v-if="userDetailWallet" :column="1" border size="small" class="drawer-block">
          <el-descriptions-item label="余额（分）">{{ userDetailWallet.balance_cents }}</el-descriptions-item>
          <el-descriptions-item label="币种">{{ fmtCell(userDetailWallet.currency) }}</el-descriptions-item>
          <el-descriptions-item label="状态">{{ fmtCell(userDetailWallet.status) }}</el-descriptions-item>
        </el-descriptions>
        <p v-else class="drawer-muted">尚无个人钱包记录（用户未触发过钱包初始化）。</p>

        <h4 class="drawer-h">API Keys</h4>
        <p class="drawer-muted">仅展示前缀与元数据；完整密钥只在用户创建 Key 时返回一次。</p>
        <el-table :data="userDetailKeys" size="small" stripe border empty-text="暂无 Key">
          <el-table-column prop="id" label="ID" width="64" />
          <el-table-column prop="name" label="名称" min-width="100" show-overflow-tooltip />
          <el-table-column prop="key_prefix" label="前缀" min-width="140" show-overflow-tooltip />
          <el-table-column prop="scope" label="范围" width="88" />
          <el-table-column prop="status" label="状态" width="88" />
          <el-table-column prop="org_id" label="组织 ID" width="96" />
          <el-table-column prop="last_used_at" label="最近使用" min-width="156" show-overflow-tooltip />
          <el-table-column prop="created_at" label="创建时间" min-width="156" show-overflow-tooltip />
        </el-table>
        <p v-if="userDetailNote" class="drawer-note">{{ userDetailNote }}</p>
      </template>
    </el-drawer>

    <el-dialog
      v-model="channelKeyDialogVisible"
      title="配置上游 API Key"
      width="480px"
      destroy-on-close
      @closed="resetChannelKeyForm"
    >
      <p class="dialog-muted">
        渠道：<strong>{{ channelKeyForm.channel_label }}</strong>。仅保存 OpenRouter（或兼容服务）的密钥；已保存的密钥不会在页面回显。
      </p>
      <el-form label-width="0">
        <el-form-item>
          <el-input
            v-model="channelKeyForm.api_key"
            type="password"
            show-password
            placeholder="粘贴 sk-or-v1-..."
            autocomplete="off"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="channelKeyDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="channelKeySaving" @click="submitChannelKey">保存</el-button>
      </template>
    </el-dialog>

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
import { useRouter } from 'vue-router'
import api, { logoutAdmin } from '../api/client'
import { ElMessage } from 'element-plus'

const router = useRouter()

type Section = 'users' | 'channels' | 'models'

const section = ref<Section>('users')
const loading = ref(true)
const users = ref<Record<string, unknown>[]>([])
const channels = ref<Record<string, unknown>[]>([])
const models = ref<Record<string, unknown>[]>([])

const userDrawerVisible = ref(false)
const userDetailLoading = ref(false)
const userDetailUser = ref<Record<string, unknown> | null>(null)
const userDetailKeys = ref<Record<string, unknown>[]>([])
const userDetailWallet = ref<Record<string, unknown> | null>(null)
const userDetailNote = ref('')

const channelKeyDialogVisible = ref(false)
const channelKeySaving = ref(false)
const channelKeyForm = reactive({
  channel_id: 0,
  channel_label: '',
  api_key: '',
})

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

function fmtCell(v: unknown) {
  if (v === undefined || v === null || v === '') return '—'
  return String(v)
}

type AdminUserDetailPayload = {
  user: Record<string, unknown>
  api_keys: Record<string, unknown>[]
  wallet: Record<string, unknown> | null
  note?: string
}

async function openUserDrawer(row: Record<string, unknown>) {
  userDrawerVisible.value = true
  userDetailLoading.value = true
  userDetailUser.value = null
  userDetailKeys.value = []
  userDetailWallet.value = null
  userDetailNote.value = ''
  try {
    const id = row.id as string | number
    const { data } = await api.get<{ code: number; message?: string; data?: AdminUserDetailPayload }>(
      `/admin/users/${id}`,
    )
    if (data.code !== 0 || !data.data) {
      ElMessage.error(data.message || '加载失败')
      return
    }
    const d = data.data
    userDetailUser.value = d.user
    userDetailKeys.value = Array.isArray(d.api_keys) ? d.api_keys : []
    userDetailWallet.value = d.wallet ?? null
    userDetailNote.value = typeof d.note === 'string' ? d.note : ''
  } catch {
    ElMessage.error('请求失败')
  } finally {
    userDetailLoading.value = false
  }
}

const panelTitle = computed(() => {
  if (section.value === 'users') return '用户列表'
  if (section.value === 'channels') return '渠道列表'
  return '模型列表'
})

const panelDesc = computed(() => {
  if (section.value === 'users') return '平台注册用户；点「详情」查看账户、钱包与 API Key 前缀'
  if (section.value === 'channels') return '在页面配置 OpenRouter Key；列表显示是否已在库中写入有效密钥'
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

function resetChannelKeyForm() {
  channelKeyForm.channel_id = 0
  channelKeyForm.channel_label = ''
  channelKeyForm.api_key = ''
}

function openChannelKeyDialog(row: Record<string, unknown>) {
  resetChannelKeyForm()
  channelKeyForm.channel_id = Number(row.id) || 0
  channelKeyForm.channel_label = `${row.id} · ${row.name ?? ''}`
  channelKeyDialogVisible.value = true
}

async function submitChannelKey() {
  const id = channelKeyForm.channel_id
  const key = channelKeyForm.api_key.trim()
  if (!id) {
    ElMessage.warning('渠道无效')
    return
  }
  if (!key) {
    ElMessage.warning('请填写 API Key')
    return
  }
  channelKeySaving.value = true
  try {
    const { data } = await api.patch<{ code: number; message?: string }>(`/admin/channels/${id}`, {
      api_key: key,
    })
    if (data.code !== 0) {
      ElMessage.error(data.message || '保存失败')
      return
    }
    ElMessage.success('已保存（网关将使用该渠道密钥，除非环境变量/配置文件覆盖）')
    channelKeyDialogVisible.value = false
    await load()
  } catch {
    ElMessage.error('请求失败')
  } finally {
    channelKeySaving.value = false
  }
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

async function doLogout() {
  await logoutAdmin()
  ElMessage.success('已退出')
  router.push('/login')
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
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}
.hdr-text {
  min-width: 0;
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
.drawer-h {
  margin: 16px 0 8px;
  font-size: 15px;
  color: #0f172a;
}
.drawer-h:first-of-type {
  margin-top: 0;
}
.drawer-block {
  margin-bottom: 8px;
}
.drawer-muted {
  margin: 0 0 12px;
  font-size: 13px;
  color: #64748b;
}
.drawer-note {
  margin-top: 12px;
  font-size: 12px;
  color: #94a3b8;
  line-height: 1.5;
}
.channel-hint {
  margin: 0 0 12px;
  font-size: 13px;
  color: #64748b;
  line-height: 1.55;
}
.channel-hint code {
  font-size: 12px;
  background: #f1f5f9;
  padding: 1px 6px;
  border-radius: 4px;
}
.dialog-muted {
  margin: 0 0 14px;
  font-size: 13px;
  color: #64748b;
  line-height: 1.5;
}
</style>
