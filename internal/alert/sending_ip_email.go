package alert

import "fmt"

type SendingIPEmailInput struct {
	CompanyID    int32
	CompanyName  string
	SendingIP    string
	Sent         uint64
	Bounced      uint64
	SpamBounced  uint64
	BounceRate   float64
	SpamRate     float64
}

func BuildSendingIPEmail(in SendingIPEmailInput) EmailContent {
	companyLabel := in.CompanyName
	if companyLabel == "" {
		companyLabel = fmt.Sprintf("Company %d", in.CompanyID)
	}

	subject := fmt.Sprintf("[Sentinel] At-risk sending IP — %s (%s)", in.SendingIP, companyLabel)

	bodyText := fmt.Sprintf(
		"Mailtarget Sentinel — Sending IP Review Required\n\n"+
			"Company: %s (ID %d)\n"+
			"Sending IP: %s\n"+
			"Emails sent (5m): %d\n"+
			"Bounce rate: %.2f%%\n"+
			"Spam rate: %.2f%%\n\n"+
			"This sending IP exceeded configured thresholds. "+
			"Please investigate reputation, routing, and recent sending activity.\n",
		companyLabel, in.CompanyID, in.SendingIP,
		in.Sent, in.BounceRate, in.SpamRate,
	)

	bodyHTML := fmt.Sprintf(`<!doctype html>
<html>
<body style="font-family:system-ui,sans-serif;color:#1e293b;">
  <h2 style="color:#dc2626;">Mailtarget Sentinel — At-risk Sending IP</h2>
  <p>Sending IP <strong>%s</strong> for <strong>%s</strong> requires Ops review.</p>
  <table style="border-collapse:collapse;margin:16px 0;">
    <tr><td style="padding:4px 12px 4px 0;color:#64748b;">Company</td><td>%s (ID %d)</td></tr>
    <tr><td style="padding:4px 12px 4px 0;color:#64748b;">Sending IP</td><td><strong>%s</strong></td></tr>
    <tr><td style="padding:4px 12px 4px 0;color:#64748b;">Sent (5m)</td><td>%d</td></tr>
    <tr><td style="padding:4px 12px 4px 0;color:#64748b;">Bounce rate</td><td style="color:#dc2626;font-weight:600;">%.2f%%</td></tr>
    <tr><td style="padding:4px 12px 4px 0;color:#64748b;">Spam rate</td><td>%.2f%%</td></tr>
  </table>
  <p style="color:#64748b;font-size:14px;">
    Please investigate IP reputation, pool assignment, and recent traffic before further delivery.
  </p>
</body>
</html>`,
		in.SendingIP, companyLabel,
		companyLabel, in.CompanyID, in.SendingIP,
		in.Sent, in.BounceRate, in.SpamRate,
	)

	return EmailContent{
		Subject:  subject,
		BodyText: bodyText,
		BodyHTML: bodyHTML,
	}
}
