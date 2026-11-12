---
name: fs-reimburse
description: Submit reimbursement receipts (meal or purchase) to the company reimbursement site nodelink-intranet (10.1.2.247) on macOS, Linux or Windows (Git Bash or PowerShell), using only tools every system already has (curl, base64, sed or PowerShell) — nothing to install. Logs in with the FSR_EMAIL / FSR_PASSWORD environment variables on every run. Use when the user asks to upload, submit or file a receipt / 小票 / 发票 / 报销凭证, says things like "帮我把这张小票按餐饮报销上传" or "采购报销", or asks about their recent reimbursement records.
---

# 报销上传（fs-reimburse）

把小票图片提交到公司报销网站（nodelink-intranet，PocketBase 后端，`http://10.1.2.247`，需要公司内网或 VPN），生成一条“待审核”记录。金额由管理员审核时填写，不需要你填。

不需要安装任何程序。下面每一步都有两套命令，按你执行命令的环境选一套，原样执行，只替换尖括号里的内容：

- **sh 版**：macOS、Linux、Windows 的 Git Bash（bash / zsh / sh 都可以）。只用到 `curl`、`base64`、`sed`、`tr`、`date`，这些系统都自带。
- **PowerShell 版**：Windows 上用 PowerShell 执行命令时（Windows PowerShell 5.1 或 PowerShell 7）。如果只能用 cmd，就用 `powershell -NoProfile -Command` 执行 PowerShell 版。

每段命令开头都会**重新登录一次**，不读也不写任何本地文件。登录只用两个环境变量：`FSR_EMAIL`（邮箱）和 `FSR_PASSWORD`（密码）。旧写法 `yx` / `mm` 也支持。

## 安全规则

- **密码绝对不能经过你。** 不要在对话里问密码，不要替用户设置密码，不要把密码写进命令、输出、issue 或 PR。命令只在内部读取环境变量，不要改成把密码写在命令里。
- 不要打印 token、密码或这两个环境变量的值。不要执行 `env`、`printenv`、`echo $FSR_PASSWORD` 这类命令。
- 登录结果的处理：
  - `NO_ENV`：告诉用户需要在智能体的运行环境里设置 `FSR_EMAIL`（报销网站邮箱）和 `FSR_PASSWORD`（密码）两个环境变量。Multica 里在“智能体设置 → 环境变量”添加；本机终端的设置方法见 skill 仓库的 README。然后**停下**。
  - `LOGIN_FAILED …`：告诉用户网站拒绝了这个邮箱和密码，请检查这两个环境变量的值，然后停下。
  - `CONNECT_FAILED …`：告诉用户连不上报销网站，请连公司内网或 VPN 后再试。

## 1. 确定参数

- **类型**：默认餐饮 `MEAL`。只有用户明确说采购、买东西、办公用品等，才用 `SHOPPING_RECEIPT`，并带上用途（memo）。用户说采购但没说用途时，问一句；用户不想写也可以不带。餐饮不带用途。
- **日期**：固定为今天，由命令自动生成。如果用户说的是以前的小票，提醒他记录日期会是今天。
- **图片**：本地图片路径。一条记录 1–5 张；超过 5 张时，先问用户是拆成多条，还是只挑 5 张。一张图里有几张小票、或者多张图属于同一笔报销，都放在同一条记录里。
  - 只支持 `.jpg` / `.jpeg` / `.png` / `.webp` / `.gif`。图片会原样上传，不做压缩。
  - iPhone 的 `.heic` 网站显示不了：请用户发截图，或者导出为 JPEG 再给你。
  - 单张超过 3 MB 时，建议用户改发截图或小一点的图片，太大可能被网站拒绝或打开很慢。

## 2. 预检（必须）

预检会登录，并检查图片在不在、有多大，**不会提交**。

**sh 版**：

```sh
U=${INTRANET_URL:-http://10.1.2.247}
j() { printf '%s' "$1" | tr '\n\r\t' '   ' | sed 's/\\/\\\\/g; s/"/\\"/g'; }
v() { printf '%s' "$R" | sed -n 's/.*"'"$1"'" *: *"\([^"]*\)".*/\1/p'; }
fsr_login() {
  E=${FSR_EMAIL:-$yx}
  if [ -z "$E" ] || [ -z "${FSR_PASSWORD:-$mm}" ]; then echo "NO_ENV"; return 1; fi
  R=$(printf '{"identity":"%s","password":"%s"}' "$(j "$E")" "$(j "${FSR_PASSWORD:-$mm}")" \
    | curl -s -m 20 -X POST "$U/api/collections/users/auth-with-password" -H 'Content-Type: application/json' --data-binary @-)
  if [ -z "$R" ]; then echo "CONNECT_FAILED $U"; return 1; fi
  TOKEN=$(v token)
  if [ -z "$TOKEN" ]; then echo "LOGIN_FAILED $(v message)"; return 1; fi
  ME_ID=$(v id); ME_EMAIL=$(v email); ME_NAME=$(v name)
}
if fsr_login; then
  for f in "<图片1>" "<图片2>"; do
    if [ -f "$f" ]; then echo "FILE $(wc -c < "$f" | tr -d ' ') bytes: $f"; else echo "MISSING $f"; fi
  done
  echo "OK $ME_EMAIL $(date +%F)"
fi
```

