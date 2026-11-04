import {defineConfig} from '@playwright/test'
export default defineConfig({
 testDir:'./tests/e2e',fullyParallel:false,workers:1,retries:0,timeout:40000,
 reporter:'list',outputDir:`test-results/run-${Date.now()}`,
 use:{baseURL:process.env.TASKFLOW_E2E_URL??'http://localhost:8080',browserName:'chromium',viewport:{width:1440,height:1000},launchOptions:process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH?{executablePath:process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH}:{},screenshot:'only-on-failure'},
})
