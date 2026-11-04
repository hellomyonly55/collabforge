<script setup lang="ts">
import {onMounted} from 'vue'
import {useHealthStore} from '../stores/health'
const store=useHealthStore()
const dependencies=[{key:'mysql',name:'MySQL',detail:'业务数据持久化'},{key:'redis',name:'Redis',detail:'登录会话与请求限流'},{key:'migrations',name:'数据库迁移',detail:'校验当前数据库版本'}]
onMounted(()=>store.refresh())
</script>

<template>
 <div class="shell">
  <aside><a href="/" class="brand"><span class="logo">C</span>CollabForge</a><div class="workspace">开发工作区</div><div class="nav active"><span>◉</span> 系统状态</div><div class="aside-note">CollabForge 1.0 · 交付验收<br>团队协作，从可靠的基础开始。</div></aside>
  <main>
   <header><span><RouterLink to="/teams">返回工作区</RouterLink> <span class="slash">/</span> 系统状态</span><el-tag type="info" effect="plain">本地开发</el-tag></header>
   <section class="content">
    <div class="eyebrow">SYSTEM OVERVIEW</div>
    <div class="title-row"><div><h1>让协作，有条不紊。</h1><p>查看真实后端依赖状态，确认工作区已经准备就绪。</p></div><el-button type="primary" :loading="store.loading" @click="store.refresh">刷新状态</el-button></div>
    <el-alert v-if="store.error" :title="store.error" type="error" :closable="false" show-icon class="alert"/>
    <div class="overview"><div><span class="live-dot" :class="{ready:store.health?.status==='ready'}"></span><strong>{{store.loading?'正在检查系统':store.health?.status==='ready'?'所有依赖已就绪':store.health?'部分依赖不可用':'等待连接后端'}}</strong><p>{{store.checkedAt?`最近检查：${store.checkedAt}`:'连接成功后显示真实检查结果'}}</p></div><span class="phase">PHASE 08</span></div>
    <div class="grid"><article v-for="item in dependencies" :key="item.key" class="card"><div class="card-top"><span class="service-icon">{{item.key==='mysql'?'DB':item.key==='redis'?'R':'↗'}}</span><el-tag :type="store.health?.checks?.[item.key]==='ok'?'success':store.health?.checks?.[item.key]==='unavailable'?'danger':'info'" effect="light">{{store.loading?'检查中':store.health?.checks?.[item.key]==='ok'?'正常':store.health?.checks?.[item.key]==='unavailable'?'不可用':'未检查'}}</el-tag></div><h2>{{item.name}}</h2><p>{{item.detail}}</p></article></div>
    <div class="section-title"><h2>开发路线</h2><span>逐模块构建，每一步均可验收</span></div>
    <div class="roadmap"><div><b>00</b><span>工程基础</span><el-tag type="success">已完成</el-tag></div><div><b>01</b><span>用户认证</span><el-tag type="success">已完成</el-tag></div><div><b>02</b><span>团队管理</span><el-tag type="success">已完成</el-tag></div><div><b>03</b><span>项目管理</span><el-tag type="success">已完成</el-tag></div><div><b>04</b><span>任务与看板</span><el-tag type="success">已完成</el-tag></div><div><b>05</b><span>评论与日志</span><el-tag type="success">已完成</el-tag></div><div><b>06</b><span>数据统计</span><el-tag type="success">已完成</el-tag></div><div><b>07</b><span>管理员后台</span><el-tag type="success">已完成</el-tag></div><div><b>08</b><span>交付与回归</span><el-tag type="success">已完成</el-tag></div></div>
    <p class="footnote">健康检查验证 MySQL、Redis 和迁移状态。业务模块与管理员后台已实现，完整演示与独立回归流程均已提供。</p><p v-if="store.requestId" class="request-id">Request ID · {{store.requestId}}</p>
   </section>
  </main>
 </div>
</template>
