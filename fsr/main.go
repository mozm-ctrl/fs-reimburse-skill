package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/term"
)

// fsr: submit reimbursement receipts to nodelink-intranet from the command line.
// Only the login token is stored, in the user's own config dir with mode 0600; passwords never are.
const usage = `usage: fsr <command> [flags]
  login [--email E] [--password-stdin] [--url U]   log in once and save the token locally
  whoami                                           show the logged-in account
  upload [--type meal|purchase] [--memo M] [--dry-run] [--json] <image>...
                                                   submit 1-5 images as one record dated today
  records [-n N] [--json]                          your most recent records
  logout                                           delete the saved token
  version

Environment: yx / mm (or FSR_EMAIL / FSR_PASSWORD) log in on every run without a token file;
FSR_CONFIG overrides the token file path; INTRANET_URL overrides the site URL.`

var version = "dev"

type config struct {
	URL   string `json:"url"`
	Token string `json:"token"`
	User  User   `json:"user"`
}

func configPath() string {
	if p := os.Getenv("FSR_CONFIG"); p != "" {
		return p
	}
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "fs-reimburse", "auth.json")
}

func loadConfig() (*config, error) {
	b, err := os.ReadFile(configPath())
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("还没有登录，请在自己的终端运行：fsr login")
	}
	if err != nil {
		return nil, err
	}
	var c config
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("%s 已损坏，请重新运行 fsr login: %w", configPath(), err)
	}
	if env := os.Getenv("INTRANET_URL"); env != "" {
		c.URL = env
	}
	return &c, nil
}

// saveConfig writes atomically with 0600 so a crash never leaves a half-written token file.
func saveConfig(c *config) error {
	p := configPath()
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(c, "", "  ")
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

func siteURL() string {
	if u := os.Getenv("INTRANET_URL"); u != "" {
		return strings.TrimRight(u, "/")
	}
	return defaultURL
}

// envCredentials returns the account configured in the agent's environment, if any.
func envCredentials() (email, password string, ok bool) {
	email, password = os.Getenv("yx"), os.Getenv("mm")
	if email == "" || password == "" {
		email, password = os.Getenv("FSR_EMAIL"), os.Getenv("FSR_PASSWORD")
	}
	return email, password, email != "" && password != ""
}

// session logs in when needed. With credentials in the environment it logs in fresh on every
// run and never touches the token file; otherwise it refreshes the token saved by fsr login.
func session() (*config, *Auth, error) {
	if email, password, ok := envCredentials(); ok {
		base := siteURL()
		a, err := Login(base, email, password)
		if err != nil {
			return nil, nil, fmt.Errorf("用环境变量 yx/mm 登录失败（请检查智能体设置里的邮箱和密码）: %w", err)
		}
		return &config{URL: base, Token: a.Token, User: a.Record}, a, nil
	}
	c, err := loadConfig()
	if err != nil {
		return nil, nil, err
	}
	a, err := Refresh(c.URL, c.Token)
	if err != nil {
		return nil, nil, err
	}
	c.Token, c.User = a.Token, a.Record
	if err := saveConfig(c); err != nil {
		return nil, nil, err
	}
	return c, a, nil
}

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		fail(usage)
	}
	var err error
	switch args[0] {
	case "login":
		err = cmdLogin(args[1:])
	case "whoami":
		err = cmdWhoami()
	case "upload":
		err = cmdUpload(args[1:])
	case "records":
		err = cmdRecords(args[1:])
	case "logout":
		err = cmdLogout()
	case "version", "--version":
		fmt.Println("fsr " + version)
	case "help", "-h", "--help":
		fmt.Println(usage)
	default:
		fail(usage)
	}
	if err != nil {
		fail("错误：" + err.Error())
	}
}

func fail(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}

func cmdLogin(args []string) error {
	fs := flag.NewFlagSet("login", flag.ExitOnError)
	email := fs.String("email", "", "intranet account email")
	fromStdin := fs.Bool("password-stdin", false, "read the password from stdin instead of prompting")
	base := fs.String("url", "", "site URL (default "+defaultURL+")")
	fs.Parse(args)
	if *base == "" {
		*base = siteURL()
	}
	*base = strings.TrimRight(*base, "/")

	in := bufio.NewReader(os.Stdin)
	if *email == "" {
		fmt.Fprint(os.Stderr, "邮箱: ")
		line, _ := in.ReadString('\n')
		*email = strings.TrimSpace(line)
	}
	var password string
	switch {
	case *fromStdin:
		line, _ := in.ReadString('\n')
		password = strings.TrimRight(line, "\r\n")
	case term.IsTerminal(int(os.Stdin.Fd())):
		fmt.Fprint(os.Stderr, "密码（不会显示，也不会保存）: ")
		b, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return err
		}
		password = string(b)
	default:
		// Git Bash (mintty) is not a console to Go, so it can't hide typed input; see README.
		return fmt.Errorf("这个终端无法隐藏密码输入；Git Bash 请用 winpty fsr login，或按 README 用 --password-stdin")
	}
	if *email == "" || password == "" {
		return fmt.Errorf("邮箱和密码都不能为空")
	}
	a, err := Login(*base, *email, password)
	if err != nil {
		return err
	}
	if err := saveConfig(&config{URL: *base, Token: a.Token, User: a.Record}); err != nil {
		return err
	}
	fmt.Printf("已登录 %s（%s），token 保存在 %s\n", a.Record.Email, a.Record.Name, configPath())
	return nil
}

