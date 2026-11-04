import test from 'node:test'
import assert from 'node:assert/strict'
import {mkdtempSync,mkdirSync,copyFileSync,readFileSync,statSync} from 'node:fs'
import {tmpdir} from 'node:os'
import {join} from 'node:path'
import {spawnSync} from 'node:child_process'
test('initialization uses private distinct credentials and never overwrites existing configuration',()=>{
 const root=mkdtempSync(join(tmpdir(),'taskflow-init-test-'));mkdirSync(join(root,'scripts'))
 copyFileSync(new URL('./init-env.mjs',import.meta.url),join(root,'scripts/init-env.mjs'));copyFileSync(new URL('../.env.example',import.meta.url),join(root,'.env.example'))
 const run=()=>spawnSync(process.execPath,[join(root,'scripts/init-env.mjs')],{encoding:'utf8'})
 const first=run();assert.equal(first.status,0);const original=readFileSync(join(root,'.env'),'utf8');assert.equal(statSync(join(root,'.env')).mode&0o777,0o600)
 const password=original.match(/^MYSQL_PASSWORD=(.+)$/m)[1],admin=original.match(/^MYSQL_ROOT_PASSWORD=(.+)$/m)[1];assert.match(password,/^[a-f0-9]{48}$/);assert.notEqual(password,admin);assert.ok(!first.stdout.includes(password));assert.equal(run().status,0);assert.equal(readFileSync(join(root,'.env'),'utf8'),original)
 // No recursive deletion: the isolated directory remains in the OS temporary area.
})
