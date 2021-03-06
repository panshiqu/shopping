package define

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
)

// ErrDataSame .
var ErrDataSame = errors.New("Data Same")

// ErrNotExist .
var ErrNotExist = errors.New("Not Exist")

// ErrAlreadyExist .
var ErrAlreadyExist = errors.New("Already Exist")

// ErrToSmallPriority .
var ErrToSmallPriority = errors.New("To Small Priority")

// ErrIllegalLen .
var ErrIllegalLen = errors.New("illegal len")

// ErrIllegalAlias .
var ErrIllegalAlias = errors.New("illegal alias")

// ErrIllegalPassword .
var ErrIllegalPassword = errors.New("illegal password")

// IndexData 首页数据
type IndexData struct {
	Args  []*IndexArgs
	Alias string
	Proms map[string]int
}

// IndexArgs 首页参数
type IndexArgs struct {
	SkuID     int64
	Price     float64
	Content   string
	MinPrice  float64
	MaxPrice  float64
	Timestamp string
	Duration  string
	Sampling  int64
	Name      string

	InsertTimestamp int64
}

// IsMinPrice .
func (i *IndexArgs) IsMinPrice() bool {
	return i.MinPrice != i.MaxPrice && i.MinPrice == i.Price
}

// JDPageConfig 页面配置
type JDPageConfig struct {
	SkuID       int64
	Name        string
	KoBeginTime int64
	KoEndTime   int64
	Src         string
	Cat         []int64
}

// JDPrice 价格
type JDPrice struct {
	Price       string `json:"p"`
	OriginPrice string `json:"op"`
}

// JDInfo 促销信息
type JDInfo struct {
	Quan       json.RawMessage `json:"quan"`
	SkuCoupon  []*JDSkuCoupon  `json:"skuCoupon"`
	AdsStatus  int64           `json:"adsStatus"`
	Ads        []*JDAds        `json:"ads"`
	QuanStatus int64           `json:"quanStatus"`
	PromStatus int64           `json:"promStatus"`
	Prom       *JDProm         `json:"prom"`

	Quans []*JDQuan
}

// JDQuan .
type JDQuan struct {
	Title  string `json:"title"`
	ActURL string `json:"actUrl"`
}

// JDSkuCoupon .
type JDSkuCoupon struct {
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

// JDAds .
type JDAds struct {
	ID string `json:"id"`
	Ad string `json:"ad"`
}

// JDProm .
type JDProm struct {
	Hit        int64           `json:"hit"`
	PickOneTag []*JDTag        `json:"pickOneTag"`
	CarGift    int64           `json:"carGift"`
	Tags       []*JDTag        `json:"tags"`
	GiftPool   json.RawMessage `json:"giftPool"`
	Ending     int64           `json:"ending"`
}

// JDTag .
type JDTag struct {
	D       string `json:"d"`
	St      string `json:"st"`
	Code    string `json:"code"`
	Content string `json:"content"`
	Tr      int64  `json:"tr"`
	AdURL   string `json:"adurl,omitempty"`
	Name    string `json:"name"`
	Pid     string `json:"pid"`

	Gifts []*JDGift `json:"gifts"`
}

// JDGift .
type JDGift struct {
	Gs  int64  `json:"gs"`
	Nm  string `json:"nm"`
	Sid string `json:"sid"`
	Ss  int64  `json:"ss"`
	Gt  int64  `json:"gt"`
	Mp  string `json:"mp"`
	Num int64  `json:"num"`
}

// JDGlobalBuy .
type JDGlobalBuy struct {
	Success bool      `json:"success"`
	TaxTxt  *JDTaxTxt `json:"taxTxt"`
}

// JDTaxTxt .
type JDTaxTxt struct {
	Content string `json:"content"`
}

// JoinCat .
func (j *JDPageConfig) JoinCat() []byte {
	if len(j.Cat) == 0 {
		return nil
	}
	var buf bytes.Buffer
	for _, v := range j.Cat {
		fmt.Fprint(&buf, v, ",")
	}
	return buf.Bytes()[:buf.Len()-1]
}

// TagsSlice .
type TagsSlice []*JDTag

func (t TagsSlice) Len() int {
	return len(t)
}

func (t TagsSlice) Less(i, j int) bool {
	if t[i].Code != t[j].Code {
		return t[i].Code < t[j].Code
	}
	return t[i].Pid < t[j].Pid
}

func (t TagsSlice) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}
