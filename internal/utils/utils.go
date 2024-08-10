package utils

import (
	"github.com/dlclark/regexp2"
	"github.com/gocolly/colly/v2"
	"strings"
)

var (
	pcppURLMatcher          = regexp2.MustCompile(`^(https?://)?([a-z]{2}\.)?pcpartpicker\.com(/.*)?$`, 0)
	productURLMatcher       = regexp2.MustCompile(`^(https?://)?([a-z]{2}\.)?pcpartpicker\.com/product/[a-zA-Z0-9]{4,8}/[\S]*`, 0)
	partListURLMatcher      = regexp2.MustCompile(`^(https?://)?([a-z]{2}\.)?pcpartpicker\.com/((list/[a-zA-Z0-9]{4,8})|((user/\w*/saved/(#view=)?[a-zA-Z0-9]{4,8})))`, 0)
	vendorNameMatcher       = regexp2.MustCompile(`(?<=pcpartpicker\.com/mr/).*(?=\/)`, 0)
	pcppUserSavedURLMatcher = regexp2.MustCompile(`^(https?://)?([a-z]{2}\.)?pcpartpicker\.com/user/[a-zA-Z0-9]*/saved/#view=[a-zA-Z0-9]{4,8}`, 0)
	scriptImageCheck        = regexp2.MustCompile(`(?<=src:\s").*(?=")`, 0)
)

func ExtractVendorName(URL string) string {
	if URL == "" {
		return ""
	}
	m, err := vendorNameMatcher.FindStringMatch(URL)
	if err != nil || m == nil {
		return ""
	}
	return m.String()
}

func ConvertListURL(URL string) string {
	match, _ := pcppUserSavedURLMatcher.MatchString(URL)

	if !match {
		return URL
	}

	return strings.Replace(URL, "#view=", "", 1)
}

func MatchPCPPURL(URL string) bool {
	match, _ := pcppURLMatcher.MatchString(URL)

	return match
}

func MatchProductURL(URL string) bool {
	match, _ := productURLMatcher.MatchString(URL)

	return match
}

func MatchPartListURL(URL string) bool {
	match, _ := partListURLMatcher.MatchString(URL)

	return match
}

func ExtractPartListURLs(text string) []string {
	return Regexp2SearchAllText(partListURLMatcher, text)
}

func FindScriptImages(script *colly.HTMLElement, images []string) []string {
	for _, match := range Regexp2SearchAllText(scriptImageCheck, script.Text) {
		if strings.HasPrefix(match, "//") {
			match = "https:" + match
		}
		images = append(images, match)
	}
	return images
}

func Regexp2SearchAllText(re *regexp2.Regexp, s string) []string {
	var matches []string
	m, _ := re.FindStringMatch(s)
	for m != nil {
		matches = append(matches, m.String())
		m, _ = re.FindNextMatch(m)
	}
	return matches
}

func BuildPrefixURL(region string) string {
	if region != "" && region != "us" {
		region += "."
	} else {
		region = ""
	}
	prefixURL := "https://" + region + "pcpartpicker.com/"
	return prefixURL
}
