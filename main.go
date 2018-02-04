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

var index = template.Must(template.New("index").Parse(`<html><body><table>
	{{range .}} <tr><td colspan="2"><hr />价格：{{.Price}} 刷新时间：{{.Timestamp}} 最低价：{{.MinPrice}} 最高价：{{.MaxPrice}}</td></tr>{{.Content}} {{end}}
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
	go spider.Start()
	http.HandleFunc("/", procRequest)
	http.HandleFunc("/admin", procAdminRequest)
	http.HandleFunc("/favicon.ico", func(http.ResponseWriter, *http.Request) {})
	log.Fatal(http.ListenAndServe(":8080", nil))
}
