<template>
  <el-container class="layout">
    <el-header class="hdr">
      <span class="hdr-brand"><strong>模链云</strong> · 控制台</span>
      <el-button type="danger" link @click="doLogout">退出登录</el-button>
    </el-header>
    <el-main>
      <el-card class="mb" header="账户与组织">
        <p v-if="me" class="me-line">
          <el-tag size="small" type="info">{{ me.email || '—' }}</el-tag>
          <el-tag v-if="me.role" size="small" :type="me.role === 'admin' ? 'warning' : 'success'" class="ml">
            {{ me.role === 'admin' ? '管理员（含全站组织数据权限）' : `角色：${me.role}` }}
          </el-tag>
        </p>
        <div v-if="orgs.length" class="org-row">
          <span class="label">钱包视角：</span>
          <el-select
            v-model="walletScope"
            placeholder="选择"
            style="min-width: 260px"
            filterable
            @change="onWalletScopeChange"
          >
            <el-option label="个人（user 钱包）" value="__personal__" />
            <el-option v-for="o in orgs" :key="o.id" :label="`${o.id} · ${o.name}`" :value="String(o.id)" />
          </el-select>
          <span v-if="me?.role === 'admin'" class="hint">管理员可查看任意组织钱包；创建 org 范围 Key 时请在此选中目标组织。</span>
        </div>
        <p v-else class="muted">暂无组织；仅显示个人钱包。</p>
      </el-card>

      <el-row :gutter="16">
        <el-col :span="8">
          <el-card header="钱包（分）">
            <p v-if="wallet">
              {{ wallet.balance_cents }} {{ wallet.currency }}
              <span v-if="wallet.credit_status" class="muted"> · {{ wallet.credit_status }}</span>
            </p>
            <el-button size="small" @click="load">刷新</el-button>
          </el-card>
        </el-col>
        <el-col :span="16">
          <el-card header="充值（mock）">
            <el-input-number v-model="amt" :min="100" :step="100" />
            <el-button type="primary" @click="doRecharge" style="margin-left: 8px">创建订单并模拟支付</el-button>
          </el-card>
        </el-col>
      </el-row>
      <el-card header="我的订单" class="mt">
        <el-button size="small" @click="loadOrders">刷新订单</el-button>
        <el-table :data="orders" class="order-table" size="small" stripe border empty-text="暂无订单">
          <el-table-column prop="id" label="订单号" width="96" />
          <el-table-column prop="amount_cents" label="金额(分)" width="100" />
          <el-table-column prop="currency" label="币种" width="72" />
          <el-table-column prop="channel" label="支付渠道" width="100" />
          <el-table-column prop="status" label="状态" width="96" />
          <el-table-column prop="order_type" label="类型" width="100" />
          <el-table-column prop="org_id" label="组织 ID" width="96" />
          <el-table-column prop="created_at" label="创建时间" min-width="168" show-overflow-tooltip />
          <el-table-column label="操作" width="88" fixed="right">
            <template #default="{ row }">
              <el-button link type="primary" @click="openOrderDetail(row)">详情</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-card>

      <el-card header="API Key" class="mt">
        <div class="key-toolbar">
          <el-input v-model="keyName" placeholder="名称" style="max-width: 220px; margin-right: 8px" />
          <el-button type="primary" @click="makeKey('personal')">创建（personal）</el-button>
          <el-button
            type="primary"
            plain
            :disabled="!selectedOrgIdForKey"
            @click="makeKey('org')"
          >
            创建（org · 需已选组织）
          </el-button>
          <el-button size="small" @click="loadApiKeys">刷新列表</el-button>
        </div>
        <p class="muted small">
          完整密钥仅在创建成功时显示一次，请立即用「复制密钥」保存；关闭或清除后无法再次查看，旧 Key 也无法从服务端找回。
        </p>
        <div v-if="lastFullSecret" class="secret-block">
          <div class="secret-row">
            <pre class="secret-pre">{{ lastFullSecret }}</pre>
            <el-button type="primary" @click="copyText(lastFullSecret)">复制密钥</el-button>
          </div>
          <el-button link type="info" size="small" @click="clearNewKeyDisplay">清除本页显示</el-button>
        </div>
        <el-table :data="apiKeys" class="key-table" size="small" stripe border empty-text="暂无 API Key">
          <el-table-column prop="id" label="ID" width="72" />
          <el-table-column prop="name" label="名称" min-width="120" show-overflow-tooltip />
          <el-table-column prop="scope" label="范围" width="88" />
          <el-table-column prop="org_id" label="组织 ID" width="96" />
          <el-table-column prop="key_prefix" label="前缀" min-width="160" show-overflow-tooltip />
          <el-table-column prop="status" label="状态" width="88" />
          <el-table-column label="操作" width="140" fixed="right">
            <template #default="{ row }">
              <el-button
                v-if="canCopyFullKeyForRow(row)"
                link
                type="primary"
                @click="copyText(lastFullSecret)"
              >
                复制密钥
              </el-button>
              <el-button
                link
                type="danger"
                :disabled="row.status !== 'active'"
                @click="disableKey(row)"
              >
                停用
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-card>
      <el-card header="网关调用提示" class="mt">
        <p>将 OpenAI SDK 的 baseURL 设为公网 Nginx 地址下的 <code>/mlk/v1</code>（如 <code>http://&lt;IP&gt;:8100/mlk/v1</code>）。</p>
        <p>Authorization: Bearer <code>&lt;mk_live_...&gt;</code></p>
      </el-card>
    </el-main>

    <el-drawer v-model="orderDrawerVisible" title="订单详情" size="440px" destroy-on-close>
      <el-skeleton v-if="orderDetailLoading" :rows="8" animated />
      <el-descriptions v-else-if="orderDetail" :column="1" border size="small">
        <el-descriptions-item label="订单号">{{ orderDetail.id }}</el-descriptions-item>
        <el-descriptions-item label="状态">{{ fmt(orderDetail.status) }}</el-descriptions-item>
        <el-descriptions-item label="类型">{{ fmt(orderDetail.order_type) }}</el-descriptions-item>
        <el-descriptions-item label="金额（分）">{{ fmt(orderDetail.amount_cents) }}</el-descriptions-item>
        <el-descriptions-item label="币种">{{ fmt(orderDetail.currency) }}</el-descriptions-item>
        <el-descriptions-item label="支付渠道">{{ fmt(orderDetail.channel) }}</el-descriptions-item>
        <el-descriptions-item label="组织 ID">{{ orderDetail.org_id != null ? orderDetail.org_id : '—' }}</el-descriptions-item>
        <el-descriptions-item label="创建时间">{{ fmt(orderDetail.created_at) }}</el-descriptions-item>
        <el-descriptions-item label="支付时间">{{ fmt(orderDetail.paid_at) }}</el-descriptions-item>
        <el-descriptions-item label="渠道流水号">{{ fmt(orderDetail.provider_trade_no) }}</el-descriptions-item>
      </el-descriptions>
    </el-drawer>
  </el-container>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import {
  fetchWallet,
  createApiKey,
  recharge,
  mockPayOrder,
  logoutAndClearLocal,
  fetchMe,
  listOrgs,
  switchOrg,
  applySessionTokens,
  listOrders,
  getOrder,
  listApiKeys,
  deleteApiKey,
} from '../api/client'
import { ElMessage, ElMessageBox } from 'element-plus'

