package main

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"text/template"

	_ "github.com/go-sql-driver/mysql"
	"github.com/panshiqu/shopping/cache"
	"github.com/panshiqu/shopping/db"
	"github.com/panshiqu/shopping/define"
	"github.com/panshiqu/shopping/spider"
)

var aliasMutex sync.Mutex

var captcha map[string]int32

var index = template.Must(template.New("index").Parse(`<html><body><ul><li>只是来玩游戏的请点击 <a href='http://13.250.117.241:8081' target='_blank'>这里</a></li><li>请搜索 <font color="red">Min</font> 快速浏览当前价格为最低价的商品</li><li>请搜索 <font color="red">京东秒杀</font> 快速浏览正在参与或即将参与秒杀的商品</li></ul><table>
	{{range .Args}} <tr><td colspan="2"><hr />{{if .IsMinPrice}}<font color="red" size="4">Min</font> {{end}}编号：{{.SkuID}} 价格：<font color="red" size="4">{{.Price}}</font> 刷新时间：{{.Timestamp}} 最低价：{{.MinPrice}} 最高价：{{.MaxPrice}} 已持续：{{.Duration}} 有效采样{{.Sampling}}次 {{if eq $.Alias ""}}<a href='{{printf "/subscribe?sku=%d&keywords=%s" .SkuID .Name}}' target='_blank'>订阅</a>{{else}}<a href='{{printf "/unsubscribe?sku=%d&alias=%s" .SkuID $.Alias}}' target='_blank'>退订</a>{{end}}</td></tr>{{.Content}} {{end}}
	</table></body></html>`))

func procRequest(w http.ResponseWriter, r *http.Request) {
	var err error
	var rows *sql.Rows
	alias := r.FormValue("alias")

	if alias == "" {
		rows, err = db.Ins.Query("SELECT sku FROM sku ORDER BY priority")
	} else {
		var id string

		if err := db.Ins.QueryRow("SELECT id FROM user WHERE alias = ?", alias).Scan(&id); err != nil {
			log.Println("procRequest QueryRow", err)
			fmt.Fprint(w, err)
			return
		}

		rows, err = db.Ins.Query("SELECT sku FROM subscribe WHERE id = ? ORDER BY keywords", id)
	}

	if err != nil {
		log.Println("procRequest Query", err)
		fmt.Fprint(w, err)
		return
	}

	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var sku int64

		if err := rows.Scan(&sku); err != nil {
			log.Println("procRequest Scan", err)
			fmt.Fprint(w, err)
			return
		}

		ids = append(ids, sku)
	}

	if err := rows.Err(); err != nil {
		log.Println("procRequest Err", err)
		fmt.Fprint(w, err)
		return
	}

	data := &define.IndexData{
		Args:  cache.Select(ids),
		Alias: alias,
	}

	if err := index.Execute(w, data); err != nil {
		log.Println("procRequest Execute", err)
		fmt.Fprint(w, err)
		return
	}
}

func procBindRequest(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")

	if id == "" || captcha[id] == 0 || fmt.Sprintf("%d", captcha[id]) != r.FormValue("captcha") {
		fmt.Fprint(w, `
			<html>
			<body>
			<form>
			<input type="text" name="id" size="32">*休闲益智游戏公众号发送 id 获得<br />
			<input type="text" name="alias">*请设置别名，暂仅支持纯字母组合，不区分大小写<br />
			<input type="text" name="password">*请设置密码，暂仅支持6位纯数字组合<br />
			<input type="number" name="captcha"> <a href="/captcha" target="_blank">获取</a><br /><br />
			<input type="submit" value="绑定">
			</form>
			<img src="http://13.250.117.241:8081/qrcode.jpg" alt="休闲益智游戏">
			</body>
			</html>
			`)
		return
	}

	alias := strings.ToLower(r.FormValue("alias"))
	password := r.FormValue("password")

	log.Println("procBindRequest", id, alias, password)

	if l := len(alias); l == 0 || l > 128 || len(password) != 6 {
		log.Println("procBindRequest", define.ErrIllegalLen)
		fmt.Fprint(w, define.ErrIllegalLen)
		return
	}

	for _, v := range alias {
		if v < 'a' || v > 'z' {
			log.Println("procBindRequest", define.ErrIllegalAlias)
			fmt.Fprint(w, define.ErrIllegalAlias)
			return
		}
	}

	for _, v := range password {
		if v < '0' || v > '9' {
			log.Println("procBindRequest", define.ErrIllegalPassword)
			fmt.Fprint(w, define.ErrIllegalPassword)
			return
		}
	}

	aliasMutex.Lock()
	defer aliasMutex.Unlock()

	if err := db.Ins.QueryRow("SELECT alias FROM user WHERE alias = ? AND id <> ?", alias, id).Scan(&alias); err != sql.ErrNoRows {
		if err == nil {
			err = define.ErrAlreadyExist
		}
		log.Println("procBindRequest QueryRow", err)
		fmt.Fprint(w, err)
		return
	}

	if _, err := db.Ins.Exec("INSERT INTO user (id,alias,password) VALUES (?,?,?) ON DUPLICATE KEY UPDATE alias = ?,password = ?", id, alias, password, alias, password); err != nil {
		log.Println("procBindRequest Exec", err)
		fmt.Fprint(w, err)
		return
	}

	fmt.Fprintf(w, "<html><body>绑定成功，请自主<a href='/subscribe' target='_blank'>订阅商品</a> or <a href='/' target='_blank'>首页快速订阅</a>，然后访问您的<a href='/?alias=%s' target='_blank'>专属链接</a></body></html>", alias)
}

