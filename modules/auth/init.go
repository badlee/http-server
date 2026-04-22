package auth

import (
	"beba/modules"
	"fmt"
)

func Initialize(configs map[string]*AuthManagerConfig) error {
	for name, cfg := range configs {
		m := NewManager(name, cfg.Secret)
		
		if cfg.Server != nil {
			m.serverConfig = cfg.Server
		}

		if cfg.Database != "" {
			err := m.initDB(cfg.Database)
			if err != nil {
				fmt.Printf("Warning: failed to initialize auth database for %s: %v\n", name, err)
			}
		} else {
			m.initDB("sqlite://:memory:")
		}

		// Add strategies
		for _, sCfg := range cfg.Strategies {
			m.AddStrategy(sCfg)
		}
		// Add OAuth2 clients
		for _, cCfg := range cfg.Clients {
			m.AddClient(&OAuth2Client{
				ID:           cCfg.ID,
				Secret:       cCfg.Secret,
				RedirectURIs: cCfg.RedirectURIs,
				Scopes:       cCfg.Scopes,
			})
		}
		fmt.Printf("Auth Manager [%s] initialized with %d strategies\n", name, len(cfg.Strategies))
	}
	return nil
}

func init() {
	modules.RegisterModule(&JSModule{})
}