const router = useRouter()

type Me = { id?: number; email?: string; display_name?: string; role?: string; current_org_id?: number | null }
const me = ref<Me | null>(null)
const orgs = ref<{ id: number; name: string; slug?: string; status?: string }[]>([])
/** el-select value: personal token or org id string */
const walletScope = ref<string>('__personal__')

const wallet = ref<{ balance_cents: number; currency: string; credit_status?: string } | null>(null)
const amt = ref(1000)
const keyName = ref('dev-key')
/** 仅本次创建返回的完整密钥，用于复制；离开或清除后不可恢复 */
const lastFullSecret = ref('')
/** 与 lastFullSecret 对应的 Key 行 id，列表中仅此行可「复制密钥」 */
const lastCreatedKeyId = ref<number | null>(null)
const apiKeys = ref<Record<string, unknown>[]>([])
const orders = ref<Record<string, unknown>[]>([])
const orderDrawerVisible = ref(false)
const orderDetailLoading = ref(false)
const orderDetail = ref<Record<string, unknown> | null>(null)

const selectedOrgIdForKey = computed(() => {
  const v = walletScope.value
  if (v === '__personal__') return null
  const n = Number(v)
  return Number.isFinite(n) ? n : null
})

function fmt(v: unknown) {
  if (v === undefined || v === null || v === '') return '—'
  return String(v)
}

