package provisioner

import (
	"errors"
	"fmt"
	"github.com/GabeCordo/mango-go/processor/components/module"
	"github.com/GabeCordo/mango/components/cluster"
	"io/fs"
	"os"
	"path/filepath"
)

func NewProvisioner() *Provisioner {
	provisioner := new(Provisioner)

	provisioner.modules = make(map[string]*ModuleWrapper)

	defaultFrameworkModule := NewModuleWrapper()
	defaultFrameworkModule.Identifier = DefaultFrameworkModule
	defaultFrameworkModule.Version = 1.0
	defaultFrameworkModule.Mount()
	provisioner.modules[DefaultFrameworkModule] = defaultFrameworkModule

	return provisioner
}

func (provisioner *Provisioner) ModuleExists(moduleName string) bool {
	provisioner.mutex.RLock()
	defer provisioner.mutex.RUnlock()

	_, found := provisioner.modules[moduleName]
	return found
}

func (provisioner *Provisioner) GetModules() []*ModuleWrapper {

	provisioner.mutex.RLock()
	defer provisioner.mutex.RUnlock()

	modules := make([]*ModuleWrapper, 0)
	for _, moduleWrapper := range provisioner.modules {
		modules = append(modules, moduleWrapper)
	}

	return modules
}

func (provisioner *Provisioner) GetModule(moduleName string) (instance *ModuleWrapper, found bool) {
	provisioner.mutex.RLock()
	defer provisioner.mutex.RUnlock()

	instance, found = provisioner.modules[moduleName]
	if !found {
		return nil, false
	}

	if instance.MarkForDeletion {
		return nil, false
	}

	return instance, found
}

func (provisioner *Provisioner) AddModule(identifier string) error {

	provisioner.mutex.Lock()
	defer provisioner.mutex.Unlock()

	if _, found := provisioner.modules[identifier]; found {
		return errors.New("module already exists")
	}

	mod := NewModuleWrapper()
	mod.Mounted = true
	mod.Identifier = identifier
	provisioner.modules[identifier] = mod

	return nil
}

func (provisioner *Provisioner) DeleteModule(identifier string) (deleted, markedForDeletion, found bool) {

	provisioner.mutex.Lock()
	defer provisioner.mutex.Unlock()

	deleted = false

	if moduleWrapper, foundModule := provisioner.modules[identifier]; foundModule {
		found = true

		provisioner.modules[identifier].MarkForDeletion = true
		markedForDeletion = true

		if moduleWrapper.CanDelete() {
			delete(provisioner.modules, identifier)
			deleted = true
			//log.Printf("[provisioner] module %s deleted\n", identifier)
		} else {
			//log.Printf("[provisioner] could not delete %s\n", identifier)
		}
	} else {
		//log.Printf("[provisioner] could not find %s\n", identifier)
		found = false
	}

	return deleted, markedForDeletion, found
}

func (provisioner *Provisioner) InjectModules(folder string) error {

	if _, err := os.Stat(folder); err != nil {
		return err
	}

	filepath.WalkDir(folder, func(path string, d fs.DirEntry, err error) error {

		if path != folder {
			return nil
		}

		if !d.IsDir() {
			return nil
		}

		_, err = provisioner.InjectModuleFromPath(path)
		return err
	})

	return nil
}

func (provisioner *Provisioner) InjectModuleFromPath(folder string) (*ModuleWrapper, error) {

	remote, err := module.NewRemoteModule(folder)
	if err != nil {
		return nil, err
	}

	local, err := remote.Get()
	if err != nil {
		return nil, err
	}

	return provisioner.InjectModule(local)
}

func (provisioner *Provisioner) InjectModule(implementation *module.Module) (*ModuleWrapper, error) {

	provisioner.mutex.Lock()
	defer provisioner.mutex.Unlock()

	if implementation == nil {
		return nil, errors.New("implementation got nil but expected type *module.Module")
	}

	if _, found := provisioner.modules[implementation.Config.Name]; found {
		return nil, errors.New("module with identifier already exists")
	}

	fmt.Printf("initializing module %s\n", implementation.Config.Name)

	// return a pointer to a ModuleWrapper
	moduleWrapper := NewModuleWrapper()
	// store the pointer to the ModuleWrapper in the provisioner modules map
	provisioner.modules[implementation.Config.Name] = moduleWrapper

	moduleWrapper.Version = implementation.Config.Version
	moduleWrapper.Identifier = implementation.Config.Name

	// if this is a standalone mode, there is no support for mounting
	// -> in standalone, it will always be set to true
	// TODO : change
	moduleWrapper.Mount()

	// iterate over cluster that is stored in the module's common
	for _, export := range implementation.Config.Exports {

		// for every cluster that is defined in the common, there should be a 1:1 mapping
		// of an implementation in the go plugin in a var of the same name. Try to find
		// this variable in the go plugin
		f, err := implementation.Plugin.Lookup(export.Cluster)
		if err != nil {
			// the cluster is missing a 1:1 mapping
			continue
		}

		// the incoming struct must implement the cluster.Cluster interface
		clusterImplementation, ok := (f).(cluster.Cluster)
		if !ok {
			continue
		}

		_, implementsLoadOne := (f).(cluster.LoadOne)

		_, implementsLoadAll := (f).(cluster.LoadAll)

		// the cluster must implement either LoadOne or LoadAll interfaces
		if !implementsLoadOne && !implementsLoadAll {
			continue
		}

		defaultConfig := &cluster.Config{
			Identifier:                  export.Cluster,
			OnLoad:                      export.Config.OnLoad,
			OnCrash:                     export.Config.OnCrash,
			StartWithNTransformClusters: export.Config.Static.TFunctions,
			StartWithNLoadClusters:      export.Config.Static.LFunctions,
			ETChannelThreshold:          export.Config.Dynamic.TFunction.Threshold,
			ETChannelGrowthFactor:       export.Config.Dynamic.TFunction.GrowthFactor,
			TLChannelThreshold:          export.Config.Dynamic.LFunction.Threshold,
			TLChannelGrowthFactor:       export.Config.Dynamic.LFunction.GrowthFactor,
		}

		clusterWrapper, err := moduleWrapper.AddCluster(export.Cluster, export.Config.Mode, clusterImplementation, defaultConfig)
		if err != nil {
			continue
		}

		// the common specifies whether they want the cluster to be mounted on load
		clusterWrapper.Mounted = export.StaticMount

		// if this is in standalone mode, there is no support for mounting
		// -> the cluster mount will always be set to true
		// TODO : change
		clusterWrapper.Mount()

		fmt.Printf("registering cluster(%s) to module(%s)\n", export.Cluster, implementation.Config.Name)
	}

	return moduleWrapper, nil
}
