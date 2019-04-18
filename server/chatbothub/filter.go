package chatbothub

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http/cookiejar"
	"net/url"
	"reflect"
	"regexp"
	"strings"

	"github.com/fluent/fluent-logger-golang/fluent"

	"github.com/hawkwithwind/chat-bot-hub/server/domains"
	"github.com/hawkwithwind/chat-bot-hub/server/httpx"
)

type Filter interface {
	Fill(string) error
	Next(Filter) error
}

type BranchTag struct {
	Key   string
	Value string
}

type Router interface {
	Filter
	Branch(tag BranchTag, filter Filter) error
}

type BaseFilter struct {
	Id   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

const (
	WECHATBASEFILTER   string = "WechatBaseFilter"
	WECHATMOMENTFILTER string = "WechatMomentFilter"
	PLAINFILTER        string = "PlainFilter"
	FLUENTFILTER       string = "FluentFilter"
	REGEXROUTER        string = "RegexRouter"
	KVROUTER           string = "KVRouter"
	WEBTRIGGER         string = "WebTrigger"
)

func NewBaseFilter(filterId string, filterName string, filterType string) BaseFilter {
	return BaseFilter{
		Id:   filterId,
		Name: filterName,
		Type: filterType,
	}
}

func (f *BaseFilter) String() string {
	jsonstr, _ := json.Marshal(f)
	return string(jsonstr)
}

type WechatBaseFilter struct {
	BaseFilter
	NextFilter Filter `json:"next"`
}

func NewWechatBaseFilter(filterId string, filterName string) *WechatBaseFilter {
	return &WechatBaseFilter{BaseFilter: NewBaseFilter(filterId, filterName, "消息源:微信")}
}

func (f *WechatBaseFilter) String() string {
	jsonstr, _ := json.Marshal(f)
	return string(jsonstr)
}

func (f *WechatBaseFilter) Next(filter Filter) error {
	if f == nil {
		return fmt.Errorf("call on empty *WechatBaseFilter")
	}
	f.NextFilter = filter
	return nil
}

func (f *WechatBaseFilter) Fill(msg string) error {
	if f == nil {
		return fmt.Errorf("call on empty *WechatBaseFilter")
	}

	if f.NextFilter != nil {
		return f.NextFilter.Fill(msg)
	}

	return nil
}

type WechatMomentFilter struct {
	BaseFilter
	NextFilter Filter `json:"next"`
}

func NewWechatMomentFilter(filterId string, filterName string) *WechatMomentFilter {
	return &WechatMomentFilter{BaseFilter: NewBaseFilter(filterId, filterName, "动态源:微信")}
}

func (f *WechatMomentFilter) String() string {
	jsonstr, _ := json.Marshal(f)
	return string(jsonstr)
}

func (f *WechatMomentFilter) Next(filter Filter) error {
	if f == nil {
		return fmt.Errorf("call on empty *WechatMomentFilter")
	}
	f.NextFilter = filter
	return nil
}

func (f *WechatMomentFilter) Fill(msg string) error {
	if f == nil {
		return fmt.Errorf("call on empty *WechatMomentFilter")
	}

	if f.NextFilter != nil {
		return f.NextFilter.Fill(msg)
	}

	return nil
}

type PlainFilter struct {
	BaseFilter
	logger     *log.Logger
	NextFilter Filter `json:"next"`
}

func NewPlainFilter(filterId string, filterName string, logger *log.Logger) *PlainFilter {
	return &PlainFilter{BaseFilter: NewBaseFilter(filterId, filterName, "过滤:空"), logger: logger}
}

func (f *PlainFilter) String() string {
	jsonstr, _ := json.Marshal(f)
	return string(jsonstr)
}

func (f *PlainFilter) Next(filter Filter) error {
	if f == nil {
		return fmt.Errorf("call on empty *PlainFilter")
	}
	f.NextFilter = filter
	return nil
}

type WechatMsgSource struct {
	AtUserList  string `xml:"atuserlist" json:"atUserList"`
	Silence     int    `xml:"silence" json:"silence"`
	MemberCount int    `xml:"membercount" json:"memberCount"`
}

func (f *PlainFilter) Fill(msg string) error {
	if f == nil {
		return fmt.Errorf("call on empty *TextPlainFilter")
	}
	brief := msg
	if len(msg) > 80 {
		brief = msg[:80]
	}

	o := &ErrorHandler{}
	body := o.FromJson(msg)
	if body != nil {
		contentptr := o.FromMap("content", body, "body", nil)

		fromUser := o.FromMapString("fromUser", body, "body", false, "")
		toUser := o.FromMapString("toUser", body, "body", false, "")
		groupId := o.FromMapString("groupId", body, "body", true, "")
		status := int64(o.FromMapFloat("status", body, "body", true, 0))
		//timestamp := int64(o.FromMapFloat("timestamp", body, "eventRequest.body", false, 0))
		//tm := o.BJTimeFromUnix(timestamp)
		mtype := int64(o.FromMapFloat("mType", body, "body", false, 0))
		msgsourcexml := o.FromMapString("msgSource", body, "body", true, "")

		if msgsourcexml != "" {
			var msgSource WechatMsgSource
			o.FromXML(msgsourcexml, &msgSource)
			if o.Err != nil {
				f.logger.Printf("err %v\n%s\n", o.Err, msgsourcexml)
			} else {
				body["msgSource"] = msgSource
			}
		}

		switch content := contentptr.(type) {
		case string:
			brief = content
			if len(content) > 480 {
				brief = content[:480] + "..."
			}

			if len(groupId) > 0 {
				f.logger.Printf("%s[%s](%d) [%s] %s->%s (%d) %s",
					f.Name, f.Type, mtype, groupId, fromUser, toUser, status, brief)
			} else {
				f.logger.Printf("%s[%s](%d) %s->%s (%d) %s",
					f.Name, f.Type, mtype, fromUser, toUser, status, brief)
			}

		case map[string]interface{}:
			var msg WechatMsg
			o.Err = json.Unmarshal([]byte(o.ToJson(content["msg"])), &msg)
			if len(msg.AppMsg.Title) > 0 {
				f.logger.Printf("%s[%s](%d) %s->%s (%d) appmsg: <%s>%s",
					f.Name, f.Type, mtype, fromUser, toUser, status, msg.AppMsg.SourceDisplayName, msg.AppMsg.Title)
			} else if len(msg.Emoji.Attributions.FromUserName) > 0 {
				f.logger.Printf("%s[%s](%d) %s->%s (%d) emoji: <%s>%s",
					f.Name, f.Type, mtype, fromUser, toUser, status, msg.Emoji.Attributions.Type, msg.Emoji.Attributions.ProductId)
			}

		default:
			f.logger.Printf("%s[%s](%d) %s->%s (%d) %T %v",
				f.Name, f.Type, mtype, fromUser, toUser, status, content, content)
		}
	} else {
		f.logger.Printf("%s[%s] %s ...", f.Name, f.Type, brief)
	}

	if f.NextFilter != nil && o.Err == nil {
		return f.NextFilter.Fill(o.ToJson(body))
	}

	return o.Err
}

type FluentFilter struct {
	BaseFilter
	logger     *fluent.Fluent
	tag        string
	NextFilter Filter `json:"next"`
}

func NewFluentFilter(filterId string, filterName string, logger *fluent.Fluent, tag string) *FluentFilter {
	return &FluentFilter{BaseFilter: NewBaseFilter(filterId, filterName, "过滤:Fluent"), logger: logger, tag: tag}
}

func (f *FluentFilter) String() string {
	jsonstr, _ := json.Marshal(f)
	return string(jsonstr)
}

func (f *FluentFilter) Next(filter Filter) error {
	if f == nil {
		return fmt.Errorf("call on empty *FluentFilter")
	}
	f.NextFilter = filter
	return nil
}

func (f *FluentFilter) Fill(msg string) error {
	if f == nil {
		return fmt.Errorf("call on empty *FluentFilter")
	}

	o := &ErrorHandler{}
	body := o.FromJson(msg)

	if o.Err == nil {
		go func() {
			if body != nil {
				f.logger.Post(f.tag, body)
			}
		}()

		if f.NextFilter != nil {
			return f.NextFilter.Fill(msg)
		}
	} else {
		return o.Err
	}

	return nil
}

type RegexRouter struct {
	BaseFilter
	NextFilter        map[string]Filter `json:"next"`
	compiledRegexp    map[string]*regexp.Regexp
	DefaultNextFilter Filter `json:"defaultNext"`
}

func NewRegexRouter(filterId string, filterName string) *RegexRouter {
	return &RegexRouter{BaseFilter: NewBaseFilter(filterId, filterName, "路由:正则")}
}

func (f *RegexRouter) String() string {
	jsonstr, _ := json.Marshal(f)
	return string(jsonstr)
}

func (f *RegexRouter) Next(filter Filter) error {
	if f == nil {
		return fmt.Errorf("call on empty *RegexRouter")
	}
	f.DefaultNextFilter = filter
	return nil
}

func (f *RegexRouter) Branch(tag BranchTag, filter Filter) error {
	if f == nil {
		return fmt.Errorf("call on empty *RegexRouter")
	}

	if f.NextFilter == nil {
		f.NextFilter = make(map[string]Filter)
	}
	if f.compiledRegexp == nil {
		f.compiledRegexp = make(map[string]*regexp.Regexp)
	}

	compiledregexp := regexp.MustCompile(tag.Key)

	f.NextFilter[tag.Key] = filter
	f.compiledRegexp[tag.Key] = compiledregexp
	return nil
}

func (f *RegexRouter) Fill(msg string) error {
	if f == nil {
		return fmt.Errorf("call on empty *RegexRouter")
	}
	if f.NextFilter == nil {
		return nil
	}

	fmt.Printf("[FILTER DEBUG] matching %s\n", msg)

	for k, v := range f.NextFilter {
		if cr, found := f.compiledRegexp[k]; found {
			if cr.MatchString(msg) {
				if v != nil {
					fmt.Printf("[FILTER DEBUG][%s][%s] filled\n", f.Name, k)
					return v.Fill(msg)
				}
			}
		}
	}

	if f.DefaultNextFilter != nil {
		fmt.Printf("[FILTER DEBUG][%s][default] filled\n", f.Name)
		return f.DefaultNextFilter.Fill(msg)
	}

	return nil
}

func findByJsonPath(body map[string]interface{}, name string) interface{} {
	ks := strings.Split(name, ".")
	step := body
	var found bool
	var part interface{}
	for kn := range ks {
		if part, found = step[ks[kn]]; found {
			var m map[string]interface{}
			if reflect.TypeOf(part) == reflect.TypeOf(m) {
				step = part.(map[string]interface{})
			} else {
				if kn == len(ks)-1 {
					return part
				} else {
					return nil
				}
			}
		} else {
			return nil
		}
	}

	return step
}

type KVRouter struct {
	BaseFilter
	NextFilter        map[string]map[string]Filter `json:"next"`
	compiledRegexp    map[string]*regexp.Regexp
	DefaultNextFilter Filter                       `json:"defaultNext"`
}

func NewKVRouter(filterId string, filterName string) *KVRouter {
	return &KVRouter{BaseFilter: NewBaseFilter(filterId, filterName, "路由:字典")}
}

func (f *KVRouter) String() string {
	jsonstr, _ := json.Marshal(f)
	return string(jsonstr)
}

func (f *KVRouter) Branch(tag BranchTag, filter Filter) error {
	if f == nil {
		return fmt.Errorf("call on empty *KVRouter")
	}

	if f.NextFilter == nil {
		f.NextFilter = make(map[string]map[string]Filter)
	}

	if _, found := f.NextFilter[tag.Key]; !found {
		f.NextFilter[tag.Key] = make(map[string]Filter)
	}

	if f.compiledRegexp == nil {
		f.compiledRegexp = make(map[string]*regexp.Regexp)
	}
	
	compiledregexp := regexp.MustCompile(tag.Value)

	f.NextFilter[tag.Key][tag.Value] = filter
	f.compiledRegexp[tag.Value] = compiledregexp
	return nil
}

func (f *KVRouter) Next(filter Filter) error {
	if f == nil {
		return fmt.Errorf("call on empty *KVRouter")
	}
	f.DefaultNextFilter = filter
	return nil
}

func (f *KVRouter) Fill(msg string) error {
	if f == nil {
		return fmt.Errorf("call on empty *KVRouter")
	}

	fmt.Printf("[FILTER DEBUG] KVRouter received\n%s\n", msg)
	
	o := ErrorHandler{}
	body := o.FromJson(msg)

	if o.Err != nil {
		return o.Err
	}
	if f.NextFilter == nil {
		return nil
	}
	if body == nil {
		return nil
	}

	errlist := make([]error, 0)
	fillOnce := false

	for k, vmaps := range f.NextFilter {
		value := findByJsonPath(body, k)
		var valuestring string
		switch vstr := value.(type) {
		case string:
			valuestring = vstr
		default:
			valuestring = ""
		}

		for regstr, nextfilter := range vmaps {
			if cr, found := f.compiledRegexp[regstr]; found {
				if cr.MatchString(valuestring) {
					fillOnce = true
					if nextfilter != nil {
						if err := nextfilter.Fill(msg); err != nil {
							errlist = append(errlist, err)
						} else {
							fmt.Printf("[FILTER DEBUG][%s][%s][%s] filled\n", f.Name, k, regstr)
						}
					}
				}
			}
		}
	}

	if !fillOnce {
		if f.DefaultNextFilter != nil {
			fmt.Printf("[FILTER DEBUG][%s][default] filled\n", f.Name)
			return f.DefaultNextFilter.Fill(msg)
		} else {
			fmt.Printf("[FILTER DEBUG][%s][default] is null\n", f.Name)
			return nil
		}
	}

	if len(errlist) == 0 {
		return nil
	} else {
		return fmt.Errorf("error occured while trigger filters %v", errlist)
	}
}

type WebAction struct {
	Url    string `json:"url"`
	Method string `json:"method"`
}

type WebTrigger struct {
	BaseFilter
	NextFilter Filter    `json:"next"`
	Action     WebAction `json:"action"`
}

func (f *WebTrigger) String() string {
	jsonstr, _ := json.Marshal(f)
	return string(jsonstr)
}

func NewWebTrigger(filterId string, filterName string) *WebTrigger {
	return &WebTrigger{BaseFilter: NewBaseFilter(filterId, filterName, "触发器:Web")}
}

func (f *WebTrigger) Fill(msg string) error {
	go func() {
		o := domains.ErrorHandler{}

		jar, err := cookiejar.New(nil)
		if err != nil {
			return
		}

		var u *url.URL
		u, err = url.Parse(f.Action.Url)
		if err != nil {
			fmt.Printf("[WebTrigger] failed parse url %s\n%s\n", f.Action.Url, err)
			return
		}
		domain := strings.Split(u.Host, ":")[0]

		// parse fromUser toUser groupId from msg, and init cookie struct
		header := o.ChatMessageHeaderFromMessage(msg)

		// load cookies
		cookies := o.LoadWebTriggerCookies(chathub.redispool, header, domain)

		jar.SetCookies(u, cookies)

		rr := httpx.NewRestfulRequest(f.Action.Method, f.Action.Url)
		rr.Params["msg"] = msg
		rr.CookieJar = jar

		if resp, err := httpx.RestfulCallRetry(rr, 5, 1); err != nil {
			fmt.Printf("[WebTrigger] failed %s\n%v\n", err, resp)
		} else {
			//save cookies
			o.SaveWebTriggerCookies(chathub.redispool, header, domain, resp.Cookies)
			if o.Err != nil {
				fmt.Printf("[WebTrigger] save cookie failed %s\n", o.Err)
			}

			fmt.Printf("[WebTrigger DEBUG] trigger %s returned\n%s\n", f.Action.Url, o.ToJson(resp))
		}
	}()

	if f.NextFilter != nil {
		return f.NextFilter.Fill(msg)
	}

	return nil
}

func (f *WebTrigger) Next(filter Filter) error {
	if f == nil {
		return fmt.Errorf("call on empty *WebTrigger")
	}
	f.NextFilter = filter
	return nil
}
