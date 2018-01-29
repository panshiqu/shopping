package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/robertkrimen/otto"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

type jdPageConfig struct {
	SkuID int64
}

func getURL(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}

func getPageConfig(in []byte) ([]byte, error) {
	begin := bytes.Index(in, []byte("var pageConfig"))
	if begin == -1 {
		return nil, errors.New("Index begin")
	}
	in = in[begin:]
	end := bytes.Index(in, []byte(";"))
	if end == -1 {
		return nil, errors.New("Index end")
	}
	return in[:end+1], nil
}

func gbk2utf8(in []byte) ([]byte, error) {
	return ioutil.ReadAll(transform.NewReader(bytes.NewReader(in), simplifiedchinese.GBK.NewDecoder()))
}

func getInt(vm *otto.Otto, in string) int64 {
	if v, err := vm.Run(in); err == nil {
		if v, err := v.ToInteger(); err == nil {
			return v
		}
	}
	return 0
}

func getString(vm *otto.Otto, in string) string {
	if v, err := vm.Run(in); err == nil {
		if v, err := v.ToString(); err == nil {
			return v
		}
	}
	return ""
}

func js2Go(in []byte) (*jdPageConfig, error) {
	vm := otto.New()
	if _, err := vm.Run(in); err != nil {
		return nil, err
	}
	return &jdPageConfig{
		SkuID: getInt(vm, "pageConfig.product.skuid"),
	}, nil
}

func main() {
	body, err := getURL("https://item.jd.com/1268059.html")
	if err != nil {
		log.Fatal(err)
	}

	pc, err := getPageConfig(body)
	if err != nil {
		log.Fatal(err)
	}

	pc, err = gbk2utf8(pc)
	if err != nil {
		log.Fatal(err)
	}

	jdpc, err := js2Go(pc)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(jdpc)
}
