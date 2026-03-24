package models

type Report struct {
	Month        string  `json:"month"`
	TotalIncome  float64 `json:"total_income"`
	TotalOutcome float64 `json:"total_outcome"`
	Net          float64 `json:"net"`
}
