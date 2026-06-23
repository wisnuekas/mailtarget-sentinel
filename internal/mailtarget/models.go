package mailtarget

// Models derived from transmission-openapi-spec.json and apiconfig-openapi-spec.json.

const (
	StatusActive     = "Active"
	StatusSuspended  = "Suspended"
	StatusTerminated = "Terminated"
)

type Address struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

type OptionsAttributes struct {
	ClickTracking bool `json:"clickTracking"`
	OpenTracking  bool `json:"openTracking"`
	Transactional bool `json:"transactional"`
}

type TransmissionForm struct {
	Subject           string             `json:"subject"`
	From              Address            `json:"from"`
	To                []Address          `json:"to"`
	CC                []Address          `json:"cc,omitempty"`
	BodyText          string             `json:"bodyText,omitempty"`
	BodyHTML          string             `json:"bodyHtml,omitempty"`
	Metadata          map[string]string  `json:"metadata,omitempty"`
	OptionsAttributes *OptionsAttributes `json:"optionsAttributes,omitempty"`
}

type TransmissionData struct {
	TransmissionID string `json:"transmissionId"`
}

type UpdateSubAccountForm struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	IPPoolID int    `json:"ipPoolId"`
}

type SubAccountSummary struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	CreatedAt int64  `json:"createdAt"`
}

type SubAccountListResponse struct {
	Count       int                 `json:"count"`
	SubAccounts []SubAccountSummary `json:"subAccounts"`
}

type GetSubAccountResponse struct {
	ID             int    `json:"id"`
	CompanyID      int    `json:"companyId"`
	Status         string `json:"status"`
	IPPoolID       int    `json:"ipPoolId"`
	IPPoolName     string `json:"ipPoolName"`
	SubAccountName string `json:"subAccountName"`
	CreatedAt      int64  `json:"createdAt"`
}

type ErrorResponse struct {
	Message string `json:"message"`
	Error   string `json:"error"`
}