function clearNewKeyDisplay() {
  lastFullSecret.value = ''
  lastCreatedKeyId.value = null
}

function canCopyFullKeyForRow(row: Record<string, unknown>) {
  if (!lastFullSecret.value || lastCreatedKeyId.value == null) return false
  return Number(row.id) === lastCreatedKeyId.value
}

async function copyText(text: string) {
  const t = text.trim()
  if (!t) {
    ElMessage.warning('没有可复制的内容')
    return
  }
  try {
    await navigator.clipboard.writeText(t)
    ElMessage.success('已复制到剪贴板')
  } catch {
    try {
      const ta = document.createElement('textarea')
      ta.value = t
      ta.style.position = 'fixed'
      ta.style.left = '-9999px'
      document.body.appendChild(ta)
      ta.select()
      document.execCommand('copy')
      document.body.removeChild(ta)
      ElMessage.success('已复制到剪贴板')
    } catch {
      ElMessage.error('复制失败，请手动选择文本复制')
    }
  }
}

async function load() {
  const oid = selectedOrgIdForKey.value
  const res = await fetchWallet(oid)
  if (res.code === 0 && res.data) wallet.value = res.data
}

async function loadOrders() {
  const res = await listOrders()
  if (res.code === 0) orders.value = (res.data?.items ?? []) as Record<string, unknown>[]
  else ElMessage.error(res.message || '订单列表加载失败')
}

async function loadApiKeys() {
  const res = await listApiKeys()
  if (res.code === 0) apiKeys.value = (res.data?.items ?? []) as Record<string, unknown>[]
  else ElMessage.error(res.message || 'API Key 列表加载失败')
}

async function disableKey(row: Record<string, unknown>) {
  try {
    await ElMessageBox.confirm(`确定停用 Key「${row.name}」（${row.key_prefix}）？停用后无法恢复。`, '停用 API Key', {
      type: 'warning',
      confirmButtonText: '停用',
      cancelButtonText: '取消',
    })
  } catch {
    return
  }
  const res = await deleteApiKey(Number(row.id))
  if (res.code !== 0) {
    ElMessage.error(res.message || '停用失败')
    return
  }
  ElMessage.success('已停用')
  await loadApiKeys()
}

async function openOrderDetail(row: Record<string, unknown>) {
  orderDrawerVisible.value = true
  orderDetailLoading.value = true
  orderDetail.value = null
  try {
    const res = await getOrder(row.id as string | number)
    if (res.code !== 0 || !res.data) {
      ElMessage.error(res.message || '加载订单失败')
      return
    }
    orderDetail.value = res.data as Record<string, unknown>
  } catch {
    ElMessage.error('请求失败')
  } finally {
    orderDetailLoading.value = false
  }
}

