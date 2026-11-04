#!/usr/bin/env node
import {randomBytes} from 'node:crypto'
import {readFileSync,writeFileSync,existsSync} from 'node:fs'
import {fileURLToPath} from 'node:url'
const root=new URL('../',import.meta.url),target=new URL('.env',root)
if(existsSync(target)){console.log('.env 已存在，保留原配置；已有数据库不要随意更换密码。');process.exit(0)}
const content=readFileSync(new URL('.env.example',root),'utf8')
 .replace(/^MYSQL_PASSWORD=.*$/m,`MYSQL_PASSWORD=${randomBytes(24).toString('hex')}`)
 .replace(/^MYSQL_ROOT_PASSWORD=.*$/m,`MYSQL_ROOT_PASSWORD=${randomBytes(24).toString('hex')}`)
writeFileSync(target,content,{mode:0o600,flag:'wx'})
console.log(`已生成 ${fileURLToPath(target)}（0600），随机密码未输出。`)
