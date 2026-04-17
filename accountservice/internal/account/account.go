package account

import "github.com/google/uuid"

type Account struct {
	Id       uuid.UUID
	FullName string
	Address  []Address
}

type Address struct {
	IsPrimary bool
	Address   string
}
