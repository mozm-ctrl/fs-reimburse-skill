package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

// fsr: submit reimbursement receipts to nodelink-intranet from the command line.
// It logs in with the account in the environment on every run and stores nothing on disk.
const usage = `usage: fsr <command> [flags]
  whoami                                           check the login and show the account
  upload [--type meal|purchase] [--memo M] [--dry-run] [--json] <image>...
                                                   submit 1-5 images as one record dated today
  records [-n N] [--json]                          your most recent records
  version

Environment: FSR_EMAIL / FSR_PASSWORD (or yx / mm) are the site account, required;
INTRANET_URL overrides the site URL.`

var version = "dev"

type config struct {
	URL  string
	User User
}

func siteURL() string {
	if u := os.Getenv("INTRANET_URL"); u != "" {
		return strings.TrimRight(u, "/")
	}
	return defaultURL
}

// envCredentials returns the account configured in the agent's environment, if any.
func envCredentials() (email, password string, ok bool) {
	email, password = os.Getenv("FSR_EMAIL"), os.Getenv("FSR_PASSWORD")
	if email == "" || password == "" {
		email, password = os.Getenv("yx"), os.Getenv("mm")
	}
	return email, password, email != "" && password != ""
}

var errNoEnv = fmt.Errorf("没有设置报销网站账号。请设置环境变量 FSR_EMAIL（邮箱）和 FSR_PASSWORD（密码），设置方法见 README")

// session logs in fresh with the environment account; the token lives only in memory.
func session() (*config, *Auth, error) {
	email, password, ok := envCredentials()
	if !ok {
		return nil, nil, errNoEnv
	}
	base := siteURL()
	a, err := Login(base, email, password)
	if err != nil {
		return nil, nil, err
	}
	return &config{URL: base, User: a.Record}, a, nil
}

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		fail(usage)
	}
	var err error
	switch args[0] {
	case "whoami":
		err = cmdWhoami()
	case "upload":
		err = cmdUpload(args[1:])
	case "records":
		err = cmdRecords(args[1:])
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

func cmdWhoami() error {
	c, _, err := session()
	if err != nil {
		return err
	}
	fmt.Printf("%s\t%s\t%s\n", c.User.Email, c.User.Name, c.URL)
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
