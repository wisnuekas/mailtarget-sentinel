package alert

import "fmt"

type WarningEmailInput struct {
	SubAccountID   int32
	SubAccountName string
	CompanyID      int32
	Sent           uint64
	Bounced        uint64
	SpamBounced    uint64
	BounceRate     float64
	SpamRate       float64
}

func BuildWarningEmail(in WarningEmailInput) EmailContent {
	name := in.SubAccountName
	if name == "" {
		name = fmt.Sprintf("Sub-account %d", in.SubAccountID)
	}

	subject := fmt.Sprintf("[Sentinel] Warning — %s (ID %d)", name, in.SubAccountID)

	bodyText := fmt.Sprintf(
		"Mailtarget Sentinel Warning\n\n"+
			"Sub-account: %s (ID %d)\n"+
			"Company ID: %d\n"+
			"Emails sent (5m): %d\n"+
			"Bounce rate: %.2f%%\n"+
			"Spam rate: %.2f%%\n\n"+
			"This sub-account is approaching or exceeding configured thresholds. "+
			"Please review sending practices before automatic suspension.\n",
		name, in.SubAccountID, in.CompanyID,
		in.Sent, in.BounceRate, in.SpamRate,
	)

	bodyHTML := fmt.Sprintf(`<!doctype html>
<html>
<body style="font-family:system-ui,sans-serif;color:#1e293b;">
  <h2 style="color:#ea580c;">Mailtarget Sentinel — Warning</h2>
  <p>Sub-account <strong>%s</strong> (ID %d) requires attention.</p>
  <table style="border-collapse:collapse;margin:16px 0;">
    <tr><td style="padding:4px 12px 4px 0;color:#64748b;">Company</td><td>%d</td></tr>
    <tr><td style="padding:4px 12px 4px 0;color:#64748b;">Sent (5m)</td><td>%d</td></tr>
    <tr><td style="padding:4px 12px 4px 0;color:#64748b;">Bounce rate</td><td style="color:#dc2626;font-weight:600;">%.2f%%</td></tr>
    <tr><td style="padding:4px 12px 4px 0;color:#64748b;">Spam rate</td><td>%.2f%%</td></tr>
  </table>
  <p style="color:#64748b;font-size:14px;">
    Review sending practices before Sentinel triggers automatic suspension.
  </p>
</body>
</html>`,
		name, in.SubAccountID, in.CompanyID,
		in.Sent, in.BounceRate, in.SpamRate,
	)

	return EmailContent{
		Subject:  subject,
		BodyText: bodyText,
		BodyHTML: bodyHTML,
	}
}
