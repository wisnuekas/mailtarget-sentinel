package alert

import (
	"fmt"
	"net/url"
	"strconv"
)

type ResumeEmailInput struct {
	SubAccountID int32
	CompanyID    int32
}

func BuildResumeEmail(in ResumeEmailInput, publicBaseURL, resumeToken string) EmailContent {
	resumeURL := fmt.Sprintf("%s/api/v1/sentinel/resume-switch?token=%s&sub_account_id=%d",
		publicBaseURL,
		url.QueryEscape(resumeToken),
		in.SubAccountID,
	)
	actionURL := fmt.Sprintf("%s/api/v1/sentinel/resume-switch", publicBaseURL)

	subject := fmt.Sprintf("[Sentinel] Sub-account %d suspended — Resume when ready", in.SubAccountID)

	bodyText := fmt.Sprintf(
		"Mailtarget Sentinel kill switch was activated.\n\n"+
			"Sub-account ID: %d has been suspended.\n\n"+
			"When the issue is resolved, resume sending:\n%s\n",
		in.SubAccountID, resumeURL,
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
  <h2>Mailtarget Sentinel — Sub-account Suspended</h2>
  <p>Sub-account <strong>%d</strong> was suspended via kill switch.</p>
  <p>When you are ready to resume sending, use the button below:</p>

  <form method="post" action-xhr="%s">
    <input type="hidden" name="token" value="%s">
    <input type="hidden" name="sub_account_id" value="%s">
    <button type="submit" style="background:#16a34a;color:#fff;padding:12px 24px;border:none;border-radius:6px;font-size:16px;cursor:pointer;">
      Resume Sub-account
    </button>
  </form>

  <p style="margin-top:24px;font-size:13px;color:#666;">
    HTML fallback: <a href="%s">Resume sub-account %d</a>
  </p>
</body>
</html>`,
		in.SubAccountID,
		actionURL,
		resumeToken,
		strconv.Itoa(int(in.SubAccountID)),
		resumeURL,
		in.SubAccountID,
	)

	return EmailContent{
		Subject:  subject,
		BodyText: bodyText,
		BodyHTML: bodyHTML,
	}
}
