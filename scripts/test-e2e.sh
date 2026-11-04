#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
if [ ! -f .env ]; then echo 'Create a trusted .env from .env.example first.' >&2; exit 1; fi
set -a
. ./.env
set +a
taskflow_e2e_root=$(pwd)
taskflow_e2e_credentials=$(mktemp "${TMPDIR:-/tmp}/taskflow-e2e-credentials.XXXXXX")
export TASKFLOW_E2E_CREDENTIALS_FILE="$taskflow_e2e_credentials"
cleanup() { if [ -f "$taskflow_e2e_credentials" ]; then rm "$taskflow_e2e_credentials"; fi; docker compose -p taskflow-e2e --project-directory "$taskflow_e2e_root" -f "$taskflow_e2e_root/compose.e2e.yaml" stop; }
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
docker compose -p taskflow-e2e -f compose.e2e.yaml up --build -d
export TASKFLOW_E2E_URL="http://localhost:${E2E_WEB_PORT:-18080}"
node --input-type=module <<'JS'
const endpoint=process.env.TASKFLOW_E2E_URL+'/api/v1/health/ready';
let ready=false;
for(let i=0;i<30;i++){
 try{if((await fetch(endpoint,{signal:AbortSignal.timeout(2000)})).ok){ready=true;break}}catch{}
 await new Promise(resolve=>setTimeout(resolve,500));
}
if(!ready)throw new Error('E2E application did not become ready');
JS
node --input-type=module <<'JS'
import {randomUUID} from 'node:crypto';
import {spawnSync} from 'node:child_process';
import {writeFileSync} from 'node:fs';
const values={TASKFLOW_E2E_ADMIN_EMAIL:`admin-${randomUUID()}@example.test`,TASKFLOW_E2E_ADMIN_PASSWORD:`Admin-${randomUUID()}`,TASKFLOW_E2E_MEMBER_EMAIL:`member-${randomUUID()}@example.test`,TASKFLOW_E2E_MEMBER_PASSWORD:`Member-${randomUUID()}`};
const command=spawnSync('docker',['compose','-p','taskflow-e2e','-f','compose.e2e.yaml','exec','-T','backend','/app/admin','--email',values.TASKFLOW_E2E_ADMIN_EMAIL,'--nickname','测试平台管理员','--password-stdin'],{input:values.TASKFLOW_E2E_ADMIN_PASSWORD+'\n',encoding:'utf8'});
if(command.status!==0)throw new Error('E2E administrator CLI initialization failed');
const response=await fetch(process.env.TASKFLOW_E2E_URL+'/api/v1/auth/register',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({email:values.TASKFLOW_E2E_MEMBER_EMAIL,password:values.TASKFLOW_E2E_MEMBER_PASSWORD,nickname:'后台测试成员'})});
if(response.status!==201)throw new Error('E2E ordinary account registration failed');
writeFileSync(process.env.TASKFLOW_E2E_CREDENTIALS_FILE,Object.entries(values).map(([k,v])=>`${k}=${v}`).join('\n')+'\n',{mode:0o600});
JS
set -a
. "$taskflow_e2e_credentials"
set +a
cd frontend
# Use npm ci once before running; a browser must be installed or explicitly selected.
npm run test:e2e -- "$@"
