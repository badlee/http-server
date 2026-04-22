package auth

import (
	"context"

	"github.com/dop251/goja"
	"github.com/gofiber/fiber/v3"
)

type JSModule struct{}

func (m *JSModule) Name() string {
	return "auth"
}

func (m *JSModule) Doc() string {
	return "Unified Authentication API"
}

func (m *JSModule) ToJSObject(vm *goja.Runtime) goja.Value {
	obj := vm.NewObject()
	m.Loader(nil, vm, obj)
	return obj
}

func (m *JSModule) Loader(c any, vm *goja.Runtime, moduleObject *goja.Object) {
	// CommonJS support: if exports exists, use it as the target
	module := moduleObject
	if exp := moduleObject.Get("exports"); exp != nil && !goja.IsUndefined(exp) {
		module = exp.ToObject(vm)
	}

	o := module

	o.Set("getManager", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) == 0 {
			vm.Interrupt("getManager requires a manager name")
			return goja.Undefined()
		}

		name := call.Argument(0).String()
		manager := GetManager(name)
		if manager == nil {
			return goja.Undefined()
		}

		return m.createManagerObject(vm, manager, c)
	})

	o.Set("get", o.Get("getManager"))
}

func (m *JSModule) createManagerObject(vm *goja.Runtime, manager *Manager, c any) *goja.Object {
	obj := vm.NewObject()

	// authenticate(strategyName, credsMap) -> User | null
	obj.Set("authenticate", func(call goja.FunctionCall) goja.Value {
		strategyName := ""
		if len(call.Arguments) > 0 {
			strategyName = call.Arguments[0].String()
		}

		creds := make(map[string]string)
		if len(call.Arguments) > 1 && !goja.IsUndefined(call.Arguments[1]) {
			if credsObj, ok := call.Argument(1).Export().(map[string]interface{}); ok {
				for k, v := range credsObj {
					if strVal, ok := v.(string); ok {
						creds[k] = strVal
					}
				}
			}
		}

		var ctx context.Context = context.Background()
		if fiberCtx, ok := c.(fiber.Ctx); ok {
			ctx = fiberCtx.Context()
		}

		user, err := manager.Authenticate(ctx, strategyName, creds)
		if err != nil {
			// Instead of throwing an error, we can return null to signify failed login
			return goja.Null()
		}

		return vm.ToValue(user)
	})

	// generateToken(user, expiration, issuer) -> string
	obj.Set("generateToken", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) == 0 {
			vm.Interrupt("generateToken requires a user object")
			return goja.Undefined()
		}

		userExp := call.Argument(0).Export()
		userMap, ok := userExp.(map[string]interface{})
		if !ok {
			vm.Interrupt("invalid user object")
			return goja.Undefined()
		}

		user := &User{
			ID:       getString(userMap, "id"),
			Username: getString(userMap, "username"),
			Email:    getString(userMap, "email"),
		}

		expiration := "1h"
		if len(call.Arguments) > 1 && !goja.IsUndefined(call.Arguments[1]) {
			expiration = call.Arguments[1].String()
		}

		issuer := ""
		if len(call.Arguments) > 2 && !goja.IsUndefined(call.Arguments[2]) {
			issuer = call.Arguments[2].String()
		}

		token, err := manager.GenerateToken(user, expiration, issuer)
		if err != nil {
			vm.Interrupt(err)
			return goja.Undefined()
		}

		return vm.ToValue(token)
	})

	// validateToken(tokenStr) -> User | null
	obj.Set("validateToken", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) == 0 {
			return goja.Null()
		}
		tokenStr := call.Argument(0).String()

		user, err := manager.ValidateToken(tokenStr)
		if err != nil {
			return goja.Null()
		}

		return vm.ToValue(user)
	})

	// revokeToken(jti) -> bool
	obj.Set("revokeToken", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) == 0 {
			return vm.ToValue(false)
		}
		jti := call.Argument(0).String()

		err := manager.RevokeToken(jti)
		return vm.ToValue(err == nil)
	})

	return obj
}

func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}
