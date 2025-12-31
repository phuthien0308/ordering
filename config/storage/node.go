package storage

import (
	"context"
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/phuthien0308/ordering/config/dto"
	"github.com/redis/go-redis/v9"
)

const SERVICE_ADDRESSES = "service_addresses"

type AddressStorage interface {
	Add(ctx context.Context, appName, ip string) error
	Remove(ctx context.Context, appName, ip string) error
	GetAddresses(ctx context.Context, appName string) ([]string, error)
	GetAddressOfAllServices(ctx context.Context) ([]dto.Node, error)
}

func NewAddressStorage(rd *redis.Client) AddressStorage {
	return &addressStorage{
		rd: rd,
	}
}

type addressStorage struct {
	rd       *redis.Client
	s3Client *s3.Client
}

// GetAddresses implements AddressStorage.
func (n *addressStorage) GetAddresses(ctx context.Context, appName string) ([]string, error) {
	return n.rd.SMembers(ctx, appName).Result()
}

// GetIps implements NodeStorage.
func (n *addressStorage) GetAddressOfAllServices(ctx context.Context) ([]dto.Node, error) {
	allServiceKeys, err := n.rd.Keys(ctx, fmt.Sprintf("%v:*", SERVICE_ADDRESSES)).Result()
	if err != nil {
		return nil, err
	}
	wg := sync.WaitGroup{}
	wg.Add(len(allServiceKeys))

	dChan := make(chan dto.Node)
	for _, serviceKey := range allServiceKeys {
		go func() {
			defer wg.Done()
			data, err := n.rd.SMembers(ctx, serviceKey).Result()
			if err != nil {
				return
			} else {
				newNode := dto.Node{
					AppName: serviceKey,
					Ips:     data,
				}
				dChan <- newNode
				return
			}
		}()
	}

	go func() {
		wg.Wait()
		close(dChan)
	}()

	result := []dto.Node{}
	for re := range dChan {
		result = append(result, re)
	}

	return result, nil
}

// Remove implements NodeStorage.
func (n *addressStorage) Remove(ctx context.Context, appName string, ip string) error {
	_, err := n.rd.SRem(ctx, appName, ip).Result()
	if err != nil {
		return err
	}
	return nil
}

func (n *addressStorage) Add(ctx context.Context, appName, ip string) error {
	_, err := n.rd.SAdd(ctx, appName, ip).Result()
	if err != nil {
		return err
	}
	return nil
}
