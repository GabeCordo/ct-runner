package http

import (
	"context"
	"errors"
	"github.com/GabeCordo/mango-go/processor/threads/common"
	"github.com/GabeCordo/mango/threads"
	"github.com/GabeCordo/mango/utils"
	"net/http"
	"sync"
)

// Frontend Thread

type Thread struct {
	Interrupt chan<- threads.InterruptEvent // Upon completion or failure an interrupt can be raised

	C1 chan<- common.ProvisionerRequest  // Core is sending threads to the Database
	C2 <-chan common.ProvisionerResponse // Core is receiving responses from the Database

	ProvisionerResponseTable *utils.ResponseTable

	server    *http.Server
	mux       *http.ServeMux
	cancelCtx context.CancelFunc

	logger *utils.Logger

	accepting bool
	counter   uint32
	mutex     sync.Mutex
	wg        sync.WaitGroup
}

func NewThread(logger *utils.Logger, channels ...interface{}) (*Thread, error) {
	core := new(Thread)

	var ok bool

	core.Interrupt, ok = (channels[0]).(chan threads.InterruptEvent)
	if !ok {
		return nil, errors.New("expected type 'chan InterruptEvent' in index 0")
	}
	core.C1, ok = (channels[1]).(chan common.ProvisionerRequest)
	if !ok {
		return nil, errors.New("expected type 'chan ProvisionerRequest' in index 1")
	}
	core.C2, ok = (channels[2]).(chan common.ProvisionerResponse)
	if !ok {
		return nil, errors.New("expected type 'chan ProvisionerResponse' in index 2")
	}

	core.server = new(http.Server)

	core.accepting = true
	core.counter = 0

	if logger == nil {
		return nil, errors.New("expected non nil *utils.Logger type")
	}
	core.logger = logger
	core.logger.SetColour(utils.Green)

	return core, nil
}
