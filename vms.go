package main

import (
	"context"

	"github.com/vmware/govmomi/examples"
	"github.com/vmware/govmomi/property"
	_ "github.com/vmware/govmomi/units"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25"
	_ "github.com/vmware/govmomi/vim25/methods"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

type infoResult struct {
	VirtualMachines []mo.VirtualMachine
	// objects         []*object.VirtualMachine
	entities map[types.ManagedObjectReference]string
}

var return_vms map[string]string

func getVMS() map[string]string {
	examples.Run(func(ctx context.Context, c *vim25.Client) error {
		m := view.NewManager(c)
		v, err := m.CreateContainerView(ctx, c.ServiceContent.RootFolder, []string{"VirtualMachine"}, true)
		if err != nil {
			return err
		}
		defer v.Destroy(ctx)
		var vms []mo.VirtualMachine
		err = v.Retrieve(ctx, []string{"VirtualMachine"}, []string{"summary"}, &vms)
		if err != nil {
			return err
		}
		refs := make([]types.ManagedObjectReference, 0, len(vms))
		for _, vm := range vms {
			refs = append(refs, vm.Reference())
		}
		var res infoResult
		var props []string
		pc := property.DefaultCollector(c)
		err = pc.Retrieve(ctx, refs, props, &res.VirtualMachines)
		if err != nil {
			return err
		}
		res.collectReferences(pc, ctx)

		return_vms = make(map[string]string)

		for _, vm := range vms {
			hname := res.entities[*vm.Summary.Runtime.Host]
			return_vms[vm.Summary.Config.Name] = hname

		}
		return nil
	})
	return return_vms
}
func (r *infoResult) collectReferences(pc *property.Collector, ctx context.Context) error {
	r.entities = make(map[types.ManagedObjectReference]string) // MOR -> Name map
	var host []mo.HostSystem
	vrefs := map[string]*struct {
		dest interface{}
		refs []types.ManagedObjectReference
		save func()
	}{
		"HostSystem": {
			&host, nil, func() {
				for _, e := range host {
					r.entities[e.Reference()] = e.Name
				}
			},
		},
	}
	xrefs := make(map[types.ManagedObjectReference]bool)
	addRef := func(refs ...types.ManagedObjectReference) {
		for _, ref := range refs {
			if _, exists := xrefs[ref]; exists {
				return
			}
			xrefs[ref] = true
			vref := vrefs[ref.Type]
			vref.refs = append(vref.refs, ref)
		}
	}
	for _, vm := range r.VirtualMachines {
		if ref := vm.Summary.Runtime.Host; ref != nil {
			addRef(*ref)
		}
	}
	for _, vref := range vrefs {
		if vref.refs == nil {
			continue
		}
		err := pc.Retrieve(ctx, vref.refs, []string{"name"}, vref.dest)
		if err != nil {
			return err
		}
		vref.save()
	}
	return nil
}
