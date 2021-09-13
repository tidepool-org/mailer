package templates

import (
	_ "embed"
	"github.com/vanng822/go-premailer/premailer"
	"strings"
)

const (
	cssRef = "<link rel=\"stylesheet\" type=\"text/css\" href=\"/css/styles.css\" />"
	replacementStartTag = "<style type=\"text/css\">"
	replacementEndTag = "</style>"
)

var opts = premailer.Options{
	RemoveClasses:     true,
	CssToAttributes:   true,
	KeepBangImportant: true,
}

//go:embed sources/css/styles.css
var styles []byte

func inlineCSS(html []byte) (string, error){
	replacement := strings.Join([]string{replacementStartTag, string(styles), replacementEndTag}, "\n")
	body := strings.ReplaceAll(string(html), cssRef, replacement)
	prem, err := premailer.NewPremailerFromString(body, &opts)
	if err != nil {
		return "", err
	}

	return prem.Transform()
}
