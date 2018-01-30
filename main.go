package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"text/template"
	"time"

	"github.com/robertkrimen/otto"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

const index = `<html><body><table>
{{range .}} <tr><td colspan="2">{{.Price}}</td></tr>{{.Content}} {{end}}
</table></body></html>`

var indexT = template.Must(template.New("index").Parse(index))

type indexP struct {
	Price   string
	Content string
}

type jdPrice struct {
	Price       string `json:"p"`
	OriginPrice string `json:"op"`
}

type jdInfo struct {
	Quan       json.RawMessage `json:"quan"`
	SkuCoupon  []*jdSkuCoupon  `json:"skuCoupon"`
	AdsStatus  int64           `json:"adsStatus"`
	Ads        []*jdAds        `json:"ads"`
	QuanStatus int64           `json:"quanStatus"`
	PromStatus int64           `json:"promStatus"`
	Prom       *jdProm         `json:"prom"`

	Quans []*jdQuan
}

type jdQuan struct {
	Title  string `json:"title"`
	ActURL string `json:"actUrl"`
}

type jdSkuCoupon struct {
	CouponType   int64   `json:"couponType"`
	TrueDiscount float64 `json:"trueDiscount"`
	CouponKind   int64   `json:"couponKind"`
	DiscountDesc string  `json:"discountDesc"`
	BeginTime    string  `json:"beginTime"`
	UserClass    int64   `json:"userClass"`
	URL          string  `json:"url"`
	OverlapDesc  string  `json:"overlapDesc"`
	CouponStyle  int64   `json:"couponStyle"`
	Area         int64   `json:"area"`
	HourCoupon   int64   `json:"hourCoupon"`
	Overlap      int64   `json:"overlap"`
	EndTime      string  `json:"endTime"`
	Key          string  `json:"key"`
	AddDays      int64   `json:"addDays"`
	Quota        int64   `json:"quota"`
	ToURL        string  `json:"toUrl"`
	TimeDesc     string  `json:"timeDesc"`
	RoleID       int64   `json:"roleId"`
	Discount     int64   `json:"discount"`
	DiscountFlag int64   `json:"discountFlag"`
	LimitType    int64   `json:"limitType"`
	Name         string  `json:"name"`
	BatchID      int64   `json:"batchId"`

	AllDesc      string          `json:"allDesc"`
	DiscountJSON json.RawMessage `json:"discountJson"`
	SimDesc      string          `json:"simDesc"`
	HighCount    int64           `json:"highCount"`
	HighDesc     string          `json:"highDesc"`
}

type jdAds struct {
	ID string `json:"id"`
	Ad string `json:"ad"`
}

type jdProm struct {
	Hit        int64           `json:"hit"`
	PickOneTag []*jdTag        `json:"pickOneTag"`
	CarGift    int64           `json:"carGift"`
	Tags       []*jdTag        `json:"tags"`
	GiftPool   json.RawMessage `json:"giftPool"`
	Ending     int64           `json:"ending"`
}

type jdTag struct {
	D       string `json:"d"`
	St      string `json:"st"`
	Code    string `json:"code"`
	Content string `json:"content"`
	Tr      int64  `json:"tr"`
	AdURL   string `json:"adurl,omitempty"`
	Name    string `json:"name"`
	Pid     string `json:"pid"`
}

type jdPageConfig struct {
	SkuID       int64
	Name        string
	KoBeginTime int64
	KoEndTime   int64
	Src         string
	Cat         []int64
}

func (j *jdPageConfig) JoinCat() []byte {
	if len(j.Cat) == 0 {
		return nil
	}
	var buf bytes.Buffer
	for _, v := range j.Cat {
		fmt.Fprint(&buf, v, ",")
	}
	return buf.Bytes()[:buf.Len()-1]
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

func js2Go(in []byte) (*jdPageConfig, error) {
	vm := otto.New()
	if _, err := vm.Run(in); err != nil {
		return nil, err
	}
	return &jdPageConfig{
		SkuID:       getInt(vm, "pageConfig.product.skuid"),
		Name:        getString(vm, "pageConfig.product.name"),
		KoBeginTime: getInt(vm, "pageConfig.product.koBeginTime"),
		KoEndTime:   getInt(vm, "pageConfig.product.koEndTime"),
		Src:         fmt.Sprintf("http://img14.360buyimg.com/n1/%s", getString(vm, "pageConfig.product.src")),
		Cat:         getIntSlice(vm, "pageConfig.product.cat"),
	}, nil
}

func getJDPrice(in *jdPageConfig) (*jdPrice, error) {
	body, err := getURL(fmt.Sprintf("https://p.3.cn/prices/mgets?skuIds=J_%d", in.SkuID))
	if err != nil {
		return nil, err
	}
	var jdps []*jdPrice
	if err := json.Unmarshal(body, &jdps); err != nil {
		return nil, err
	}
	return jdps[0], nil
}

func getJDInfo(in *jdPageConfig) (*jdInfo, error) {
	body, err := getURL(fmt.Sprintf("https://cd.jd.com/promotion/v2?skuId=%d&area=7_412_47301_0&cat=%s", in.SkuID, in.JoinCat()))
	if err != nil {
		return nil, err
	}
	body, err = gbk2utf8(body)
	if err != nil {
		return nil, err
	}
	jdi := &jdInfo{}
	if err := json.Unmarshal(body, jdi); err != nil {
		return nil, err
	}
	if jdi.Quan[0] == '[' {
		if err := json.Unmarshal(jdi.Quan, &jdi.Quans); err != nil {
			return nil, err
		}
	} else {
		jdq := &jdQuan{}
		if err := json.Unmarshal(jdi.Quan, jdq); err != nil {
			return nil, err
		}
		jdi.Quans = append(jdi.Quans, jdq)
	}
	return jdi, nil
}

func serializeHTML(jdi *jdInfo, jdpc *jdPageConfig) string {
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

func procRequest(w http.ResponseWriter, r *http.Request) {

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

	jdp, err := getJDPrice(jdpc)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(jdp)

	jdi, err := getJDInfo(jdpc)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(jdi)

	http.HandleFunc("/", procRequest)
	log.Fatal(http.ListenAndServe(":80", nil))
}
