# fs-reimburse skill

让 AI 智能体（Claude Code、Multica 等）把小票提交到公司报销网站（nodelink-intranet，`http://10.1.2.247`）。

skill 通过一个单文件命令行程序 `fsr` 完成所有操作：登录、图片压缩（含 iPhone 的 HEIC）、提交和查询。macOS、Linux、Windows（PowerShell、cmd、Git Bash）都能用。**需要安装的只有 `fsr` 这一个文件**（不需要 `jq`、`sips`、Go 或 Python），而且智能体第一次用时发现没装会自己装。

前提：电脑能访问公司内网（在外面请先连 VPN）。

## 1. 安装 fsr

在智能体运行的那台电脑上装一次。不需要管理员权限。第一次用 skill 时，智能体发现没装也会自己执行这一步。

**macOS / Linux**（终端）：

```sh
curl -fsSL https://raw.githubusercontent.com/mozm-ctrl/fs-reimburse-skill/main/install.sh | sh
```

装到 `~/.local/bin/fsr`。如果提示 `~/.local/bin` 不在 PATH 里，按提示把它加到 `~/.zshrc` 或 `~/.bashrc`。

**Windows PowerShell**：

```powershell
irm https://raw.githubusercontent.com/mozm-ctrl/fs-reimburse-skill/main/install.ps1 | iex
```

装到 `%LOCALAPPDATA%\fsr\fsr.exe`，并加入用户 PATH，重新打开终端后生效。PowerShell、cmd 和 Git Bash 都能用这个 `fsr`。

**Windows Git Bash**：运行上面 macOS / Linux 的那条命令即可，它会自动调用 PowerShell 安装脚本，结果和在 PowerShell 里装的一样。

**手动安装**：在 [Releases](https://github.com/mozm-ctrl/fs-reimburse-skill/releases) 下载对应系统的文件（例如 `fsr-windows-amd64.exe`、`fsr-darwin-arm64`、`fsr-linux-amd64`），改名为 `fsr`（Windows 为 `fsr.exe`），放进 PATH 里的任意目录；macOS / Linux 再执行 `chmod +x fsr`。装了 Go 1.26+ 的也可以运行 `go install github.com/mozm-ctrl/fs-reimburse-skill/fsr@latest`。

## 2. 安装 skill

**Multica**：在工作区的 Skills 页面选择“从 URL 导入”，填：

```
https://github.com/mozm-ctrl/fs-reimburse-skill/tree/main/fs-reimburse
```

然后把它绑定到运行在你自己电脑上的智能体。

**Claude Code**：把 `fs-reimburse` 文件夹复制到 `~/.claude/skills/`（Windows 是 `%USERPROFILE%\.claude\skills\`）。

## 3. 设置账号（环境变量）

`fsr` 每次运行都用两个环境变量直接登录报销网站，不需要在终端里输入密码，也不会在本机保存 token 或任何登录文件：

| 变量 | 内容 |
| --- | --- |
| `FSR_EMAIL` | 报销网站邮箱 |
| `FSR_PASSWORD` | 报销网站密码 |

（之前用过的 `yx` / `mm` 也仍然支持；两组都设置时以 `FSR_EMAIL` / `FSR_PASSWORD` 为准。）

没设置时，`fsr` 会报“没有设置报销网站账号”，智能体会告诉你要设置哪两个变量，然后停下。智能体不会在对话里问你的密码；密码也不会出现在命令行参数、输出或日志里（`fsr` 在程序内部读取环境变量）。

**在智能体运行环境里配置（推荐）**：

- **Multica**：智能体设置 → 环境变量，添加 `FSR_EMAIL` 和 `FSR_PASSWORD`，保存即可，下一次运行生效。
- **Claude Code** 等直接在本机终端里启动的智能体：按下面各系统的方法设置，然后重新打开终端、重新启动智能体。

**macOS / Linux**：把下面两行加到 shell 配置文件（macOS 默认是 `~/.zshrc`，Linux 一般是 `~/.bashrc`），然后重新打开终端：

```sh
export FSR_EMAIL="zhangsan@nodelink.it"
export FSR_PASSWORD="你的密码"
```

建议执行 `chmod 600 ~/.zshrc`（或 `~/.bashrc`），只让自己能读。

**Windows（PowerShell、cmd、Git Bash 通用）**：设置为用户环境变量。在 PowerShell 里运行下面的命令，它会提示你输入密码，输入时不显示，也不会留在命令历史里：

```powershell
[Environment]::SetEnvironmentVariable("FSR_EMAIL", "zhangsan@nodelink.it", "User")
[Environment]::SetEnvironmentVariable("FSR_PASSWORD", [Net.NetworkCredential]::new("", (Read-Host "报销网站密码" -AsSecureString)).Password, "User")
```

也可以用图形界面：开始菜单搜索“编辑账户的环境变量”，在“用户变量”里新建这两个变量。设置后重新打开终端（包括 Git Bash）和智能体才会生效。

## 使用

对智能体说：

- “帮我把这张小票按餐饮报销上传” + 图片
- “这两张是采购，用途是办公用品，帮我报销”
- “看看我最近的报销记录”

智能体会先预检，向你确认账号、类型、日期和张数，你同意后才提交，最后告诉你记录 ID。

也可以自己直接用（所有系统命令相同）：

```sh
fsr whoami                                              # 检查账号能否登录
fsr upload --dry-run receipt.heic                       # 只检查，不提交
fsr upload receipt.jpg                                  # 餐饮，日期为今天
fsr upload --type purchase --memo "办公用品" a.jpg b.png  # 采购，最多 5 张
fsr records -n 10                                       # 最近的记录
```

`upload` 和 `records` 加 `--json` 会输出 JSON。

规则：

- 类型：`meal`（餐饮，默认）或 `purchase`（采购，可以带用途 `--memo`）
- 日期固定为当天；网站要求当天先有饭否/采购打卡，否则会拒绝，`fsr` 会原样显示网站的报错
- 一条记录 1–5 张图；支持 JPEG / PNG / HEIC / GIF / WebP / BMP / TIFF，手机照片的方向会按 EXIF 自动转正；图片按网站前端的格式压缩（最宽 800px、JPEG 质量 60）
- 已知问题：报销网站对普通员工账号可能不返回记录，`fsr records` 会是空的。结果为空不代表没提交过，请以网站页面为准

| 环境变量 | 说明 |
| --- | --- |
| `FSR_EMAIL` / `FSR_PASSWORD` | 报销网站邮箱 / 密码，必填（也可以用 `yx` / `mm`） |
| `INTRANET_URL` | 报销网站地址，默认 `http://10.1.2.247` |

## 开发

`fsr/` 是 Go 模块，不需要 CGO，HEIC 用内嵌的 WASM 解码器（[gen2brain/heic](https://github.com/gen2brain/heic)）。

```sh
cd fsr
go test ./...
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -tags nodynamic -o fsr.exe .   # 交叉编译
```

发布：运行 `sh fsr/release.sh v0.1.0`（需要 Go 1.26+ 和已登录的 `gh`），会编译 macOS / Linux / Windows（amd64 和 arm64）的版本并发布到 Releases，安装脚本从最新的 Release 下载。
