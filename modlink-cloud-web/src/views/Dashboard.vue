<template>
  <el-container class="layout">
    <el-header><strong>模链云</strong> · 控制台</el-header>
    <el-main>
      <el-row :gutter="16">
        <el-col :span="8">
          <el-card header="钱包（分）">
            <p v-if="wallet">{{ wallet.balance_cents }} {{ wallet.currency }}</p>
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
      <el-card header="API Key" style="margin-top: 16px">
        <el-input v-model="keyName" placeholder="名称" style="max-width: 240px; margin-right: 8px" />
        <el-button type="primary" @click="makeKey">创建（personal）</el-button>
        <pre v-if="secret" style="margin-top: 12px; white-space: pre-wrap">{{ secret }}</pre>
      </el-card>
      <el-card header="网关调用提示" style="margin-top: 16px">
        <p>将 OpenAI SDK 的 baseURL 设为 <code>http://127.0.0.1:8080/mlk/v1</code>（或通过 Nginx 暴露的域名）。</p>
        <p>Authorization: Bearer <code>&lt;mk_live_...&gt;</code></p>
      </el-card>
    </el-main>
  </el-container>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { fetchWallet, createApiKey, recharge, mockPayOrder } from '../api/client'
import { ElMessage } from 'element-plus'

const wallet = ref<{ balance_cents: number; currency: string } | null>(null)
const amt = ref(1000)
const keyName = ref('dev-key')
const secret = ref('')

async function load() {
  const res = await fetchWallet()
  if (res.code === 0 && res.data) wallet.value = res.data
}

async function doRecharge() {
  const r = await recharge(amt.value)
  if (r.code !== 0 || !r.data?.order_id) {
    ElMessage.error(r.message || '下单失败')
    return
  }
  const p = await mockPayOrder(r.data.order_id)
  if (p.code === 0) {
    ElMessage.success('mock 入账成功')
    await load()
  } else ElMessage.error(p.message || 'mock 支付失败')
}

async function makeKey() {
  const r = await createApiKey(keyName.value || 'key', 'personal', undefined, false)
  if (r.code !== 0 || !r.data?.secret) {
    ElMessage.error(r.message || '创建失败')
    return
  }
  secret.value = `保存此密钥（仅显示一次）：\n${r.data.secret}`
}

onMounted(load)
</script>

<style scoped>
.layout {
  min-height: 100vh;
}
.el-header {
  line-height: 60px;
  border-bottom: 1px solid #e5e7eb;
}
code {
  background: #f1f5f9;
  padding: 2px 6px;
}
</style>
