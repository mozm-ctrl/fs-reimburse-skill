---
name: fs-reimburse
description: Submit reimbursement receipts (meal or purchase) to the company reimbursement site nodelink-intranet (10.1.2.247) using only built-in macOS tools (curl, sips, base64, jq) — nothing to install. Use when the user asks to upload, submit or file a receipt / 小票 / 发票 / 报销凭证, says things like "帮我把这张小票按餐饮报销上传" or "采购报销", or asks about their recent reimbursement records.
---

# 报销上传（fs-reimburse）

把小票图片提交到公司报销网站（nodelink-intranet，PocketBase 后端），生成一条“待审核”记录。金额由管理员审核时填写。

全程只用 macOS 自带的 `curl`、`sips`、`base64`、`jq`，**不需要安装任何程序**。下面的命令请原样执行，只替换尖括号里的内容。

- 网站地址：`http://10.1.2.247`（需要公司内网或 VPN）
- 登录信息保存在：`~/.config/fs-reimburse/auth.json`（权限 600，只保存 token，不保存密码）

## 0. 检查登录

```sh
F=~/.config/fs-reimburse/auth.json
[ -f "$F" ] && jq -r '.user.email' "$F" || echo NOT_LOGGED_IN
```

输出 `NOT_LOGGED_IN` 时，按第 1 步登录；否则记下邮箱，直接进入第 2 步。

## 1. 登录（每台电脑一次，token 过期时重来）

**密码绝对不能经过你（智能体）。** 不要在对话里问密码，也不要把密码写进命令。请用户在**自己的终端**里粘贴运行下面这段（Claude Code 里可以在开头加 `! ` 直接运行）：

```sh
mkdir -p ~/.config/fs-reimburse && chmod 700 ~/.config/fs-reimburse
printf "报销网站邮箱: "; read -r E; printf "密码（不显示，不保存）: "; stty -echo 2>/dev/null; read -r P; stty echo 2>/dev/null; echo
R=$(jq -n --arg i "$E" --arg p "$P" '{identity:$i,password:$p}' \
 | curl -s -X POST http://10.1.2.247/api/collections/users/auth-with-password -H 'Content-Type: application/json' -d @-)
unset P
if echo "$R" | jq -e '.token' >/dev/null 2>&1; then
  echo "$R" | jq '{url:"http://10.1.2.247",token:.token,user:{id:.record.id,email:.record.email,name:.record.name}}' > ~/.config/fs-reimburse/auth.json
  chmod 600 ~/.config/fs-reimburse/auth.json; echo "已登录 $(jq -r .user.email ~/.config/fs-reimburse/auth.json)"
else echo "登录失败：邮箱或密码不对，或者连不上公司内网"; fi
unset R
```

用户回复登录成功后，再执行第 0 步确认一下。

## 2. 确定参数

- **类型**：默认餐饮 `MEAL`。只有用户明确说采购、买东西、办公用品等，才用 `SHOPPING_RECEIPT`，并带上用途（memo）。用户说采购但没说用途时，问一句；用户不想写也可以不带。餐饮不带用途。
- **日期**：固定为今天（`date +%F`）。如果用户说的是以前的小票，提醒他记录日期会是今天。
- **图片**：本地图片路径。一条记录放 1–5 张；超过 5 张时，先问用户是拆成多条，还是只挑 5 张。`sips` 能读取 JPEG/PNG/HEIC/GIF/TIFF/BMP 等格式，HEIC 也可以直接用。

## 3. 提交前确认（必须）

先刷新 token，确认登录还有效（同时也会延长 token 的有效期）：

```sh
F=~/.config/fs-reimburse/auth.json
R=$(curl -s -w '\n%{http_code}' -X POST "$(jq -r .url $F)/api/collections/users/auth-refresh" -H "Authorization: $(jq -r .token $F)")
CODE=$(echo "$R" | tail -1); BODY=$(echo "$R" | sed '$d')
if [ "$CODE" = 200 ]; then echo "$BODY" | jq --arg u "$(jq -r .url $F)" '{url:$u,token:.token,user:{id:.record.id,email:.record.email,name:.record.name}}' > $F.tmp && mv $F.tmp $F && chmod 600 $F && echo "OK $(jq -r .user.email $F)"; else echo "REFRESH_FAILED $CODE"; fi
```

- `REFRESH_FAILED 401`/`403`/`404`：登录已过期，回到第 1 步。
- `REFRESH_FAILED 000`：连不上网站，请用户连公司内网或 VPN。

