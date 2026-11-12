# fs-reimburse skill

让 AI 智能体（Multica、Claude Code 等）把小票提交到公司报销网站（nodelink-intranet，`http://10.1.2.247`）。

整个 skill 只有一个 `SKILL.md`，**不需要安装任何程序**。里面写好了每一步要执行的命令，智能体照着执行：

- macOS、Linux、Windows Git Bash：用 sh 版命令，只用到系统自带的 `curl`、`base64`、`sed`、`tr`、`date`
- Windows PowerShell（5.1 或 7）：用 PowerShell 版命令，只用 PowerShell 自带的功能

不需要 `jq`、`sips`、Python 或其他工具。图片原样上传，不做压缩。

前提：智能体运行的那台电脑能访问公司内网（在外面请先连 VPN）。

## 1. 安装 skill

**Multica**：在工作区的 Skills 页面选择“从 URL 导入”，填：

```
https://github.com/mozm-ctrl/fs-reimburse-skill/tree/main/fs-reimburse
```

也可以新建一个 skill，把 `fs-reimburse/SKILL.md` 的内容整个复制进去。然后把它绑定到运行在你自己电脑上的智能体。

**Claude Code**：把 `fs-reimburse` 文件夹复制到 `~/.claude/skills/`（Windows 是 `%USERPROFILE%\.claude\skills\`）。

## 2. 设置账号（环境变量）

每次使用时，智能体都会用两个环境变量重新登录报销网站。不需要在终端里输入密码，也不会在本机保存 token 或任何登录文件：

| 变量 | 内容 |
| --- | --- |
| `FSR_EMAIL` | 报销网站邮箱 |
| `FSR_PASSWORD` | 报销网站密码 |

之前用过的 `yx` / `mm` 也仍然支持；两组都设置时以 `FSR_EMAIL` / `FSR_PASSWORD` 为准。

没设置时，智能体会告诉你要设置哪两个变量，然后停下，不会在对话里问你的密码。命令只在内部读取环境变量，密码不会出现在命令行参数、输出或日志里。

**在智能体运行环境里配置（推荐）**：

- **Multica**：智能体设置 → 环境变量，添加 `FSR_EMAIL` 和 `FSR_PASSWORD`，保存即可，下一次运行生效。
- **Claude Code** 等在本机终端里启动的智能体：按下面各系统的方法设置，然后重新打开终端、重新启动智能体。

**macOS / Linux**：把下面两行加到 shell 配置文件（macOS 默认是 `~/.zshrc`，Linux 一般是 `~/.bashrc`），然后重新打开终端。密码用单引号括起来，避免 `$` 等符号被 shell 解释：

```sh
export FSR_EMAIL='zhangsan@nodelink.it'
export FSR_PASSWORD='你的密码'
```

建议执行 `chmod 600 ~/.zshrc`（或 `~/.bashrc`），只让自己能读。

**Windows（PowerShell、cmd、Git Bash 通用）**：设置为用户环境变量。在 PowerShell 里运行下面的命令，它会提示你输入密码，输入时不显示，也不会留在命令历史里：

```powershell
[Environment]::SetEnvironmentVariable("FSR_EMAIL", "zhangsan@nodelink.it", "User")
[Environment]::SetEnvironmentVariable("FSR_PASSWORD", [Net.NetworkCredential]::new("", (Read-Host "报销网站密码" -AsSecureString)).Password, "User")
```

也可以用图形界面：开始菜单搜索“编辑账户的环境变量”，在“用户变量”里新建这两个变量。设置后要重新打开终端（包括 Git Bash）和智能体才会生效。

## 使用

对智能体说：

- “帮我把这张小票按餐饮报销上传” + 图片
- “这两张是采购，用途是办公用品，帮我报销”
- “看看我最近的报销记录”

智能体会先预检（登录并检查图片，不提交），向你确认账号、类型、日期和张数，你同意后才提交，最后告诉你记录 ID。

规则：

- 类型：餐饮（默认）或采购（可以带用途）
- 日期固定为当天；网站要求当天先有饭否/采购打卡，否则会拒绝，智能体会转告网站的原话
- 一条记录 1–5 张图；支持 JPEG / PNG / WebP / GIF。iPhone 的 HEIC 网站显示不了，请发截图或导出为 JPEG。图片原样上传，单张最好不超过 3 MB
- 已知问题：报销网站对普通员工账号可能不返回记录，查询结果会是空的。结果为空不代表没提交过，请以网站页面为准

| 环境变量 | 说明 |
| --- | --- |
| `FSR_EMAIL` / `FSR_PASSWORD` | 报销网站邮箱 / 密码，必填（也可以用 `yx` / `mm`） |
| `INTRANET_URL` | 报销网站地址，可选，默认 `http://10.1.2.247` |
