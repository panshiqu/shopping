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
	{{range .}} <tr><td colspan="2">{{.Price}}</td></tr>{{.Content}} {{end}}
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

		if err := rows.Scan(sku); err != nil {
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

func main() {
	spider.Start()
	http.HandleFunc("/", procRequest)
	http.HandleFunc("/favicon.ico", func(http.ResponseWriter, *http.Request) {})
	log.Fatal(http.ListenAndServe(":8080", nil))
}