**PowerShell 版**：

```powershell
$U = if ($env:INTRANET_URL) { $env:INTRANET_URL } else { "http://10.1.2.247" }
$E = if ($env:FSR_EMAIL) { $env:FSR_EMAIL } else { $env:yx }
$A = $null
if (-not $E -or -not ($env:FSR_PASSWORD -or $env:mm)) { "NO_ENV" } else {
  $Login = @{ identity = $E; password = $(if ($env:FSR_PASSWORD) { $env:FSR_PASSWORD } else { $env:mm }) } | ConvertTo-Json -Compress
  try { $A = Invoke-RestMethod -Method Post -Uri "$U/api/collections/users/auth-with-password" -ContentType "application/json; charset=utf-8" -Body ([Text.Encoding]::UTF8.GetBytes($Login)) -TimeoutSec 20 }
  catch { if ($_.Exception.Response) { "LOGIN_FAILED " + $_.ErrorDetails.Message } else { "CONNECT_FAILED " + $_.Exception.Message } }
  Remove-Variable Login
}
if ($A) {
  foreach ($f in @("<图片1>", "<图片2>")) {
    if (Test-Path -LiteralPath $f -PathType Leaf) { "FILE $((Get-Item -LiteralPath $f).Length) bytes: $f" } else { "MISSING $f" }
  }
  "OK $($A.record.email) $(Get-Date -Format yyyy-MM-dd)"
}
```

- 输出 `OK <邮箱> <日期>`：预检通过。
- 有 `MISSING`：图片路径不对，请用户重新提供。
- `NO_ENV` / `LOGIN_FAILED` / `CONNECT_FAILED`：按上面“安全规则”处理，然后停下。

## 3. 请用户确认（必须）

告诉用户：“将以 <邮箱> 提交 <N> 张凭证：<餐饮/采购（用途）>，日期 <今天>。确认提交吗？”用户明确同意后才继续。用户在同一句话里已经说了“直接提交，不用确认”的，可以跳过确认，但预检仍然要做。

## 4. 提交

把 `TYPE`、`MEMO` 和图片路径换成实际值；餐饮时 `MEMO` 留空。

**sh 版**：

```sh
TYPE=MEAL            # 或 SHOPPING_RECEIPT
MEMO=""              # 采购用途，餐饮留空
U=${INTRANET_URL:-http://10.1.2.247}
j() { printf '%s' "$1" | tr '\n\r\t' '   ' | sed 's/\\/\\\\/g; s/"/\\"/g'; }
v() { printf '%s' "$R" | sed -n 's/.*"'"$1"'" *: *"\([^"]*\)".*/\1/p'; }
fsr_login() {
  E=${FSR_EMAIL:-$yx}
  if [ -z "$E" ] || [ -z "${FSR_PASSWORD:-$mm}" ]; then echo "NO_ENV"; return 1; fi
  R=$(printf '{"identity":"%s","password":"%s"}' "$(j "$E")" "$(j "${FSR_PASSWORD:-$mm}")" \
    | curl -s -m 20 -X POST "$U/api/collections/users/auth-with-password" -H 'Content-Type: application/json' --data-binary @-)
  if [ -z "$R" ]; then echo "CONNECT_FAILED $U"; return 1; fi
  TOKEN=$(v token)
  if [ -z "$TOKEN" ]; then echo "LOGIN_FAILED $(v message)"; return 1; fi
  ME_ID=$(v id); ME_EMAIL=$(v email); ME_NAME=$(v name)
}
if fsr_login; then
  T=$(mktemp -d); B="$T/body.json"; BAD=""; n=0
  printf '{"userId":"%s","userEmail":"%s","userName":"%s","date":"%s","type":"%s","status":"pending","reimbursementAmount":0,' \
    "$ME_ID" "$ME_EMAIL" "$ME_NAME" "$(date +%F)" "$TYPE" > "$B"
  if [ "$TYPE" = SHOPPING_RECEIPT ] && [ -n "$MEMO" ]; then printf '"memo":"%s",' "$(j "$MEMO")" >> "$B"; fi
  printf '"receiptUrl":"[' >> "$B"
  for f in "<图片1>" "<图片2>"; do
    m=""
    case "$f" in
      *.[jJ][pP][gG] | *.[jJ][pP][eE][gG]) m=image/jpeg ;;
      *.[pP][nN][gG]) m=image/png ;;
      *.[wW][eE][bB][pP]) m=image/webp ;;
      *.[gG][iI][fF]) m=image/gif ;;
    esac
    if [ -z "$m" ] || [ ! -f "$f" ]; then BAD="$BAD $f"; continue; fi
    if [ $n -gt 0 ]; then printf ',' >> "$B"; fi
    printf '\\"data:%s;base64,' "$m" >> "$B"
    base64 < "$f" | tr -d '\r\n' >> "$B"
    printf '\\"' >> "$B"; n=$((n+1))
  done
  printf ']"}' >> "$B"
  if [ -n "$BAD" ]; then echo "BAD_IMAGE$BAD"
  else
    CODE=$(curl -s -m 120 -o "$T/res.json" -w '%{http_code}' -X POST "$U/api/collections/records/records?fields=id,date,type,status" \
      -H 'Content-Type: application/json' -H "Authorization: $TOKEN" --data-binary @"$B")
    R=$(cat "$T/res.json" 2>/dev/null)
    if [ "$CODE" = 200 ]; then echo "CREATED id=$(v id) date=$(v date) type=$(v type) status=$(v status)"
    else echo "FAILED $CODE $R"; fi
  fi
  rm -rf "$T"
fi
```

