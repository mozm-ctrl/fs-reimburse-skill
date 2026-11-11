# fs-reimburse skill

让 AI 智能体（Claude Code、Multica 等）把小票提交到公司报销网站（nodelink-intranet）。只用 macOS 自带的 `curl`、`sips`、`base64`、`jq`，**不需要安装任何程序**。

## 使用

**Multica**：在工作区的 Skills 页面选择“从 URL 导入”，填：

```
https://github.com/mozm-ctrl/fs-reimburse-skill/tree/main/fs-reimburse
```

然后把它绑定到运行在你自己电脑上的智能体。

**Claude Code**：把 `fs-reimburse` 文件夹复制到 `~/.claude/skills/`。

第一次报销时，智能体会给你一段登录命令，在自己的终端里运行一次即可。密码不会经过智能体，也不会被保存；登录 token 只存在本机 `~/.config/fs-reimburse/auth.json`。

前提：macOS，电脑能访问公司内网。
