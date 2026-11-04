import {defineConfig} from '@playwright/test'
import base from './playwright.config'
export default defineConfig({...base,testDir:'./tests/release',timeout:90000,outputDir:`test-results/release-${Date.now()}`})
