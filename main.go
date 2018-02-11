package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"text/template"

	_ "github.com/go-sql-driver/mysql"
	"github.com/panshiqu/shopping/cache"
	"github.com/panshiqu/shopping/db"
	"github.com/panshiqu/shopping/define"
	"github.com/panshiqu/shopping/spider"
)

var index = template.Must(template.New("index").Parse(`<html><body><ul><li>只是来玩游戏的请点击 <a href='http://13.250.117.241:8081' target='_blank'>这里</a></li><li>请搜索 <font color="red">Min</font> 快速浏览当前价格为最低价的商品</li><li>请搜索 <font color="red">京东秒杀</font> 快速浏览正在参与或即将参与秒杀的商品</li></ul><table>
	{{range .}} <tr><td colspan="2"><hr />{{if .IsMinPrice}}<font color="red" size="4">Min</font> {{end}}编号：{{.SkuID}} 价格：<font color="red" size="4">{{.Price}}</font> 刷新时间：{{.Timestamp}} 最低价：{{.MinPrice}} 最高价：{{.MaxPrice}} 已持续：{{.Duration}} 有效采样{{.Sampling}}次</td></tr>{{.Content}} {{end}}
	</table></body></html>`))

func procRequest(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Ins.Query("SELECT sku FROM sku ORDER BY priority")
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

	if err := index.Execute(w, cache.Select(ids)); err != nil {
		log.Println("procRequest Execute", err)
		fmt.Fprint(w, err)
		return
	}
}

func procAdminRequest(w http.ResponseWriter, r *http.Request) {
	if r.FormValue("password") != "161015" {
		fmt.Fprint(w, `<html><body><form><input type="number" name="sku">*商品编号（https://item.jd.com/商品编号.html）<br /><input type="number" name="priority" value="28800" min="28800">*优先级（作为刷新周期，越小越频繁，以秒为单位，最低28800秒）<br /><input type="password" name="password">*请输入密码，不能谁都能添加吧<br /><br /><input type="submit" value="Submit"></form></body></html>`)
		return
	}

	log.Println("procAdminRequest", r.FormValue("sku"), r.FormValue("priority"))

	sku, err := strconv.Atoi(r.FormValue("sku"))
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

	priority, err := strconv.Atoi(r.FormValue("priority"))
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

func main() {
	log.SetFlags(log.Flags() | log.Lshortfile)
	log.Println("Start...")

	go spider.Start()
	http.HandleFunc("/", procRequest)
	http.HandleFunc("/admin", procAdminRequest)
	http.HandleFunc("/favicon.ico", func(http.ResponseWriter, *http.Request) {})
	log.Fatal(http.ListenAndServe(":8080", nil))
}
