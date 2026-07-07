<script setup lang="ts">
import { ref } from 'vue'
import { ElNotification } from 'element-plus'
import { request, setLoggedIn } from '../utils/request'

const emit = defineEmits<{ logged: [] }>()
const password = ref('')
const loading = ref(false)

async function submit() {
  if (!password.value) {
    ElNotification.warning({ title: '提示', message: '请输入密码' })
    return
  }
  loading.value = true
  try {
    const res = await request('/api/admin/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ password: password.value })
    })
    if (res.ok) {
      setLoggedIn(true)
      emit('logged')
    } else {
      const j = await res.json().catch(() => ({}))
      ElNotification.error({ title: '登录失败', message: (j as any).error || '密码错误' })
    }
  } catch (e: any) {
    ElNotification.error({ title: '网络错误', message: e.message })
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="login-wrap">
    <el-form class="login-form" @submit.prevent="submit">
      <h2 class="login-title">0E7 管理端</h2>
      <el-input
        v-model="password"
        type="password"
        placeholder="请输入管理端密码"
        show-password
        @keyup.enter="submit"
      />
      <el-button type="primary" :loading="loading" @click="submit" style="width: 100%; margin-top: 12px">
        登录
      </el-button>
    </el-form>
  </div>
</template>

<style scoped>
.login-wrap {
  height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f5f7fa;
}
.login-form {
  width: 320px;
  padding: 32px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
}
.login-title {
  text-align: center;
  margin-bottom: 20px;
  color: #303133;
}
</style>
