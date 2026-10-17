package main

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// i18n is joucho's few strings. PAM's own messages ("Password:",
// "Login incorrect") pass through in the language the system configured.
type i18n struct{ lang string }

func newI18n(setting string) *i18n {
	lang := "en"
	switch v := strings.ToLower(setting); {
	case setting == "en", setting == "zh_CN":
		lang = setting
	case setting == "auto" || setting == "":
		for _, env := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
			if v := os.Getenv(env); v != "" {
				if strings.HasPrefix(strings.ToLower(v), "zh") {
					lang = "zh_CN"
				}
				break
			}
		}
	case strings.HasPrefix(v, "zh"):
		lang = "zh_CN"
	}
	return &i18n{lang: lang}
}

var zhCN = map[string]string{
	"Password":                       "密码",
	"Sign in":                        "登录",
	"Session":                        "会话",
	"User":                           "用户",
	"Checking…":                      "正在验证…",
	"Suspend":                        "休眠",
	"Restart":                        "重启",
	"Shut down":                      "关机",
	"Shutting down…":                 "正在关机…",
	"Restarting…":                    "正在重启…",
	"Suspending…":                    "正在休眠…",
	"No permission.":                 "没有权限。",
	"Lost the connection to greetd.": "与 greetd 的连接中断。",
	"greetd did not answer: %v":      "greetd 没有应答：%v",
	"no regular users found":         "没有找到普通用户",
	"no sessions in %s":              "在 %s 里没有找到会话",
}

// T translates a string of ours.
func (t *i18n) T(s string) string {
	if t.lang == "zh_CN" {
		if v, ok := zhCN[s]; ok {
			return v
		}
	}
	return s
}

// Tf is T for a format string.
func (t *i18n) Tf(format string, a ...any) string {
	tr := t.T(format)
	if tr == format {
		return fmt.Sprintf(format, a...)
	}
	return fmt.Sprintf(tr, a...) // zh: same verbs, translated text
}

var weekdaysZH = [...]string{"星期一", "星期二", "星期三", "星期四", "星期五", "星期六", "星期日"}

// date is the line under the clock.
func (t *i18n) date(now time.Time) string {
	if t.lang == "zh_CN" {
		return fmt.Sprintf("%d月%d日 %s", int(now.Month()), now.Day(), weekdaysZH[(int(now.Weekday())+6)%7])
	}
	return now.Format("Monday, January 2")
}

// clock is the time on the card.
func (t *i18n) clock(now time.Time, seconds bool) string {
	if seconds {
		return now.Format("15:04:05")
	}
	return now.Format("15:04")
}