async function bootstrap() {
  const m = await fetchMe()
  if (m.code !== 0) {
    ElMessage.error(m.message || '无法加载账户')
    return
  }
  me.value = (m.data as Me) || null

  const o = await listOrgs()
  if (o.code === 0 && o.data?.items) {
    orgs.value = o.data.items
    const cur = me.value?.current_org_id
    if (cur != null && orgs.value.some((x) => x.id === cur)) {
      walletScope.value = String(cur)
    } else {
      walletScope.value = '__personal__'
    }
  } else {
    orgs.value = []
    walletScope.value = '__personal__'
  }
  await load()
  await loadOrders()
  await loadApiKeys()
}

async function onWalletScopeChange() {
  const oid = selectedOrgIdForKey.value
  if (oid != null) {
    const sw = await switchOrg(oid)
    if (sw.code !== 0 || !sw.data) {
      ElMessage.error(sw.message || '切换组织失败')
      return
    }
    applySessionTokens(sw.data as { access_token?: string; refresh_token?: string })
    me.value = { ...me.value, current_org_id: oid } as Me
  } else {
    me.value = { ...me.value, current_org_id: null } as Me
  }
  await load()
}

async function doRecharge() {
  const oid = selectedOrgIdForKey.value
  const r = await recharge(amt.value, 'wechat', oid ?? undefined)
  if (r.code !== 0 || !r.data?.order_id) {
    ElMessage.error(r.message || '下单失败')
    return
  }
  const p = await mockPayOrder(r.data.order_id)
  if (p.code === 0) {
    ElMessage.success('mock 入账成功')
    await load()
    await loadOrders()
  } else ElMessage.error(p.message || 'mock 支付失败')
}

async function makeKey(scope: 'personal' | 'org') {
  const oid = scope === 'org' ? selectedOrgIdForKey.value : undefined
  if (scope === 'org' && oid == null) {
    ElMessage.warning('请先在「钱包视角」中选择组织')
    return
  }
  const r = await createApiKey(keyName.value || 'key', scope, oid ?? undefined, false)
  if (r.code !== 0 || !r.data?.secret) {
    ElMessage.error(r.message || '创建失败')
    return
  }
  lastFullSecret.value = String(r.data.secret)
  lastCreatedKeyId.value = typeof r.data.id === 'number' ? r.data.id : Number(r.data.id)
  if (!Number.isFinite(lastCreatedKeyId.value)) lastCreatedKeyId.value = null
  await loadApiKeys()
}

async function doLogout() {
  await logoutAndClearLocal()
  ElMessage.success('已退出')
  router.push('/login')
}

onMounted(bootstrap)
</script>

<style scoped>
.layout {
  min-height: 100vh;
}
.hdr {
  display: flex;
  align-items: center;
  justify-content: space-between;
  line-height: 60px;
  border-bottom: 1px solid #e5e7eb;
}
.hdr-brand {
  font-size: 16px;
}
.mb {
  margin-bottom: 16px;
}
.mt {
  margin-top: 16px;
}
.me-line {
  margin: 0 0 12px;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}
.ml {
  margin-left: 4px;
}
.org-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}
.org-row .label {
  font-size: 14px;
  color: #475569;
}
.hint {
  font-size: 12px;
  color: #64748b;
  flex: 1 1 200px;
}
.muted {
  color: #94a3b8;
  font-size: 13px;
}
.secret-block {
  margin: 10px 0 12px;
  padding: 12px;
  background: #fefce8;
  border: 1px solid #fde047;
  border-radius: 8px;
}
.secret-row {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  gap: 10px;
  margin-bottom: 6px;
}
.secret-pre {
  flex: 1 1 200px;
  margin: 0;
  padding: 8px 10px;
  background: #fff;
  border: 1px solid #e2e8f0;
  border-radius: 6px;
  font-size: 12px;
  white-space: pre-wrap;
  word-break: break-all;
  max-height: 120px;
  overflow: auto;
}
.order-table {
  margin-top: 10px;
}
.key-toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}
.small {
  font-size: 12px;
  margin: 0 0 10px;
}
.key-table {
  margin-top: 4px;
}
code {
  background: #f1f5f9;
  padding: 2px 6px;
}
</style>
