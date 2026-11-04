export function passwordError(value:string):string{
 if([...value].length<8||new TextEncoder().encode(value).length>72)return '密码至少 8 个字符，且最多 72 字节'
 return ''
}
export function nicknameError(value:string):string{return [...value.trim()].length<1||[...value.trim()].length>40?'昵称长度应为 1 至 40 个字符':''}
