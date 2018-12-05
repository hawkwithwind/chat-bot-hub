package chatbothub

import (
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"regexp"
	"strings"
	
	"github.com/google/uuid"
	"github.com/fluent/fluent-logger-golang/fluent"
)

type Filter interface {
	Fill(string) error
}

type BaseFilter struct {
	Id   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

func (f *BaseFilter) init(name string) string {
	f.Id = uuid.New().String()
	f.Name = name
	return f.Id
}

func (f *BaseFilter) String() string {
	jsonstr, _ := json.Marshal(f)
	return string(jsonstr)
}

type WechatBaseFilter struct {
	BaseFilter
	NextFilter Filter `json:"next"`
}

func NewWechatBaseFilter() *WechatBaseFilter {
	return &WechatBaseFilter{BaseFilter: BaseFilter{Type: "源:微信"}}
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

type PlainFilter struct {
	BaseFilter
	logger     *log.Logger
	NextFilter Filter `json:"next"`
}

func NewPlainFilter(logger *log.Logger) *PlainFilter {
	return &PlainFilter{BaseFilter: BaseFilter{Type: "过滤:空"}, logger: logger}
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
		contentptr := o.FromMap("content", body, "eventRequest.body", nil)

		fromUser := o.FromMapString("fromUser", body, "eventRequest.body", false, "")
		toUser := o.FromMapString("toUser", body, "eventRequest.body", false, "")
		groupId := o.FromMapString("groupId", body, "eventRequest.body", true, "")
		status := int64(o.FromMapFloat("status", body, "eventRequest.body", false, 0))
		timestamp := int64(o.FromMapFloat("timestamp", body, "eventRequest.body", false, 0))
		tm := o.BJTimeFromUnix(timestamp)
		mtype := int64(o.FromMapFloat("mType", body, "eventRequest.body", false, 0))

		switch content := contentptr.(type) {
		case string:
			brief = content
			if len(content) > 60 {
				brief = content[:60] + "..."
			}

			if len(groupId) > 0 {
				f.logger.Printf("%s[%s](%d) %s [%s] %s->%s (%d) %s",
					f.Name, f.Type, mtype, tm, groupId, fromUser, toUser, status, brief)
			} else {
				f.logger.Printf("%s[%s](%d) %s %s->%s (%d) %s",
					f.Name, f.Type, mtype, tm, fromUser, toUser, status, brief)
			}
		case map[string]interface{}:
			f.logger.Printf("%s[%s](%d) %s %s->%s (%d) appmsg: %v",
				f.Name, f.Type, mtype, tm, fromUser, toUser, status, content)
		default:
			f.logger.Printf("%s[%s](%d) %s %s->%s (%d) %T %v",
				f.Name, f.Type, mtype, tm, fromUser, toUser, status, content, content)
		}
	} else {
		f.logger.Printf("%s[%s] %s ...", f.Name, f.Type, brief)
	}

	if f.NextFilter != nil && o.Err == nil {
		return f.NextFilter.Fill(msg)
	}

	return o.Err
}

type FluentFilter struct {
	BaseFilter
	logger *fluent.Fluent
	tag string
	NextFilter Filter `json:"next"`
}

func NewFluentFilter(logger *fluent.Fluent, tag string) *FluentFilter {
	return &FluentFilter{BaseFilter: BaseFilter{Type: "过滤:Fluent"}, logger: logger, tag: tag}
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
				fmt.Println("[FLUENT] logged body")
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

func NewRegexRouter() *RegexRouter {
	return &RegexRouter{BaseFilter: BaseFilter{Type: "路由:正则"}}
}

func (f *RegexRouter) String() string {
	jsonstr, _ := json.Marshal(f)
	return string(jsonstr)
}

func (f *RegexRouter) DefaultNext(filter Filter) error {
	if f == nil {
		return fmt.Errorf("call on empty *RegexRouter")
	}
	f.DefaultNextFilter = filter
	return nil
}

func (f *RegexRouter) Next(regstr string, filter Filter) error {
	if f == nil {
		return fmt.Errorf("call on empty *RegexRouter")
	}
	if f.NextFilter == nil {
		f.NextFilter = make(map[string]Filter)
	}
	if f.compiledRegexp == nil {
		f.compiledRegexp = make(map[string]*regexp.Regexp)
	}

	compiledregexp := regexp.MustCompile(regstr)

	f.NextFilter[regstr] = filter
	f.compiledRegexp[regstr] = compiledregexp
	return nil
}

func (f *RegexRouter) Fill(msg string) error {
	if f == nil {
		return fmt.Errorf("call on empty *RegexRouter")
	}
	if f.NextFilter == nil {
		return nil
	}

	for k, v := range f.NextFilter {
		if cr, found := f.compiledRegexp[k]; found {
			if cr.MatchString(msg) {
				if v != nil {
					return v.Fill(msg)
				}
			}
		}
	}

	if f.DefaultNextFilter != nil {
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
	DefaultNextFilter Filter                       `json:"defaultNext"`
}

func NewKVRouter() *KVRouter {
	return &KVRouter{BaseFilter: BaseFilter{Type: "路由:字典"}}
}

func (f *KVRouter) String() string {
	jsonstr, _ := json.Marshal(f)
	return string(jsonstr)
}

func (f *KVRouter) Next(name string, value string, filter Filter) error {
	if f == nil {
		return fmt.Errorf("call on empty *KVRouter")
	}

	if f.NextFilter == nil {
		f.NextFilter = make(map[string]map[string]Filter)
	}

	if _, found := f.NextFilter[name]; !found {
		f.NextFilter[name] = make(map[string]Filter)
	}

	f.NextFilter[name][value] = filter
	return nil
}

func (f *KVRouter) DefaultNext(filter Filter) error {
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
		if value := findByJsonPath(body, k); value != nil {
			var s string
			if reflect.TypeOf(value) == reflect.TypeOf(s) {
				if filter, found := vmaps[value.(string)]; found {
					fillOnce = true
					if filter != nil {
						if err := filter.Fill(msg); err != nil {
							errlist = append(errlist, err)
						}
					}
				}
			} else {
				errlist = append(errlist, fmt.Errorf("key[%s] = %v; type string expected", k, value))
			}
		}
	}

	if !fillOnce {
		if f.DefaultNextFilter != nil {
			return f.DefaultNextFilter.Fill(msg)
		} else {
			return nil
		}
	}

	if len(errlist) == 0 {
		return nil
	} else {
		return fmt.Errorf("multiple error occured while trigger filters %v", errlist)
	}
}
