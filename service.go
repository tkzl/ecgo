package ecgo

import "fmt"

type (
	Service struct {
		*App
		*Dao
	}
	serviceNode struct {
		services []string
		models   []string
	}
	IService interface {
		Addr() string
	}
)

func (this *Service) Addr() string {
	return fmt.Sprintf("%p", this)
}
