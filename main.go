package main

import (
	"log"
	"net/http"
	"text/template"

	_ "github.com/go-sql-driver/mysql"
	"github.com/panshiqu/shopping/spider"
)

var indexT *template.Template

const index = `<html><body><table>
{{range .}} <tr><td colspan="2">{{.Price}}</td></tr>{{.Content}} {{end}}
</table></body></html>`

type indexP struct {
	Price   string
	Content string
}

func procRequest(w http.ResponseWriter, r *http.Request) {

}

func main() {
	spider.StartSpider()
	http.HandleFunc("/", procRequest)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func init() {
	var err error

	if indexT, err = template.New("index").Parse(index); err != nil {
		log.Fatal(err)
	}
}
