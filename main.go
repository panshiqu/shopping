package main

import (
	"log"
	"net/http"
	"text/template"

	_ "github.com/go-sql-driver/mysql"
	"github.com/panshiqu/shopping/spider"
)

var index = template.Must(template.New("index").Parse(`<html><body><table>
	{{range .}} <tr><td colspan="2">{{.Price}}</td></tr>{{.Content}} {{end}}
	</table></body></html>`))

func procRequest(w http.ResponseWriter, r *http.Request) {

}

func main() {
	spider.Start()
	http.HandleFunc("/", procRequest)
	http.HandleFunc("/favicon.ico", func(http.ResponseWriter, *http.Request) {})
	log.Fatal(http.ListenAndServe(":8080", nil))
}