**PowerShell 版**：

```powershell
$TYPE = "MEAL"       # 或 SHOPPING_RECEIPT
$MEMO = ""           # 采购用途，餐饮留空
$U = if ($env:INTRANET_URL) { $env:INTRANET_URL } else { "http://10.1.2.247" }
$E = if ($env:FSR_EMAIL) { $env:FSR_EMAIL } else { $env:yx }
$A = $null
if (-not $E -or -not ($env:FSR_PASSWORD -or $env:mm)) { "NO_ENV" } else {
  $Login = @{ identity = $E; password = $(if ($env:FSR_PASSWORD) { $env:FSR_PASSWORD } else { $env:mm }) } | ConvertTo-Json -Compress
  try { $A = Invoke-RestMethod -Method Post -Uri "$U/api/collections/users/auth-with-password" -ContentType "application/json; charset=utf-8" -Body ([Text.Encoding]::UTF8.GetBytes($Login)) -TimeoutSec 20 }
  catch { if ($_.Exception.Response) { "LOGIN_FAILED " + $_.ErrorDetails.Message } else { "CONNECT_FAILED " + $_.Exception.Message } }
  Remove-Variable Login
}
if ($A) {
  $Mime = @{ ".jpg" = "image/jpeg"; ".jpeg" = "image/jpeg"; ".png" = "image/png"; ".webp" = "image/webp"; ".gif" = "image/gif" }
  $Urls = @(); $Bad = @()
  foreach ($f in @("<图片1>", "<图片2>")) {
    $m = $Mime[[IO.Path]::GetExtension($f).ToLower()]
    if (-not $m -or -not (Test-Path -LiteralPath $f -PathType Leaf)) { $Bad += $f; continue }
    $Urls += "data:$m;base64," + [Convert]::ToBase64String([IO.File]::ReadAllBytes((Resolve-Path -LiteralPath $f).ProviderPath))
  }
  if ($Bad) { "BAD_IMAGE $Bad" } else {
    $Rec = [ordered]@{ userId = $A.record.id; userEmail = $A.record.email; userName = $A.record.name; date = (Get-Date -Format yyyy-MM-dd)
      type = $TYPE; status = "pending"; reimbursementAmount = 0; receiptUrl = (ConvertTo-Json -InputObject $Urls -Compress) }
    if ($TYPE -eq "SHOPPING_RECEIPT" -and $MEMO) { $Rec.memo = $MEMO }
    try {
      $R = Invoke-RestMethod -Method Post -Uri "$U/api/collections/records/records?fields=id,date,type,status" -Headers @{ Authorization = $A.token } `
        -ContentType "application/json; charset=utf-8" -Body ([Text.Encoding]::UTF8.GetBytes(($Rec | ConvertTo-Json -Compress))) -TimeoutSec 120
      "CREATED id=$($R.id) date=$($R.date) type=$($R.type) status=$($R.status)"
    } catch { if ($_.Exception.Response) { "FAILED $([int]$_.Exception.Response.StatusCode) $($_.ErrorDetails.Message)" } else { "FAILED 000 " + $_.Exception.Message } }
  }
}
```

- `CREATED id=…`：成功。把**记录 ID**、类型和日期告诉用户。
- `BAD_IMAGE …`：列出的图片不存在或格式不支持，按第 1 步处理。
- `FAILED <状态码> …`：见下面的“出错时”。

同一组图片只提交一次。如果不确定上次有没有成功，请用户先到网站上看一下，不要直接重复提交。

## 5. 查询最近记录

**sh 版**（前面的 `U=`、`j()`、`v()`、`fsr_login()` 这几行和第 2 步完全一样，原样复制）：

```sh
U=${INTRANET_URL:-http://10.1.2.247}
j() { printf '%s' "$1" | tr '\n\r\t' '   ' | sed 's/\\/\\\\/g; s/"/\\"/g'; }
v() { printf '%s' "$R" | sed -n 's/.*"'"$1"'" *: *"\([^"]*\)".*/\1/p'; }
fsr_login() {
  E=${FSR_EMAIL:-$yx}
  if [ -z "$E" ] || [ -z "${FSR_PASSWORD:-$mm}" ]; then echo "NO_ENV"; return 1; fi
  R=$(printf '{"identity":"%s","password":"%s"}' "$(j "$E")" "$(j "${FSR_PASSWORD:-$mm}")" \
    | curl -s -m 20 -X POST "$U/api/collections/users/auth-with-password" -H 'Content-Type: application/json' --data-binary @-)
  if [ -z "$R" ]; then echo "CONNECT_FAILED $U"; return 1; fi
  TOKEN=$(v token)
  if [ -z "$TOKEN" ]; then echo "LOGIN_FAILED $(v message)"; return 1; fi
  ME_ID=$(v id); ME_EMAIL=$(v email); ME_NAME=$(v name)
}
if fsr_login; then
  curl -s -m 20 -G "$U/api/collections/records/records" -H "Authorization: $TOKEN" \
    --data-urlencode "filter=userId=\"$ME_ID\"" --data-urlencode "sort=-created" --data-urlencode "perPage=10" \
    --data-urlencode "fields=id,date,type,status,reimbursementAmount,memo"
fi
```

**PowerShell 版**（前面的登录部分和第 2 步完全一样）：

```powershell
$U = if ($env:INTRANET_URL) { $env:INTRANET_URL } else { "http://10.1.2.247" }
$E = if ($env:FSR_EMAIL) { $env:FSR_EMAIL } else { $env:yx }
$A = $null
if (-not $E -or -not ($env:FSR_PASSWORD -or $env:mm)) { "NO_ENV" } else {
  $Login = @{ identity = $E; password = $(if ($env:FSR_PASSWORD) { $env:FSR_PASSWORD } else { $env:mm }) } | ConvertTo-Json -Compress
  try { $A = Invoke-RestMethod -Method Post -Uri "$U/api/collections/users/auth-with-password" -ContentType "application/json; charset=utf-8" -Body ([Text.Encoding]::UTF8.GetBytes($Login)) -TimeoutSec 20 }
  catch { if ($_.Exception.Response) { "LOGIN_FAILED " + $_.ErrorDetails.Message } else { "CONNECT_FAILED " + $_.Exception.Message } }
  Remove-Variable Login
}
if ($A) {
  $Q = "filter=" + [uri]::EscapeDataString("userId=""$($A.record.id)""") + "&sort=-created&perPage=10&fields=id,date,type,status,reimbursementAmount,memo"
  Invoke-RestMethod -Uri "$U/api/collections/records/records?$Q" -Headers @{ Authorization = $A.token } -TimeoutSec 20 | ConvertTo-Json -Depth 4
}
```

返回的是 JSON，`items` 里每条记录有日期、类型、状态、金额、记录 ID、用途，按时间倒序。你自己读懂后整理成表格告诉用户。

注意：报销网站对普通员工账号可能不返回记录（包括待审核的），这时 `items` 是空的。**结果为空不代表没有提交过**，不要据此判断“没有重复”，也不要据此重新提交；请用户以网站页面为准。

## 出错时

| 情况 | 怎么处理 |
| --- | --- |
| `NO_ENV` | 告诉用户设置 `FSR_EMAIL` 和 `FSR_PASSWORD` 两个环境变量，然后停下 |
| `LOGIN_FAILED` | 请用户检查这两个环境变量的值，然后停下 |
| `CONNECT_FAILED`、超时、状态码 000 | 请用户连公司内网或 VPN 后重试 |
| `FAILED 400`（比如当天还没有饭否/采购打卡） | 把网站的原话转告用户。网站要求当天先打卡，请用户先去网站打卡，再让你重试 |
| `FAILED 400` 且提到 `receiptUrl` 或长度 | 图片太大，请用户改发截图或更小的图片 |
| `FAILED 401` / `403` | 登录状态被网站拒绝，重新执行一次；还不行就请用户在网站上确认账号正常 |
| `BAD_IMAGE`、`MISSING` | 请用户换一张，或者导出为 JPEG/PNG |
| 找不到 `curl`（极少见） | Windows 请改用 PowerShell 版；Linux 请用户装 curl |