func procAdminRequest(w http.ResponseWriter, r *http.Request) {
	if r.FormValue("password") != "161015" {
		fmt.Fprint(w, `
			<html>
			<body>
			<form>
			<input type="number" name="sku">*商品编号（https://item.jd.com/商品编号.html）<br />
			<input type="number" name="priority" value="28800" min="28800">*优先级（作为刷新周期，越小越频繁，以秒为单位，最低28800秒）<br />
			<input type="password" name="password">*请输入密码，不能谁都能添加吧<br /><br />
			<input type="submit" value="Submit">
			</form>
			</body>
			</html>
			`)
		return
	}

	skuStr := r.FormValue("sku")
	priorityStr := r.FormValue("priority")

	log.Println("procAdminRequest", skuStr, priorityStr)

	sku, err := strconv.Atoi(skuStr)
	if err != nil {
		log.Println("procAdminRequest sku", err)
		fmt.Fprint(w, err)
		return
	}

	if cache.Exist(int64(sku)) {
		log.Println("procAdminRequest", define.ErrAlreadyExist)
		fmt.Fprint(w, define.ErrAlreadyExist)
		return
	}

	priority, err := strconv.Atoi(priorityStr)
	if err != nil {
		log.Println("procAdminRequest priority", err)
		fmt.Fprint(w, err)
		return
	}

	if priority < 8*60*60 {
		log.Println("procAdminRequest", define.ErrToSmallPriority)
		fmt.Fprint(w, define.ErrToSmallPriority)
		return
	}

	if _, err := db.Ins.Exec("INSERT INTO sku (sku,priority) VALUES (?,?)", sku, priority); err != nil {
		log.Println("procAdminRequest Exec", err)
		fmt.Fprint(w, err)
		return
	}

	if err := spider.Add(int64(sku), int64(priority)); err != nil {
		log.Println("procAdminRequest Add", err)
		fmt.Fprint(w, err)
		return
	}

	http.Redirect(w, r, "", http.StatusFound)
}

func procCaptchaRequest(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")

	if id == "" {
		fmt.Fprint(w, `
			<html>
			<body>
			<form>
			<input type="text" name="id" size="32">*休闲益智游戏公众号发送 id 获得<br /><br />
			<input type="submit" value="获取">
			</form>
			</body>
			</html>
			`)
		return
	}

	captcha[id] = rand.Int31n(900000) + 100000

	resp, err := http.Get(fmt.Sprintf(`http://localhost/push?id=%s&message=验证码：%d`, id, captcha[id]))
	if err != nil {
		log.Println("procCaptchaRequest Get", err)
		fmt.Fprint(w, err)
		return
	}

	resp.Body.Close()

	fmt.Fprint(w, "已发送，请打开休闲益智游戏公众号查看\n若未收到，可能因为您好久未与公众号交互，请在公众号内发送任意内容之后再次获取验证码")
}

