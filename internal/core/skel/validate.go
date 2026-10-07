package skel

import (
	"fmt"

	"github.com/Masterminds/semver/v3"
	"go.yorun.ai/skel/descriptor"
	"go.yorun.ai/vine/buildinfo"
)

// ValidateDescriptor checks runtime invariants and derived policies. It does not
// repair data or resolve contracts from other domains, which may register later.
func ValidateDescriptor(domain *descriptor.Domain) error {
	if domain == nil || domain.Name == "" {
		return fmt.Errorf("domain descriptor requires a name")
	}
	if err := validateCompilerVersion(domain); err != nil {
		return err
	}
	for _, service := range domain.Services {
		if err := validateServiceDescriptor(service); err != nil {
			return err
		}
		if (service.Api && (service.Pub || service.Ext)) || (service.Pub && service.Ext) {
			return fmt.Errorf("service %s has conflicting modifiers", service.SkelName)
		}
		if !service.Api {
			if len(service.Audiences) > 0 || service.Require != nil {
				return fmt.Errorf("service %s: client admission rules require api; dual entry services are unsupported", service.SkelName)
			}
			for _, method := range service.Methods {
				if method.Require != nil {
					return fmt.Errorf("service %s: client admission rules require api", service.SkelName)
				}
			}
		}
	}
	for _, actor := range domain.Actors {
		if actor == nil {
			return fmt.Errorf("nil actor descriptor")
		}
		if actor.Auth != nil {
			if actor.Auth.Credential == nil || actor.Auth.Info == nil {
				return fmt.Errorf("actor %s: authentication requires credential and info descriptors", actor.SkelName)
			}
			if err := validateServiceDescriptor(actor.Auth.Service); err != nil {
				return err
			}
			if actor.Auth.Method() == nil {
				return fmt.Errorf("actor %s: unknown authentication method %q", actor.SkelName, actor.Auth.MethodName)
			}
		}
		if actor.Permission != nil {
			if err := validateServiceDescriptor(actor.Permission.Service); err != nil {
				return err
			}
			if actor.Permission.Method() == nil {
				return fmt.Errorf("actor %s: unknown permission method %q", actor.SkelName, actor.Permission.MethodName)
			}
		}
		for _, via := range actor.Vias {
			switch via {
			case descriptor.ActorViaClient, descriptor.ActorViaAgent, descriptor.ActorViaOpenAPI:
			default:
				return fmt.Errorf("actor %s: invalid transport %q", actor.SkelName, via)
			}
		}
	}
	for _, resource := range domain.Resources {
		if resource == nil {
			return fmt.Errorf("nil resource descriptor")
		}
		if resource.CheckService != nil {
			if err := validateServiceDescriptor(resource.CheckService); err != nil {
				return err
			}
		}
		checks := append([]*descriptor.ResourceCheck(nil), resource.Checks...)
		for _, action := range resource.Actions {
			if action == nil {
				return fmt.Errorf("nil resource action")
			}
			checks = append(checks, action.Checks...)
		}
		for _, check := range checks {
			if check == nil || resource.CheckMethod(check) == nil {
				return fmt.Errorf("resource %s: unresolved check method", resource.SkelName)
			}
		}
	}
	for _, web := range domain.Webs {
		if web == nil {
			return fmt.Errorf("nil web descriptor")
		}
		if !web.AuthMode.IsValid() || web.AuthMode == descriptor.AuthModeInherit {
			return fmt.Errorf("web %s: invalid auth mode %q", web.SkelName, web.AuthMode)
		}
	}
	for _, config := range domain.Configs {
		if config == nil {
			return fmt.Errorf("nil config descriptor")
		}
		switch config.Lifecycle {
		case descriptor.ConfigLifecycleEternal, descriptor.ConfigLifecycleInstant:
		default:
			return fmt.Errorf("config %s: invalid lifecycle %q", config.SkelName, config.Lifecycle)
		}
	}
	return descriptor.ValidateEffectivePolicy(domain)
}
func validateServiceDescriptor(service *descriptor.Service) error {
	if service == nil {
		return fmt.Errorf("nil service descriptor")
	}
	if !service.AuthMode.IsValid() || service.AuthMode == descriptor.AuthModeInherit || service.AuthMode == descriptor.AuthModeOff {
		return fmt.Errorf("service %s: invalid auth mode %q", service.SkelName, service.AuthMode)
	}
	names, wireNames := map[string]bool{}, map[string]bool{}
	for _, method := range service.Methods {
		if method == nil {
			return fmt.Errorf("service %s: nil method", service.SkelName)
		}
		if names[method.Name] || wireNames[method.SkelName] {
			return fmt.Errorf("service %s: duplicate method %s", service.SkelName, method.Name)
		}
		names[method.Name], wireNames[method.SkelName] = true, true
		if !method.AuthMode.IsValid() || method.AuthMode == descriptor.AuthModeOff {
			return fmt.Errorf("method %s/%s: invalid auth mode %q", service.SkelName, method.SkelName, method.AuthMode)
		}
	}
	return nil
}
func validateCompilerVersion(domain *descriptor.Domain) error {
	if domain.Generated == nil || domain.Generated.CompilerVersion == "" {
		return fmt.Errorf("domain descriptor %s missing generated compiler version; Vine requires skelc version %s or higher", domain.Name, MinSkelcVersion())
	}
	if domain.Generated.CompilerVersion == buildinfo.DevVersion {
		return nil
	}
	version, err := semver.NewVersion(domain.Generated.CompilerVersion)
	if err != nil {
		return fmt.Errorf("parse generated compiler version %s: %w", domain.Generated.CompilerVersion, err)
	}
	if version.LessThan(semver.MustParse(MinSkelcVersion())) {
		return fmt.Errorf("domain descriptor %s generated by skelc %s is lower than Vine required skelc version %s", domain.Name, version, MinSkelcVersion())
	}
	return nil
}
