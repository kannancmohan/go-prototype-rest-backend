package testutils

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// Infra setup types and its methods
type InfraSetupFunc func(context.Context) (InfraSetupCleanupFunc, error)
type InfraSetupCleanupFunc func(context.Context) error

type infraSetup struct {
	setupFuncRegistries []*infraSetupFuncRegistry
	SetupDoneChan       chan struct{} // to signal setup has done
	SetupErrChan        chan struct{} // to signal setup has error
}

func NewInfraSetup(setupFuncRegistries ...*infraSetupFuncRegistry) *infraSetup {
	return &infraSetup{setupFuncRegistries: setupFuncRegistries, SetupDoneChan: make(chan struct{}), SetupErrChan: make(chan struct{})}
}

func (s *infraSetup) Start(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	var setupWG sync.WaitGroup
	for _, setupReg := range s.setupFuncRegistries {
		setupWG.Add(1)
		go func(name string, setupFunc InfraSetupFunc) {
			defer setupWG.Done()
			if err := ctx.Err(); err != nil { //skip in case ctx is cancelled/timeout
				log.Printf("Skipping setup of %s container: %v", name, err)
				close(s.SetupErrChan) // notify that there was error in setup
				return
			}
			cleanup, err := setupFunc(ctx)
			setupReg.addInfraSetupCleanupFunc(cleanup)
			if err != nil {
				log.Printf("Failed to start %s container: %v", name, err)
				cancel()              // Cancel the context so as other setup could be stopped
				close(s.SetupErrChan) // notify that there was error in setup
				return                // return from goroutine
			}
			log.Printf("Successfully started %s container", name)
		}(setupReg.name, setupReg.setupFunc)
	}
	setupWG.Wait()
	close(s.SetupDoneChan) // notify that setup has been done
}

func (s *infraSetup) Cleanup(ctx context.Context) {
	log.Print("Invoking cleanup...")
	var cleanupWG sync.WaitGroup
	for _, setupReg := range s.setupFuncRegistries {
		cleanupWG.Add(1)
		go func(setupReg *infraSetupFuncRegistry) {
			defer cleanupWG.Done()
			if setupReg.cleanupFunc != nil {
				if err := setupReg.cleanupFunc(ctx); err != nil {
					log.Printf("Failed to cleanup %s: %v", setupReg.name, err)
					return // return from goroutine
				}
				log.Printf("Successfully cleaned up %s", setupReg.name)
			}
		}(setupReg)
	}
	cleanupWG.Wait()
}

type infraSetupFuncRegistry struct {
	name        string
	setupFunc   InfraSetupFunc
	cleanupFunc InfraSetupCleanupFunc
}

func NewInfraSetupFuncRegistry(name string, setupFunc InfraSetupFunc) *infraSetupFuncRegistry {
	return &infraSetupFuncRegistry{
		name:      name,
		setupFunc: setupFunc,
	}
}

func (s *infraSetupFuncRegistry) addInfraSetupCleanupFunc(cleanupFunc InfraSetupCleanupFunc) {
	s.cleanupFunc = cleanupFunc
}

// App setup types and its methods
type AppSetupFuncResponse struct {
	Addr string
}
type AppSetupFunc func(context.Context) (AppSetupFuncResponse, error)
type appSetup struct {
	appSetupFuncRegistries map[string]*appSetupFuncRegistry
	DoneChan               chan struct{} // to signal app has started
	ErrChan                chan struct{} // to signal app has error
}

func NewAppSetup() *appSetup {
	return &appSetup{appSetupFuncRegistries: make(map[string]*appSetupFuncRegistry), DoneChan: make(chan struct{}), ErrChan: make(chan struct{})}
}

func (s *appSetup) AddSetupFunc(appName string, appSetupFunc AppSetupFunc) *appSetup {
	if s.appSetupFuncRegistries == nil {
		s.appSetupFuncRegistries = make(map[string]*appSetupFuncRegistry)
	}
	s.appSetupFuncRegistries[appName] = &appSetupFuncRegistry{setupFunc: appSetupFunc}
	return s
}

func (s *appSetup) GetAppSetupFunResponse(appName string) (AppSetupFuncResponse, error) {
	val, ok := s.appSetupFuncRegistries[appName]
	if !ok {
		return AppSetupFuncResponse{}, fmt.Errorf("AppSetupFuncResponse not found for appName:%s", appName)
	}
	return val.response, nil
}

func (s *appSetup) Start(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	var setupWG sync.WaitGroup
	for name, setupReg := range s.appSetupFuncRegistries {
		setupWG.Add(1)
		go func(name string, setupFunc AppSetupFunc) {
			defer setupWG.Done()
			if err := ctx.Err(); err != nil { //skip in case ctx is cancelled/timeout
				log.Printf("Skipping setup of %s app: %v", name, err)
				close(s.ErrChan) // notify that there was error in setup
				return
			}
			resp, err := setupFunc(ctx)
			if err != nil {
				log.Printf("Failed to start %s app: %v", name, err)
				cancel()         // Cancel the context so as other setup could be stopped
				close(s.ErrChan) // notify that there was error in setup
				return           // return from goroutine
			}
			setupReg.addAppSetupFuncResponse(resp)
			log.Printf("Successfully started %s app", name)
		}(name, setupReg.setupFunc)
	}
	setupWG.Wait()
	close(s.DoneChan) // notify that all apps are started
}

type appSetupFuncRegistry struct {
	setupFunc AppSetupFunc
	response  AppSetupFuncResponse
}

func (s *appSetupFuncRegistry) addAppSetupFuncResponse(response AppSetupFuncResponse) {
	s.response = response
}
