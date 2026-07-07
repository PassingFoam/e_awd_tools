<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount } from 'vue'
import TabManager from './components/TabManager.vue'
import Login from './components/Login.vue'
import { getLoggedIn, setLoggedIn } from './utils/request'

// 登录态：先以 localStorage 快速渲染（避免刷新闪烁），挂载后用 /api/admin/status 校正
const loggedIn = ref(getLoggedIn())

async function probe() {
  try {
    const res = await fetch('/api/admin/status', { credentials: 'include' })
    const j = await res.json()
    loggedIn.value = !!j.logged_in
    setLoggedIn(loggedIn.value)
  } catch {
    loggedIn.value = false
  }
}

function onUnauth() {
  loggedIn.value = false
}

onMounted(() => {
  probe()
  window.addEventListener('0e7-unauth', onUnauth)
})
onBeforeUnmount(() => {
  window.removeEventListener('0e7-unauth', onUnauth)
})
</script>

<template>
  <div class="app-container">
    <Login v-if="!loggedIn" @logged="loggedIn = true" />
    <TabManager v-else />
  </div>
</template>

<style>
/* 全局重置样式 */
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

html, body {
  height: 100%;
  width: 100%;
  overflow: hidden;
}

#app {
  height: 100vh;
  width: 100vw;
}
</style>

<style scoped>
.app-container {
  height: 100vh;
  display: flex;
  flex-direction: column;
  background: #f5f7fa;
  padding: 20px;
}

.results-container {
  border: 1px solid #e6e8eb;
  border-radius: 4px;
  padding: 20px;
  margin-top: 20px;
  background: #fff;
  box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.1);
}

.border-card {
  border-radius: 2px;
  margin-top: 20px;
}
</style>
