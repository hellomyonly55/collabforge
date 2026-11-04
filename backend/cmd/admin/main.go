// Command admin creates a new administrator without a default password or public promotion API.
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"golang.org/x/term"

	"taskflow/internal/config"
	"taskflow/internal/dto"
	"taskflow/internal/infrastructure/database"
	"taskflow/internal/repository"
	"taskflow/internal/service"
	"taskflow/migrations"
)

func main() { os.Exit(run()) }
func run() int {
	flags := flag.NewFlagSet("admin", flag.ContinueOnError)
	email := flags.String("email", "", "新管理员邮箱")
	name := flags.String("nickname", "", "管理员昵称")
	stdin := flags.Bool("password-stdin", false, "从非终端标准输入读取一行密码，供安全管道使用")
	if err := flags.Parse(os.Args[1:]); err != nil {
		return 1
	}
	if *email == "" || *name == "" || flags.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "用法：admin --email <邮箱> --nickname <昵称> [--password-stdin]")
		return 1
	}
	var secret []byte
	var err error
	if *stdin {
		if term.IsTerminal(int(os.Stdin.Fd())) {
			fmt.Fprintln(os.Stderr, "--password-stdin 只接受管道或重定向；交互终端请省略该选项")
			return 1
		}
		secret, err = bufio.NewReader(io.LimitReader(os.Stdin, 76)).ReadBytes('\n')
		if err == io.EOF {
			err = nil
		}
		secret = []byte(strings.TrimSuffix(strings.TrimSuffix(string(secret), "\n"), "\r"))
	} else {
		if !term.IsTerminal(int(os.Stdin.Fd())) {
			fmt.Fprintln(os.Stderr, "交互模式需要终端；自动化使用 --password-stdin")
			return 1
		}
		fmt.Fprint(os.Stderr, "管理员密码（不回显）：")
		secret, err = term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr)
		if err == nil {
			fmt.Fprint(os.Stderr, "再次输入密码：")
			again, e := term.ReadPassword(int(os.Stdin.Fd()))
			fmt.Fprintln(os.Stderr)
			if e != nil || string(secret) != string(again) {
				clear(secret)
				clear(again)
				fmt.Fprintln(os.Stderr, "两次密码不一致或读取失败")
				return 1
			}
			clear(again)
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "读取密码失败")
		return 1
	}
	defer clear(secret)
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "配置无效：", err)
		return 1
	}
	db, err := database.Open(cfg.Database)
	if err != nil {
		fmt.Fprintln(os.Stderr, "无法连接数据库，请检查配置")
		return 1
	}
	defer db.SQL.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	var version int
	if err = db.SQL.QueryRowContext(ctx, "SELECT COALESCE(MAX(version),0) FROM schema_migrations").Scan(&version); err != nil || version != migrations.LatestVersion {
		fmt.Fprintln(os.Stderr, "请先成功执行数据库迁移")
		return 1
	}
	user, err := service.BootstrapAdmin(ctx, &repository.AdminRepository{DB: db.ORM}, dto.RegisterRequest{Email: *email, Nickname: *name, Password: string(secret)})
	if err != nil {
		fmt.Fprintln(os.Stderr, service.AsAppError(err).Message)
		return 1
	}
	fmt.Fprintf(os.Stdout, "管理员已创建：%s（ID %s）。请在登录页使用刚设置的密码登录。\n", user.Email, user.ID)
	return 0
}
