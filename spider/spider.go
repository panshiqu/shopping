package spider

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/panshiqu/framework/utils"
	"github.com/panshiqu/shopping/cache"
	"github.com/panshiqu/shopping/db"
	"github.com/panshiqu/shopping/define"
	"github.com/robertkrimen/otto"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// Spider 蜘蛛
type Spider struct {
}

var schedule *utils.Schedule

// Add 增加
func Add(sku, priority int64) error {
	if err := jdSpider(sku); err != nil {
		return err
	}
	schedule.Add(int(sku), time.Duration(priority)*time.Second, nil, true)
	return nil
}

// Start 开始
func Start() {
	schedule = utils.NewSchedule(&Spider{})
	go schedule.Start()

	rows, err := db.Ins.Query("SELECT sku,priority FROM sku")
	if err != nil {
		log.Fatal(err)
	}

	defer rows.Close()

	for rows.Next() {
		var sku, priority int64

		if err := rows.Scan(&sku, &priority); err != nil {
			log.Fatal(err)
		}

		if err := Add(sku, priority); err != nil {
			log.Fatal(err)
		}
	}

	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
}

// OnTimer 定时器到期
func (s *Spider) OnTimer(id int, parameter interface{}) {
	if err := jdSpider(int64(id)); err != nil {
		log.Println("OnTimer", id, err)
	}
}

func fetchURL(in string) ([]byte, error) {
	resp, err := http.Get(in)
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

func getIntSlice(vm *otto.Otto, in string) []int64 {
	if v, err := vm.Run(in); err == nil {
		if v, err := v.Export(); err == nil {
			if v, ok := v.([]int64); ok {
				return v
			}
		}
	}
	return nil
}

func js2Go(in []byte) (*define.JDPageConfig, error) {
	vm := otto.New()
	if _, err := vm.Run(in); err != nil {
		return nil, err
	}
	return &define.JDPageConfig{
		SkuID:       getInt(vm, "pageConfig.product.skuid"),
		Name:        getString(vm, "pageConfig.product.name"),
		KoBeginTime: getInt(vm, "pageConfig.product.koBeginTime"),
		KoEndTime:   getInt(vm, "pageConfig.product.koEndTime"),
		Src:         fmt.Sprintf("http://img14.360buyimg.com/n1/%s", getString(vm, "pageConfig.product.src")),
		Cat:         getIntSlice(vm, "pageConfig.product.cat"),
	}, nil
}

func getJDPrice(in *define.JDPageConfig) (*define.JDPrice, []byte, error) {
	body, err := fetchURL(fmt.Sprintf("https://p.3.cn/prices/mgets?area=7_412_47301_0&pduid=%d&skuIds=J_%d", time.Now().UnixNano(), in.SkuID))
	if err != nil {
		return nil, nil, err
	}
	var jdps []*define.JDPrice
	if err := json.Unmarshal(body, &jdps); err != nil {
		return nil, nil, err
	}
	return jdps[0], body, nil
}

func getJDInfo(in *define.JDPageConfig) (*define.JDInfo, []byte, error) {
	body, err := fetchURL(fmt.Sprintf("https://cd.jd.com/promotion/v2?skuId=%d&area=7_412_47301_0&cat=%s", in.SkuID, in.JoinCat()))
	if err != nil {
		return nil, nil, err
	}
	body, err = gbk2utf8(body)
	if err != nil {
		return nil, nil, err
	}
	jdi := &define.JDInfo{}
	if err := json.Unmarshal(body, jdi); err != nil {
		return nil, nil, err
	}
	if jdi.Quan[0] == '[' {
		if err := json.Unmarshal(jdi.Quan, &jdi.Quans); err != nil {
			return nil, nil, err
		}
	} else {
		jdq := &define.JDQuan{}
		if err := json.Unmarshal(jdi.Quan, jdq); err != nil {
			return nil, nil, err
		}
		jdi.Quans = append(jdi.Quans, jdq)
	}
	return jdi, body, nil
}

func serializeHTML(jdi *define.JDInfo, jdpc *define.JDPageConfig) string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "<tr><td><a href='https://item.jd.com/%d.html' target='_blank'><img src='%s' /></a></td><td>", jdpc.SkuID, jdpc.Src)
	if jdpc.KoBeginTime != 0 {
		fmt.Fprintf(&buf, "【京东秒杀%s开始】", time.Unix(jdpc.KoBeginTime/1000, 0).Format("01-02 15:04"))
	}
	if jdpc.KoEndTime != 0 {
		fmt.Fprintf(&buf, "【京东秒杀%s结束】", time.Unix(jdpc.KoEndTime/1000, 0).Format("01-02 15:04"))
	}
	fmt.Fprintf(&buf, "<a href='https://item.jd.com/%d.html' target='_blank'>%s</a><br />", jdpc.SkuID, jdpc.Name)
	for _, v := range jdi.SkuCoupon {
		switch v.CouponStyle {
		case 0:
			fmt.Fprintf(&buf, "【满%d减%d】%s %s %s<br />", v.Quota, v.Discount, v.TimeDesc, v.Name, v.OverlapDesc)
		case 3:
			fmt.Fprintf(&buf, "【%s-%s】%s %s %s<br />", v.AllDesc, v.HighDesc, v.TimeDesc, v.Name, v.OverlapDesc)
		default:
			fmt.Fprintf(&buf, "Unknown Coupon Style: %d<br />", v.CouponStyle)
		}
	}
	for _, v := range jdi.Ads {
		if v.Ad != "" {
			fmt.Fprintf(&buf, "%s<br />", v.Ad)
		}
	}
	for _, v := range jdi.Quans {
		fmt.Fprintf(&buf, "<a href='%s' target='_blank'>%s</a><br />", v.ActURL, v.Title)
	}
	for _, v := range jdi.Prom.PickOneTag {
		fmt.Fprintf(&buf, "【%s】<a href='%s' target='_blank'>%s</a><br />", v.Name, v.AdURL, v.Content)
	}
	for _, v := range jdi.Prom.Tags {
		if v.AdURL != "" {
			fmt.Fprintf(&buf, "【%s】<a href='%s' target='_blank'>%s</a><br />", v.Name, v.AdURL, v.Content)
		} else {
			fmt.Fprintf(&buf, "【%s】%s<br />", v.Name, v.Content)
		}
	}
	if bytes.HasSuffix(buf.Bytes(), []byte("<br />")) {
		buf.Truncate(buf.Len() - 6)
	}
	fmt.Fprintf(&buf, "</td></tr>")
	return buf.String()
}

func jdSpider(in int64) error {
	body, err := fetchURL(fmt.Sprintf("https://item.jd.com/%d.html", in))
	if err != nil {
		return err
	}
	pc, err := getPageConfig(body)
	if err != nil {
		return err
	}
	pc, err = gbk2utf8(pc)
	if err != nil {
		return err
	}
	jdpc, err := js2Go(pc)
	if err != nil {
		return err
	}
	jdp, pdt, err := getJDPrice(jdpc)
	if err != nil {
		return err
	}
	jdi, idt, err := getJDInfo(jdpc)
	if err != nil {
		return err
	}
	price, err := strconv.ParseFloat(jdp.Price, 64)
	if err != nil {
		return err
	}
	content := serializeHTML(jdi, jdpc)
	if err := cache.Update(in, price, content); err != nil {
		return err
	}
	if _, err := db.Ins.Exec("INSERT INTO jd (sku,price,content,jd_price,jd_promotion,jd_page_config) VALUES (?,?,?,?,?,?)", in, price, content, pdt, idt, pc); err != nil {
		return err
	}
	return nil
}