func cmdWhoami() error {
	c, _, err := session()
	if err != nil {
		return err
	}
	fmt.Printf("%s\t%s\t%s\n", c.User.Email, c.User.Name, c.URL)
	return nil
}

func cmdLogout() error {
	err := os.Remove(configPath())
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	fmt.Println("已退出登录，本机 token 已删除")
	return nil
}

var types = map[string]string{
	"meal": "MEAL", "餐饮": "MEAL", "MEAL": "MEAL",
	"purchase": "SHOPPING_RECEIPT", "采购": "SHOPPING_RECEIPT", "SHOPPING_RECEIPT": "SHOPPING_RECEIPT",
}

func cmdUpload(args []string) error {
	fs := flag.NewFlagSet("upload", flag.ExitOnError)
	typFlag := fs.String("type", "meal", "meal (餐饮) or purchase (采购)")
	memo := fs.String("memo", "", "purpose, for purchase only")
	dry := fs.Bool("dry-run", false, "check login and images without submitting")
	asJSON := fs.Bool("json", false, "print the result as JSON")
	fs.Parse(args)
	images := fs.Args()
	typ, ok := types[*typFlag]
	if !ok {
		return fmt.Errorf("--type 只能是 meal 或 purchase")
	}
	if len(images) == 0 || len(images) > 5 {
		return fmt.Errorf("一条记录需要 1-5 张图片，当前 %d 张", len(images))
	}
	if typ == "MEAL" && *memo != "" {
		return fmt.Errorf("餐饮报销不带用途；如果是采购请加 --type purchase")
	}

	// Encode first: a bad image should fail before touching the site.
	var urls []string
	for _, f := range images {
		b, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		u, err := ToDataURL(b)
		if err != nil {
			return fmt.Errorf("%s: %w", f, err)
		}
		urls = append(urls, u)
	}
	c, a, err := session()
	if err != nil {
		return err
	}
	date := time.Now().Format("2006-01-02")
	label := map[string]string{"MEAL": "餐饮", "SHOPPING_RECEIPT": "采购"}[typ]

	if *dry {
		if *asJSON {
			return printJSON(map[string]any{"dryRun": true, "email": c.User.Email, "type": typ, "memo": *memo, "date": date, "images": len(urls)})
		}
		fmt.Printf("[dry-run] 将以 %s 提交 %d 张凭证（%s，%s", c.User.Email, len(urls), label, date)
		if *memo != "" {
			fmt.Printf("，用途：%s", *memo)
		}
		fmt.Println("），未实际提交")
		return nil
	}
	rec, err := CreateRecord(c.URL, a, urls, typ, *memo, date)
	if err != nil {
		return err
	}
	if *asJSON {
		return printJSON(map[string]any{"id": rec.ID, "email": c.User.Email, "type": typ, "memo": rec.Memo, "date": rec.Date, "status": rec.Status, "images": len(urls)})
	}
	fmt.Printf("已上传 %d 张凭证（%s，%s，%s），记录 ID：%s\n", len(urls), label, rec.Date, c.User.Email, rec.ID)
	return nil
}

func cmdRecords(args []string) error {
	fs := flag.NewFlagSet("records", flag.ExitOnError)
	n := fs.Int("n", 10, "number of records")
	asJSON := fs.Bool("json", false, "print as JSON")
	fs.Parse(args)
	if *n < 1 || *n > 200 {
		*n = 10
	}
	c, a, err := session()
	if err != nil {
		return err
	}
	recs, err := ListRecords(c.URL, a, *n)
	if err != nil {
		return err
	}
	if *asJSON {
		if recs == nil {
			recs = []Record{}
		}
		return printJSON(recs)
	}
	if len(recs) == 0 {
		fmt.Fprintf(os.Stderr, "网站没有返回 %s 的任何记录（这不代表一定没有提交过，请以网站页面为准）\n", c.User.Email)
	}
	for _, r := range recs {
		fmt.Println(strings.Join([]string{r.Date, r.Type, r.Status, fmt.Sprint(r.ReimbursementAmount), r.ID, r.Memo}, "\t"))
	}
	return nil
}

func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
