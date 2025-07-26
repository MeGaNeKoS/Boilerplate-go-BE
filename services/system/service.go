package system

import (
	"context"
	"fmt"
	"os"
	"time"

	"project-template/pkg/code"
)

type ServiceImpl interface {
	Echo(ctx context.Context) (interface{}, *code.Code)
	Crash(ctx context.Context)
	Long(ctx context.Context, sleepSeconds int) (interface{}, *code.Code)
}

type Service struct{}

func NewService() ServiceImpl {
	return &Service{}
}

func (s *Service) Echo(ctx context.Context) (interface{}, *code.Code) {
	return "echo", nil
}

func (s *Service) Crash(ctx context.Context) {
	panic("intentional crash")
}

func (s *Service) Long(ctx context.Context, sleepSeconds int) (interface{}, *code.Code) {
	if sleepSeconds > 0 {
		time.Sleep(time.Duration(sleepSeconds) * time.Second)
	}
	return fmt.Sprintf("%d", os.Getpid()), nil
}
