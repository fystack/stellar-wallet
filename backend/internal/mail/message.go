package mail

import (
	"html/template"
	"strings"
)

const product = "Stellar MPC Wallet"

type Detail struct {
	Label string
	Value string
	Mono  bool
}

// Message is one transactional email; Text and HTML render the same content for a multipart body.
type Message struct {
	Subject     string
	Heading     string
	Intro       string
	Details     []Detail
	ActionLabel string
	ActionURL   string
	Note        string
}

func (m Message) Text() string {
	var b strings.Builder
	b.WriteString(m.Heading + "\n\n" + m.Intro + "\n")
	if len(m.Details) > 0 {
		b.WriteString("\n")
		for _, d := range m.Details {
			b.WriteString(d.Label + ": " + d.Value + "\n")
		}
	}
	b.WriteString("\n" + m.ActionLabel + ":\n" + m.ActionURL + "\n")
	if m.Note != "" {
		b.WriteString("\n" + m.Note + "\n")
	}
	b.WriteString("\n" + product + "\n")
	return b.String()
}

func (m Message) HTML() string {
	var b strings.Builder
	if err := page.Execute(&b, struct {
		Message
		Product string
	}{m, product}); err != nil {
		return "<pre>" + template.HTMLEscapeString(m.Text()) + "</pre>"
	}
	return b.String()
}

var page = template.Must(template.New("mail").Parse(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width">
<title>{{.Subject}}</title>
</head>
<body style="margin:0;padding:0;background:#f7f8fa;">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background:#f7f8fa;padding:32px 16px;">
<tr><td align="center">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="max-width:520px;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;color:#0f2748;">
<tr><td style="padding:0 0 16px;font-size:13px;font-weight:700;letter-spacing:.04em;text-transform:uppercase;color:#627186;">{{.Product}}</td></tr>
<tr><td style="background:#ffffff;border:1px solid #eceef2;padding:28px 28px 24px;">
<h1 style="margin:0 0 12px;font-size:20px;line-height:1.3;font-weight:700;">{{.Heading}}</h1>
<p style="margin:0 0 20px;font-size:15px;line-height:1.5;color:#5b6b82;">{{.Intro}}</p>
{{if .Details}}<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="margin:0 0 20px;background:#f4f6f9;font-size:14px;">
{{range .Details}}<tr>
<td style="padding:10px 14px;border-bottom:1px solid #eceef2;color:#627186;white-space:nowrap;vertical-align:top;">{{.Label}}</td>
<td style="padding:10px 14px;border-bottom:1px solid #eceef2;font-weight:600;word-break:break-all;{{if .Mono}}font-family:ui-monospace,SFMono-Regular,Menlo,Consolas,monospace;font-size:13px;{{end}}">{{.Value}}</td>
</tr>{{end}}
</table>{{end}}
<table role="presentation" cellpadding="0" cellspacing="0" style="margin:0 0 16px;"><tr>
<td style="background:#1f6bff;"><a href="{{.ActionURL}}" style="display:inline-block;padding:12px 20px;font-size:14px;font-weight:600;color:#ffffff;text-decoration:none;">{{.ActionLabel}}</a></td>
</tr></table>
<p style="margin:0;font-size:12px;line-height:1.5;color:#627186;word-break:break-all;">Or open this link: <a href="{{.ActionURL}}" style="color:#1f6bff;">{{.ActionURL}}</a></p>
</td></tr>
{{if .Note}}<tr><td style="padding:16px 4px 0;font-size:12px;line-height:1.5;color:#627186;">{{.Note}}</td></tr>{{end}}
</table>
</td></tr>
</table>
</body>
</html>
`))