func procSubscribeRequest(w http.ResponseWriter, r *http.Request) {
	skuStr := r.FormValue("sku")
	alias := r.FormValue("alias")
	password := r.FormValue("password")
	keywords := r.FormValue("keywords")

	if skuStr == "" || alias == "" {
		fmt.Fprintf(w, `
			<html>
			<body>
			<form>
			<input type="number" name="sku" value="%s">*请订阅添加过的商品编号，<a href='/admin' target='_blank'>添加商品</a><br />
			<input type="text" name="alias">*绑定时输入的别名<br />
			<input type="text" name="password">*绑定时输入的密码<br />
			<input type="text" name="keywords" value="%s" size="64">*关键字用于排序<br /><br />
			<input type="submit" value="订阅">
			</form>
			</body>
			</html>
			`, skuStr, keywords)
		return
	}

	log.Println("procSubscribeRequest", skuStr, alias, password, keywords)

	var id string

	if err := db.Ins.QueryRow("SELECT id FROM user WHERE alias = ? AND password = ?", alias, password).Scan(&id); err != nil {
		log.Println("procSubscribeRequest QueryRow", err)
		fmt.Fprint(w, err)
		return
	}

	sku, err := strconv.Atoi(skuStr)
	if err != nil {
		log.Println("procSubscribeRequest", err)
		fmt.Fprint(w, err)
		return
	}

	if !cache.Exist(int64(sku)) {
		log.Println("procSubscribeRequest", define.ErrNotExist)
		fmt.Fprint(w, define.ErrNotExist)
		return
	}

	if _, err := db.Ins.Exec("INSERT INTO subscribe (id,sku,keywords) VALUES (?,?,?) ON DUPLICATE KEY UPDATE keywords = ?", id, sku, keywords, keywords); err != nil {
		log.Println("procSubscribeRequest Exec", err)
		fmt.Fprint(w, err)
		return
	}

	fmt.Fprintf(w, "<html><body>订阅成功，<a href='/' target='_blank'>首页快速订阅</a> or <a href='/subscribe' target='_blank'>继续订阅</a> or <a href='/?alias=%s' target='_blank'>专属链接</a></body></html>", alias)
}

func procUnSubscribeRequest(w http.ResponseWriter, r *http.Request) {
	sku := r.FormValue("sku")
	alias := r.FormValue("alias")
	password := r.FormValue("password")

	if password == "" {
		fmt.Fprintf(w, `
			<html>
			<body>
			<form>
			<input type="number" name="sku" value="%s">*请退订订阅过的商品编号，<a href='/subscribe' target='_blank'>订阅商品</a><br />
			<input type="text" name="alias" value="%s">*绑定时输入的别名<br />
			<input type="text" name="password">*绑定时输入的密码<br /><br />
			<input type="submit" value="退订">
			</form>
			</body>
			</html>
			`, sku, alias)
		return
	}

	log.Println("procUnSubscribeRequest", sku, alias, password)

	var id string

	if err := db.Ins.QueryRow("SELECT id FROM user WHERE alias = ? AND password = ?", alias, password).Scan(&id); err != nil {
		log.Println("procUnSubscribeRequest QueryRow", err)
		fmt.Fprint(w, err)
		return
	}

	if _, err := db.Ins.Exec("DELETE FROM subscribe WHERE id = ? AND sku = ?", id, sku); err != nil {
		log.Println("procUnSubscribeRequest Exec", err)
		fmt.Fprint(w, err)
		return
	}

	fmt.Fprintf(w, "<html><body>退订成功，<a href='/unsubscribe' target='_blank'>继续退订</a> or <a href='/?alias=%s' target='_blank'>专属链接快速退订</a></body></html>", alias)
}

func main() {
	captcha = make(map[string]int32)

	log.SetFlags(log.Flags() | log.Lshortfile)
	log.Println("Start...")

	go spider.Start()
	http.HandleFunc("/", procRequest)
	http.HandleFunc("/bind", procBindRequest)
	http.HandleFunc("/admin", procAdminRequest)
	http.HandleFunc("/captcha", procCaptchaRequest)
	http.HandleFunc("/subscribe", procSubscribeRequest)
	http.HandleFunc("/unsubscribe", procUnSubscribeRequest)
	http.HandleFunc("/favicon.ico", func(http.ResponseWriter, *http.Request) {})
	log.Fatal(http.ListenAndServe(":8080", nil))
}
