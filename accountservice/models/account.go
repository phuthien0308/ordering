package models

import "github.com/google/uuid"

type Account struct {
	Id      string
	Name    string
	Address []string
}

func NewAccount(name string, address []string) (*Account, error) {
	idGenerator, err := uuid.NewV6()
	if err != nil {
		return nil, err
	}
	return &Account{
		Id:      idGenerator.String(),
		Name:    name,
		Address: address,
	}, nil
}
