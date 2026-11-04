<template>
 <div class="auth-page">
  <section class="auth-story"><RouterLink to="/" class="brand"><span class="logo">C</span>CollabForge</RouterLink><div class="story-copy"><span class="eyebrow">BUILD TOGETHER. FLOW BETTER.</span><h1>好的协作，<br>从连接彼此开始。</h1><p>让目标清晰，让每一步都有方向。<br>你的团队工作空间，从这里启程。</p><div class="story-lines"><i></i><i></i><i></i></div></div><p class="story-footer">清晰 · 专注 · 一起完成</p></section>
  <main class="auth-main"><div class="auth-card"><slot /></div><RouterLink to="/health" class="auth-health">查看系统状态 ↗</RouterLink></main>
 </div>
</template>
