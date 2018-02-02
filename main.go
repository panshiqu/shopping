package main

import (
	"fmt"
	"log"
	"net/http"
	"text/template"

	_ "github.com/go-sql-driver/mysql"
	"github.com/panshiqu/shopping/cache"
	"github.com/panshiqu/shopping/db"
	"github.com/panshiqu/shopping/spider"
)

var index = template.Must(template.New("index").Parse(`<html><body><table>
	{{range .}} <tr><td colspan="2"><hr />价格：{{.Price}} 刷新时间：{{.Timestamp}} 最低价：{{.MinPrice}} 最高价：{{.MaxPrice}}</td></tr>{{.Content}} {{end}}
	</table></body></html>`))

func procRequest(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Ins.Query("SELECT sku FROM sku ORDER BY priority")
	if err != nil {
		fmt.Fprint(w, err)
		return
	}

	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var sku int64

		if err := rows.Scan(&sku); err != nil {
			fmt.Fprint(w, err)
			return
		}

		ids = append(ids, sku)
	}

	if err := rows.Err(); err != nil {
		fmt.Fprint(w, err)
		return
	}

	if err := index.Execute(w, cache.Select(ids)); err != nil {
		fmt.Fprint(w, err)
		return
	}
}

func procAdminRequest(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, `<html><body><form><input type="number" name="sku">*商品编号（https://item.jd.com/商品编号.html）<br /><input type="number" name="priority">*优先级（作为刷新周期，越小越频繁，以秒为单位）<br /><input type="password" name="password">*请输入密码，不能谁都能添加吧<br /><br /><input type="submit" value="Submit"></form></body></html>`)
}

func main() {
	go spider.Start()
	http.HandleFunc("/", procRequest)
	http.HandleFunc("/admin", procAdminRequest)
	http.HandleFunc("/favicon.ico", func(http.ResponseWriter, *http.Request) {})
	log.Fatal(http.ListenAndServe(":8080", nil))
}
