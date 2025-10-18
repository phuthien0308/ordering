package model

import "time"

type Account struct {
	Id        string
	FirstName string
	LastName  string
	Addresses []Address
}

type Address struct {
	Id        string
	Addr      string
	IsPrimary bool
}

type Metadata struct {
	CreatedBy string
	CreatedAt time.Time
	UpdatedBy string
	UpdatedAt time.Time
}

type Order struct {
	Metadata
	Id           string          `json:"id"`
	Items        []OrderItem     `json:"items"`
	ShippingInfo ShippingAddress `json:"shipping_info"`
}

type OrderItem struct {
	Id       string  `json:"id"`
	Price    float64 `json:"price"`
	Quantity int8    `json:"quantity"`
}

type ShippingAddress struct {
	Address     string `json:"address"`
	PhoneNumber string `json:"phone_number"`
}
