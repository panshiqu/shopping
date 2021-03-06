package spider

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

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
	end := bytes.Index(in, []byte("};"))
	if end == -1 {
		return nil, errors.New("Index end")
	}
	return in[:end+2], nil
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

func getJDTax(in int64) (float64, error) {
	body, err := fetchURL(fmt.Sprintf("https://c.3.cn/globalBuy?skuId=%d", in))
	if err != nil {
		return 0, err
	}
	body, err = gbk2utf8(body)
	if err != nil {
		return 0, err
	}
	jdgb := &define.JDGlobalBuy{}
	if err := json.Unmarshal(body, jdgb); err != nil {
		return 0, err
	}
	if !jdgb.Success {
		return 0, nil
	}
	pos := strings.Index(jdgb.TaxTxt.Content, "￥")
	if pos == -1 {
		return 0, errors.New("Index pos")
	}
	return strconv.ParseFloat(jdgb.TaxTxt.Content[pos+3:], 64)
}

func serializeHTML(jdi *define.JDInfo, jdpc *define.JDPageConfig, price float64) (string, float64) {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "<tr><td><a href='https://item.jd.com/%d.html' target='_blank'><img src='%s' /></a></td><td>", jdpc.SkuID, jdpc.Src)
	if jdpc.KoBeginTime != 0 {
		fmt.Fprintf(&buf, "<font color='red'>【京东秒杀%s开始】</font>", time.Unix(jdpc.KoBeginTime/1000, 0).Format("01-02 15:04"))
	}
	if jdpc.KoEndTime != 0 {
		fmt.Fprintf(&buf, "<font color='red'>【京东秒杀%s结束】</font>", time.Unix(jdpc.KoEndTime/1000, 0).Format("01-02 15:04"))
	}
	fmt.Fprintf(&buf, "<a href='https://item.jd.com/%d.html' target='_blank'>%s</a><br /><!--begin-->", jdpc.SkuID, jdpc.Name)
	discount := float64(0.95) // 全品类满200减10
	for _, v := range jdi.SkuCoupon {
		var dis float64
		switch v.CouponStyle {
		case 0:
			fmt.Fprintf(&buf, "【满%d减%d】%s %s %s", v.Quota, v.Discount, v.TimeDesc, v.Name, v.OverlapDesc)
			quota := float64(v.Quota)
			if price > quota {
				quota = price
			}
			dis = (quota - float64(v.Discount)) / quota
		case 3:
			fmt.Fprintf(&buf, "##【%s-%s】%s %s %s", v.AllDesc, v.HighDesc, v.TimeDesc, v.Name, v.OverlapDesc)
		default:
			fmt.Fprintf(&buf, "Unknown Coupon Style: %d", v.CouponStyle)
		}
		if dis != 0 {
			fmt.Fprintf(&buf, "<!--dis=%f-->", dis)
			if dis < discount {
				discount = dis
			}
		}
		fmt.Fprintf(&buf, "<br />")
	}
	for _, v := range jdi.Ads {
		if v.Ad != "" {
			fmt.Fprintf(&buf, "%s<br />", v.Ad)
		}
	}
	for _, v := range jdi.Quans {
		fmt.Fprintf(&buf, "【满额返券】<a href='%s' target='_blank'>%s</a><br />", v.ActURL, v.Title)
	}
	tags := append(jdi.Prom.PickOneTag, jdi.Prom.Tags...)
	sort.Sort(define.TagsSlice(tags))
	np := serializeTag(&buf, tags, price)
	if bytes.HasSuffix(buf.Bytes(), []byte("<br />")) {
		buf.Truncate(buf.Len() - 6)
	}
	fmt.Fprintf(&buf, "<!--end--></td></tr>")
	if np != -1 { // 商品未下柜
		np *= discount
	}
	return buf.String(), np
}

func serializeTag(buf *bytes.Buffer, tags []*define.JDTag, price float64) float64 {
	discount := float64(1)
	for _, v := range tags {
		if len(v.Gifts) != 0 {
			for _, vv := range v.Gifts {
				fmt.Fprintf(buf, "【%s】<a href='https://item.jd.com/%s.html' target='_blank'>%s</a>X%d%s", v.Name, vv.Sid, vv.Nm, vv.Num, v.Content)
			}
		} else if v.AdURL != "" {
			fmt.Fprintf(buf, "【%s】<a href='%s' target='_blank'>%s</a>", v.Name, v.AdURL, v.Content)
		} else {
			fmt.Fprintf(buf, "【%s】%s", v.Name, v.Content)
		}
		var dis float64
		switch v.Code {
		case "15": // 满减
			var a, b float64
			if strings.Contains(v.Content, "选") {
				fmt.Sscanf(v.Content, "%f元选%f件", &a, &b)
				dis = a / b / price
			} else {
				s := v.Content
				if n := strings.LastIndex(s, "最多"); n != -1 {
					s = s[:n]
				}
				if n := strings.LastIndex(s, "满"); n != -1 {
					s = s[n:]
				}
				fmt.Sscanf(formatStr(s), "%f元%f元", &a, &b)
				dis = (a - b) / a
			}
			fmt.Fprintf(buf, "<!--a=%f,b=%f-->", a, b)
		case "19": // 多买优惠
			if n := strings.LastIndex(v.Content, "打"); n != -1 {
				fmt.Sscanf(v.Content[n:], "打%f折", &dis)
			}
			dis = dis / 10
		}
		if dis != 0 {
			fmt.Fprintf(buf, "<!--dis=%f-->", dis)
			if dis < discount {
				discount = dis
			}
		}
		fmt.Fprintf(buf, "<br />")
	}
	if price != -1 { // 商品未下柜
		price *= discount
	}
	return price
}

func formatStr(in string) (out string) {
	for _, v := range in {
		if unicode.IsNumber(v) || v == '.' || v == '元' {
			out += string(v)
		}
	}
	return
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
	tax, err := getJDTax(in)
	if err != nil {
		return err
	}
	price, err := strconv.ParseFloat(jdp.Price, 64)
	if err != nil {
		return err
	}
	content, price := serializeHTML(jdi, jdpc, price)
	price = math.Trunc((price+tax)*100+0.5) / 100
	push, err := cache.Update(in, price, content, jdpc.Name)
	if err == define.ErrDataSame {
		return nil
	}
	if err != nil {
		return err
	}
	if _, err := db.Ins.Exec("INSERT INTO jd (sku,price,content,jd_price,jd_promotion,jd_page_config) VALUES (?,?,?,?,?,?)", in, price, content, pdt, idt, pc); err != nil {
		return err
	}
	if !push {
		return nil
	}
	msg := url.QueryEscape(fmt.Sprintf("%s降价至%.2f https://item.jd.com/%d.html", jdpc.Name, price, in))
	rows, err := db.Ins.Query("SELECT id FROM subscribe WHERE sku = ?", in)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			log.Println("jdSpider Scan", err)
			continue
		}
		resp, err := http.Get(fmt.Sprintf(`http://localhost/push?id=%s&message=%s`, id, msg))
		if err != nil {
			log.Println("jdSpider Get", err)
			continue
		}
		resp.Body.Close()
	}
	return rows.Err()
}
