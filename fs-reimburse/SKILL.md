---
name: fs-reimburse
description: Submit reimbursement receipts (meal or purchase) to the company reimbursement site nodelink-intranet (10.1.2.247) with the single-file `fsr` CLI, which works the same on macOS, Linux and Windows (PowerShell, cmd, Git Bash) and needs no other tools. Use when the user asks to upload, submit or file a receipt / 小票 / 发票 / 报销凭证, says things like "帮我把这张小票按餐饮报销上传" or "采购报销", or asks about their recent reimbursement records.
---

# 报销上传（fs-reimburse）

用 `fsr` 命令把小票图片提交到公司报销网站（nodelink-intranet，`http://10.1.2.247`，需要公司内网或 VPN），生成一条“待审核”记录。金额由管理员审核时填写，不需要你填。

`fsr` 是一个单文件程序，macOS、Linux、Windows 上的命令完全一样（PowerShell、cmd、Git Bash 里都直接写 `fsr ...`）。图片压缩、HEIC 转换、登录（读取环境变量 `FSR_EMAIL` / `FSR_PASSWORD`）都在 `fsr` 内部完成，不需要 `sips`、`jq`、`base64` 等工具。

## 1. 确认装了 fsr

运行 `fsr version`。找不到命令时，直接替用户安装（不需要密码，也不需要管理员权限）：

- macOS / Linux / Git Bash：
  ```sh
  curl -fsSL https://raw.githubusercontent.com/mozm-ctrl/fs-reimburse-skill/main/install.sh | sh
  ```
- Windows PowerShell：
  ```powershell
  irm https://raw.githubusercontent.com/mozm-ctrl/fs-reimburse-skill/main/install.ps1 | iex
  ```

装好后如果当前终端还找不到 `fsr`，用安装脚本最后打印的完整路径运行（macOS / Linux 是 `~/.local/bin/fsr`，Windows 是 `%LOCALAPPDATA%\fsr\fsr.exe`）。

## 2. 确定参数

- **类型**：默认是餐饮（`--type meal`）。只有用户明确说是采购、买东西、办公用品等，才用采购（`--type purchase`），并带上用途 `--memo "<用途>"`。用户说采购但没说用途时，问一句用途；用户不想写也可以不带。餐饮不带用途。
- **日期**：自动取今天，不能改。如果用户说的是以前的小票，提醒他记录日期会是今天。
- **图片**：用户给的本地图片路径。一条记录 1–5 张；超过 5 张时，先问用户是拆成多条，还是只挑 5 张。支持 JPEG / PNG / HEIC / GIF / WebP / BMP / TIFF，iPhone 的 HEIC 可以直接用。一张图里有几张小票、或者多张图属于同一笔报销，都放在同一条记录里。

## 3. 预检，并请用户确认（必须）

不用事先检查登录状态，直接预检。`fsr` 每次运行都会用环境变量里的账号登录；预检会登录、检查图片，但不会提交：

```sh
fsr upload --dry-run --type <meal|purchase> [--memo "<用途>"] <图片...>
```

- 成功时输出类似 `[dry-run] 将以 zhangsan@nodelink.it 提交 1 张凭证（餐饮，2026-11-12），未实际提交`。
- 报“没有设置报销网站账号”或“登录失败”：按第 4 步处理，然后**停下**，等用户设置好再重新预检。

预检成功后告诉用户：“将以 <邮箱> 提交 <N> 张凭证：<餐饮/采购（用途）>，日期 <今天>。确认提交吗？”用户明确同意后才继续。用户在同一句话里已经说了“直接提交，不用确认”的，可以跳过确认，但预检仍然要做。

## 4. 账号没设置或不对时

`fsr` 只从环境变量读取账号：`FSR_EMAIL`（邮箱）和 `FSR_PASSWORD`（密码），旧写法 `yx` / `mm` 也支持。没有交互式登录，也不保存 token。

- **没有设置**：告诉用户需要在智能体的运行环境里设置 `FSR_EMAIL` 和 `FSR_PASSWORD` 两个环境变量（Multica：智能体设置 → 环境变量；本机终端：见 README 的“设置账号”一节），然后停下。
- **登录失败（网站拒绝了邮箱和密码）**：请用户检查这两个变量的值，然后停下。

**密码绝对不能经过你。** 不要在对话里问密码，不要替用户设置密码，不要把密码写进任何命令、输出、issue 或 PR，也不要打印或复述这两个变量的值。

## 5. 提交

```sh
fsr upload --json --type meal <图片...>
fsr upload --json --type purchase --memo "办公用品" <图片...>
```

成功时输出 JSON，包含 `id`（记录 ID）、`date`、`type`、`status`。把记录 ID、类型和日期告诉用户。

同一组图片只提交一次。如果不确定上次有没有成功，先问用户去网站上看一下，不要直接重复提交。

## 6. 查询最近记录

```sh
fsr records -n 10          # 每行：日期、类型、状态、金额、记录 ID、用途
fsr records -n 10 --json
```

注意：报销网站对普通员工账号可能不返回记录（包括待审核的），这时结果是空的。**结果为空不代表没有提交过**，不要据此判断“没有重复”，也不要据此重新提交；请用户以网站页面为准。

## 出错时

| 错误信息 | 怎么处理 |
| --- | --- |
| 找不到 `fsr` 命令 | 按第 1 步安装 |
| 没有设置报销网站账号 | 按第 4 步，告诉用户设置 `FSR_EMAIL` 和 `FSR_PASSWORD`，然后停下 |
| 登录失败：网站拒绝了邮箱…和这个密码 | 请用户检查这两个环境变量的值，然后停下 |
| 连不上报销网站 | 请用户连上公司内网或 VPN 后重试 |
| 报销网站返回 400 …（比如当天还没有饭否/采购打卡） | 把网站原话转告用户。网站要求当天先打卡，请用户先去网站打卡，再让你重试 |
| 无法识别的图片格式 | 请用户换一张，或者导出为 JPEG/PNG |
| 一条记录需要 1-5 张图片 | 按第 2 步拆分 |

规则：
- 不要打印 token、密码或环境变量的值。
