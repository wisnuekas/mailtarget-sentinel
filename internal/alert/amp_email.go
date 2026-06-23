package alert

import (
	"fmt"
	"net/url"
	"strconv"
)

type EmailContent struct {
	Subject  string
	BodyText string
	BodyHTML string
}

func BuildAlertEmail(a AnomalyAlert, publicBaseURL, killToken string) EmailContent {
	killURL := fmt.Sprintf("%s/api/v1/sentinel/kill-switch?token=%s&sub_account_id=%d",
		publicBaseURL,
		url.QueryEscape(killToken),
		a.SubAccountID,
	)
	actionURL := fmt.Sprintf("%s/api/v1/sentinel/kill-switch", publicBaseURL)

	subject := fmt.Sprintf("[Sentinel] Anomaly detected — Sub-account %d", a.SubAccountID)

	bodyText := fmt.Sprintf(
		"Mailtarget Sentinel detected an anomaly.\n\n"+
			"Sub-account ID: %d\n"+
			"Emails sent (5m): %d\n"+
			"Bounce rate: %.2f%%\n"+
			"Spam rate: %.2f%%\n\n"+
			"Suspend immediately: %s\n",
		a.SubAccountID, a.Sent, a.BounceRate, a.SpamRate, killURL,
	)

	bodyHTML := fmt.Sprintf(`<!doctype html>
<html ⚡4email>
<head>
  <meta charset="utf-8">
  <style amp4email-boilerplate>body{visibility:hidden}</style>
  <script async src="https://cdn.ampproject.org/v0.js"></script>
  <script async custom-element="amp-form" src="https://cdn.ampproject.org/v0/amp-form-0.1.js"></script>
</head>
<body>
  <h2>Mailtarget Sentinel — Anomaly Alert</h2>
  <p><strong>Sub-account ID:</strong> %d</p>
  <p><strong>Emails sent (5m):</strong> %d</p>
  <p><strong>Bounce rate:</strong> %.2f%%</p>
  <p><strong>Spam rate:</strong> %.2f%%</p>
  <p><strong>Detected at:</strong> %s</p>

  <form method="post" action-xhr="%s">
    <input type="hidden" name="token" value="%s">
    <input type="hidden" name="sub_account_id" value="%s">
    <button type="submit" style="background:#dc2626;color:#fff;padding:12px 24px;border:none;border-radius:6px;font-size:16px;cursor:pointer;">
      Kill Switch — Suspend Sub-account
    </button>
  </form>

  <p style="margin-top:24px;font-size:13px;color:#666;">
    HTML fallback: <a href="%s">Suspend sub-account %d</a>
  </p>
</body>
</html>`,
		a.SubAccountID,
		a.Sent,
		a.BounceRate,
		a.SpamRate,
		a.DetectedAt.UTC().Format(timeRFC3339),
		actionURL,
		killToken,
		strconv.Itoa(int(a.SubAccountID)),
		killURL,
		a.SubAccountID,
	)

	return EmailContent{
		Subject:  subject,
		BodyText: bodyText,
		BodyHTML: bodyHTML,
	}
}

const timeRFC3339 = "2006-01-02 15:04:05 UTC"
