# fs-reimburse skill

让 AI 智能体（Claude Code、Multica 等）把小票提交到公司报销网站（nodelink-intranet，`http://10.1.2.247`）。

skill 通过一个单文件命令行程序 `fsr` 完成所有操作：登录、图片压缩（含 iPhone 的 HEIC）、提交和查询。macOS、Linux、Windows（PowerShell、cmd、Git Bash）都能用，**除了 `fsr` 本身，不需要安装任何东西**（不需要 `jq`、`sips`、Go 或 Python）。

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

## 3. 登录（二选一）

不需要提前登录，智能体在需要时会处理。

- **环境变量（推荐，不用碰终端）**：在 Multica 的智能体设置 → 环境变量里添加 `yx`（报销网站邮箱）和 `mm`（密码）。`fsr` 每次运行都会用它们重新登录，不在本机保存任何登录文件。旧写法 `FSR_EMAIL`、`FSR_PASSWORD` 也支持。
- **终端**：没有配置环境变量时，智能体会请你在自己的终端里运行一次：

  | 环境 | 命令 |
  | --- | --- |
  | macOS / Linux 终端、PowerShell、cmd | `fsr login` |
  | Claude Code 对话框 | `! fsr login` |
  | Windows Git Bash | `winpty fsr login`（Git Bash 不能隐藏密码输入，需要加 `winpty`；或者改在 PowerShell 里运行） |

  按提示输入邮箱和密码。密码不会经过智能体，也不会被保存；只在本机保存登录 token（`~/.config/fs-reimburse/auth.json`，Windows 是 `%USERPROFILE%\.config\fs-reimburse\auth.json`），过期后智能体会请你重新登录。退出登录：`fsr logout`。

## 使用

对智能体说：

- “帮我把这张小票按餐饮报销上传” + 图片
- “这两张是采购，用途是办公用品，帮我报销”
- “看看我最近的报销记录”

智能体会先预检，向你确认账号、类型、日期和张数，你同意后才提交，最后告诉你记录 ID。

也可以自己直接用（所有系统命令相同）：

```sh
fsr whoami                                              # 当前登录的账号
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
| `yx` / `mm` | 报销网站邮箱 / 密码，设置后每次运行自动登录 |
| `INTRANET_URL` | 报销网站地址，默认 `http://10.1.2.247` |
| `FSR_CONFIG` | 终端登录时 token 文件的路径 |

## 开发

`fsr/` 是 Go 模块，不需要 CGO，HEIC 用内嵌的 WASM 解码器（[gen2brain/heic](https://github.com/gen2brain/heic)）。

```sh
cd fsr
go test ./...
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -tags nodynamic -o fsr.exe .   # 交叉编译
```

发布：运行 `sh fsr/release.sh v0.1.0`（需要 Go 1.26+ 和已登录的 `gh`），会编译 macOS / Linux / Windows（amd64 和 arm64）的版本并发布到 Releases，安装脚本从最新的 Release 下载。
