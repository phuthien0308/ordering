package account

import (
	"context"
	"encoding/json"

	dto "github.com/phuthien0308/ordering/accountservice/repository/dto"
	"github.com/redis/go-redis/v9"
)

type AccountRepository interface {
	Save(ctx context.Context, acc dto.Account) (string, error)
	Load(ctx context.Context, accountId string) (*dto.Account, error)
	Search(ctx context.Context, term string) ([]*dto.Account, error)
}

type accountRepository struct {
	redisClient redis.Client
}

func (rp *accountRepository) Save(ctx context.Context, acc dto.Account) (*string, error) {
	data, err := json.Marshal(acc)
	if err != nil {
		return nil, err
	}
	cmd := rp.redisClient.Set(ctx, acc.Id, data, 0)
	_, err = cmd.Result()
	if err != nil {
		return nil, err
	}
	return &acc.Id, nil
}

func (rp *accountRepository) Load(ctx context.Context, accountId string) (*dto.Account, error) {
	cmd := rp.redisClient.Get(ctx, accountId)
	bytes, err := cmd.Bytes()
	if err != nil {
		return nil, err
	}
	account := &dto.Account{}
	err = json.Unmarshal(bytes, account)
	if err != nil {
		return nil, err
	}
	return account, nil
}
