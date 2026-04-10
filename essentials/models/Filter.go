package models

type FilterTransaction struct {
	FromDate  string  `bson:"-" json:"from_date"`
	ToDate    string  `bson:"-" json:"to_date"`
	MinAmount float64 `bson:"-" json:"min_amount"`
	MaxAmount float64 `bson:"-" json:"max_amount"`
	Category  string  `bson:"-" json:"category"`
	Type      string  `bson:"-" json:"type"` // "income" or "outcome"
	SortBy    string  `bson:"-" json:"sort_by"`
}
