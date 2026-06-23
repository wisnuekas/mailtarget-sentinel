package postgres

type Company struct {
	ID     int32  `json:"id"`
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

type CompanyOwner struct {
	Email string
	Name  string
}

type SubAccount struct {
	ID           int32  `json:"id"`
	CompanyID    int32  `json:"company_id"`
	Name         string `json:"name"`
	Status       string `json:"status"`
	IPPoolID     int    `json:"ip_pool_id,omitempty"`
	IPPoolName   string `json:"ip_pool_name,omitempty"`
	CreatedAt    int64  `json:"created_at"`
	CompanyName  string `json:"company_name,omitempty"`
}

type SubAccountDetail struct {
	SubAccount
	Domains []DomainLite `json:"domains,omitempty"`
}

type DomainLite struct {
	ID        int32  `json:"id"`
	Domain    string `json:"domain"`
	IsSending bool   `json:"is_sending"`
	IsBlocked bool   `json:"is_blocked"`
}

type DomainRecord struct {
	ID           int32  `json:"id"`
	Domain       string `json:"domain"`
	CompanyID    int32  `json:"company_id"`
	SubAccountID *int32 `json:"sub_account_id,omitempty"`
	IsSending    bool   `json:"is_sending"`
	IsBlocked    bool   `json:"is_blocked"`
}

const (
	StatusActive    = "Active"
	StatusSuspended = "Suspended"
)
