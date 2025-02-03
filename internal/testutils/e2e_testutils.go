package testutils

import (
	"context"
	"log"
	"sync"
	"time"
)

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

func (s *infraSetup) Start() {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	var setupWG sync.WaitGroup
	for _, setupReg := range s.setupFuncRegistries {
		setupWG.Add(1)
		go func(name string, setupFunc InfraSetupFunc) {
			defer setupWG.Done()
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

type appSetup struct {
	setupFuncRegistries []*infraSetupFuncRegistry
	SetupDoneChan       chan struct{} // to signal setup has done
	SetupErrChan        chan struct{} // to signal setup has error
}