然后告诉用户：“将以 <邮箱> 提交 <N> 张凭证：<餐饮/采购（用途）>，日期 <今天>。确认提交吗？”用户明确同意后才继续。用户在同一句话里已经说了“直接提交”的，可以跳过确认。

## 4. 提交

把图片压缩成网站前端的格式（最宽 800px、JPEG 质量 60），再提交。把 `TYPE`、`MEMO` 和图片路径换成实际值；餐饮时 `MEMO` 留空：

```sh
F=~/.config/fs-reimburse/auth.json
TYPE=MEAL            # 或 SHOPPING_RECEIPT
MEMO=""              # 采购用途，餐饮留空
T=$(mktemp -d); i=0; OK=1
for IMG in "<图片1>" "<图片2>"; do
  i=$((i+1))
  W=$(sips -g pixelWidth "$IMG" 2>/dev/null | awk '/pixelWidth/{print $2}')
  case "$W" in ''|*[!0-9]*) echo "图片无法读取: $IMG"; OK=0; break;; esac
  if [ "$W" -gt 800 ]; then
    sips -s format jpeg -s formatOptions 60 --resampleWidth 800 "$IMG" --out "$T/$i.jpg" >/dev/null 2>&1
  else
    sips -s format jpeg -s formatOptions 60 "$IMG" --out "$T/$i.jpg" >/dev/null 2>&1
  fi
  [ -s "$T/$i.jpg" ] || { echo "图片转换失败: $IMG"; OK=0; break; }
  { printf 'data:image/jpeg;base64,'; base64 -i "$T/$i.jpg" | tr -d '\n'; } > "$T/$i.txt"
done
if [ $OK = 1 ]; then
  jq -n --arg uid "$(jq -r .user.id $F)" --arg em "$(jq -r .user.email $F)" --arg nm "$(jq -r .user.name $F)" \
    --arg date "$(date +%F)" --arg type "$TYPE" --arg memo "$MEMO" \
    --slurpfile urls <(for f in "$T"/*.txt; do jq -Rs . "$f"; done) \
    '{userId:$uid,userEmail:$em,userName:$nm,date:$date,type:$type,status:"pending",reimbursementAmount:0,receiptUrl:($urls|tojson)}
     + (if $type=="SHOPPING_RECEIPT" and $memo!="" then {memo:$memo} else {} end)' > "$T/body.json"
  CODE=$(curl -s -o "$T/res.json" -w '%{http_code}' -X POST "$(jq -r .url $F)/api/collections/records/records" \
    -H 'Content-Type: application/json' -H "Authorization: $(jq -r .token $F)" -d @"$T/body.json")
  if [ "$CODE" = 200 ]; then jq '{id,date,type,status,memo}' "$T/res.json"
  else echo "提交失败 $CODE: $(jq -r '.message + " " + (.data|tostring)' "$T/res.json" 2>/dev/null || cat "$T/res.json")"; fi
fi
rm -rf "$T"
```

成功时会输出记录的 `id`、`date`、`type`、`status`。把**记录 ID**、类型和日期告诉用户。

同一组图片只提交一次。如果不确定上次有没有成功，先按第 5 步查一下，不要直接重复提交。

## 5. 查询最近记录

```sh
F=~/.config/fs-reimburse/auth.json
curl -s -G "$(jq -r .url $F)/api/collections/records/records" -H "Authorization: $(jq -r .token $F)" \
  --data-urlencode "filter=userId=\"$(jq -r .user.id $F)\"" --data-urlencode "sort=-created" --data-urlencode "perPage=10" \
  | jq -r '.items[] | [.date,.type,.status,.reimbursementAmount,.id,(.memo//"")] | @tsv'
```

## 出错时

| 情况 | 怎么处理 |
| --- | --- |
| 还没登录 / 刷新返回 401、403、404 | 请用户在自己的终端运行第 1 步 |
| 返回码 000、连接超时 | 请用户连公司内网或 VPN 后重试 |
| 提交失败 400（比如当天还没有饭否/采购打卡） | 把网站的原话转告用户。网站要求当天先打卡，请用户先去网站打卡，再让你重试 |
| 图片无法读取 | 请用户换一张，或者导出为 JPEG/PNG |

规则：
- 不要打印 token。
- 不要修改 `auth.json` 里的内容（第 1 步和第 3 步按上面的命令写入的除外）。
- 用户要退出登录时，执行 `rm ~/.config/fs-reimburse/auth.json`。
