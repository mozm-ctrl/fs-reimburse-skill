# fs-reimburse skill

让 AI 智能体（Multica、Claude Code 等）把小票提交到公司报销网站（nodelink-intranet，`http://10.1.2.247`）。

整个 skill 只有一个 `SKILL.md`，**不需要安装任何程序，也不需要在终端里输入密码**。macOS、Linux、Windows 都能用。

前提：智能体运行的那台电脑能访问公司内网（在外面请先连 VPN）。

## 在 Multica 里使用（两步）

**第 1 步：导入 skill**

在工作区的 Skills 页面选择“从 URL 导入”，填：

```
https://github.com/mozm-ctrl/fs-reimburse-skill/tree/main/fs-reimburse
```

然后把它绑定到运行在你自己电脑上的智能体。之前导入过旧版本的，刷新或重新导入一次。

**第 2 步：填账号密码**

打开智能体设置 → 环境变量，添加两项，保存：

| 变量名 | 填什么 |
| --- | --- |
| `FSR_EMAIL` | 报销网站邮箱 |
| `FSR_PASSWORD` | 报销网站密码 |

之前配过 `yx` / `mm` 的可以继续用，不用再加。

**就这样，配好了。** 不用打开终端，电脑上也不会多出任何文件。智能体每次报销都会自己读这两个变量登录。没配置时，智能体会提醒你配置，不会在对话里问你密码。

## 怎么报销

把小票图片发给智能体，说一句：

- “帮我把这张小票按餐饮报销上传”
- “这两张是采购，用途是办公用品，帮我报销”
- “看看我最近的报销记录”

智能体收到图片和类型后会直接提交，不再向你确认，完成后告诉你记录 ID。采购没说用途时不会追问，直接不带用途提交。

规则：

- 类型：餐饮（默认）或采购（可以带用途）
- 日期固定为当天；网站要求当天先有饭否/采购打卡，否则会拒绝，智能体会转告网站的原话
- 一条记录 1–5 张图；支持 JPEG / PNG / WebP / GIF。iPhone 的 HEIC 网站显示不了，请发截图或导出为 JPEG。单张最好不超过 3 MB
- 已知问题：报销网站对普通员工账号可能不返回记录，查询结果会是空的。结果为空不代表没提交过，请以网站页面为准

---

## 其他智能体（Claude Code 等）

**用 Multica 的可以跳过这一节。**

Claude Code 这类直接在终端里启动的智能体没有“环境变量”设置页面，需要把 `FSR_EMAIL` / `FSR_PASSWORD` 设到系统里：

**安装 skill**：把 `fs-reimburse` 文件夹复制到 `~/.claude/skills/`（Windows 是 `%USERPROFILE%\.claude\skills\`）。

**macOS / Linux**：把下面两行加到 shell 配置文件（macOS 默认是 `~/.zshrc`，Linux 一般是 `~/.bashrc`），然后重新打开终端。密码用单引号括起来，避免 `$` 等符号被 shell 解释：

```sh
export FSR_EMAIL='zhangsan@nodelink.it'
export FSR_PASSWORD='你的密码'
```

注意这样密码会保存在这个文件里，建议执行 `chmod 600 ~/.zshrc`（或 `~/.bashrc`），只让自己能读。

**Windows（PowerShell、cmd、Git Bash 通用）**：设置为用户环境变量。在 PowerShell 里运行下面的命令，它会提示你输入密码，输入时不显示，也不会留在命令历史里：

```powershell
[Environment]::SetEnvironmentVariable("FSR_EMAIL", "zhangsan@nodelink.it", "User")
[Environment]::SetEnvironmentVariable("FSR_PASSWORD", [Net.NetworkCredential]::new("", (Read-Host "报销网站密码" -AsSecureString)).Password, "User")
```

也可以用图形界面：开始菜单搜索“编辑账户的环境变量”，在“用户变量”里新建这两个变量。设置后要重新打开终端和智能体才会生效。

## 技术说明

- `SKILL.md` 里每一步都有两套命令：sh 版（macOS、Linux、Windows Git Bash，只用系统自带的 `curl`、`base64`、`sed`、`tr`、`date`）和 PowerShell 版（Windows PowerShell 5.1 或 7）。不需要 `jq`、`sips`、Python。
- 每次运行都重新登录，登录凭证只在内存里，不写文件。命令只在内部读取环境变量，密码不会出现在命令行参数、输出或日志里。
- 两组变量都设置时，以 `FSR_EMAIL` / `FSR_PASSWORD` 为准。

| 环境变量 | 说明 |
| --- | --- |
| `FSR_EMAIL` / `FSR_PASSWORD` | 报销网站邮箱 / 密码，必填（也可以用 `yx` / `mm`） |
| `INTRANET_URL` | 报销网站地址，可选，默认 `http://10.1.2.247` |
